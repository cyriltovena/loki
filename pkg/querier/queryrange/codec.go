package queryrange

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/cortexproject/cortex/pkg/ingester/client"
	"github.com/cortexproject/cortex/pkg/querier/queryrange"
	"github.com/grafana/loki/pkg/loghttp"
	"github.com/grafana/loki/pkg/logql/marshal"
	json "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/httpgrpc"
)

var lokiCodec = &codec{}

type codec struct{}

func (r *LokiRequest) GetEnd() int64 {
	return r.EndTs.UnixNano() / 1e6
}

func (r *LokiRequest) GetStart() int64 {
	return r.StartTs.UnixNano() / 1e6
}

func (r *LokiRequest) WithStartEnd(s int64, e int64) queryrange.Request {
	new := *r
	r.StartTs = time.Unix(0, s*int64(time.Millisecond))
	r.EndTs = time.Unix(0, e*int64(time.Millisecond))
	return &new
}

func (codec) DecodeRequest(_ context.Context, r *http.Request) (queryrange.Request, error) {
	req, err := loghttp.ParseRangeQuery(r)
	if err != nil {
		return nil, err
	}
	return &LokiRequest{
		Query:     req.Query,
		Limit:     req.Limit,
		Direction: req.Direction,
		StartTs:   req.Start,
		EndTs:     req.End,
		Step:      int64(req.Step / time.Millisecond),
		Path:      r.URL.Path,
	}, nil
}

func (codec) EncodeRequest(ctx context.Context, r queryrange.Request) (*http.Request, error) {
	lokiReq, ok := r.(*LokiRequest)
	if !ok {
		return nil, httpgrpc.Errorf(http.StatusInternalServerError, "invalid request format")
	}
	params := url.Values{
		"start": []string{fmt.Sprintf("%d", lokiReq.StartTs.UnixNano())},
		"end":   []string{fmt.Sprintf("%d", lokiReq.EndTs.UnixNano())},
		// waiting for https://github.com/grafana/loki/pull/1211 we should support float or duration.
		"step":      []string{fmt.Sprintf("%d", lokiReq.Step/int64(time.Second/time.Millisecond))},
		"query":     []string{lokiReq.Query},
		"direction": []string{lokiReq.Direction.String()},
		"limit":     []string{fmt.Sprintf("%d", lokiReq.Limit)},
	}
	u := &url.URL{
		Path:     lokiReq.Path,
		RawQuery: params.Encode(),
	}
	req := &http.Request{
		Method:     "GET",
		RequestURI: u.String(), // This is what the httpgrpc code looks at.
		URL:        u,
		Body:       http.NoBody,
		Header:     http.Header{},
	}

	return req.WithContext(ctx), nil
}

func (codec) DecodeResponse(ctx context.Context, r *http.Response) (queryrange.Response, error) {
	if r.StatusCode/100 != 2 {
		body, _ := ioutil.ReadAll(r.Body)
		return nil, httpgrpc.Errorf(r.StatusCode, string(body))
	}

	sp, _ := opentracing.StartSpanFromContext(ctx, "DecodeResponse")
	defer sp.Finish()

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sp.LogFields(otlog.Error(err))
		return nil, httpgrpc.Errorf(http.StatusInternalServerError, "error decoding response: %v", err)
	}

	sp.LogFields(otlog.Int("bytes", len(buf)))

	var resp loghttp.QueryResponse
	if err := json.Unmarshal(buf, &resp); err != nil {
		return nil, httpgrpc.Errorf(http.StatusInternalServerError, "error decoding response: %v", err)
	}
	if resp.Status != loghttp.QueryStatusSuccess {
		return nil, httpgrpc.Errorf(http.StatusInternalServerError, "error executing request: %v", resp.Status)
	}
	switch string(resp.Data.ResultType) {
	case loghttp.ResultTypeMatrix:
		return &queryrange.PrometheusResponse{
			Status: loghttp.QueryStatusSuccess,
			Data: queryrange.PrometheusData{
				ResultType: loghttp.ResultTypeMatrix,
				Result:     toProto(resp.Data.Result.(loghttp.Matrix)),
			},
		}, nil
	case loghttp.ResultTypeStream:
		return &LokiResponse{
			Status: loghttp.QueryStatusSuccess,
			Data: LokiData{
				ResultType: loghttp.ResultTypeStream,
				Result:     resp.Data.Result.(loghttp.Streams).ToProto(),
			},
		}, nil
	default:
		return nil, httpgrpc.Errorf(http.StatusBadRequest, "unsupported response type")
	}
}

func (codec) EncodeResponse(ctx context.Context, res queryrange.Response) (*http.Response, error) {
	sp, _ := opentracing.StartSpanFromContext(ctx, "APIResponse.ToHTTPResponse")
	defer sp.Finish()

	if _, ok := res.(*queryrange.PrometheusResponse); ok {
		return queryrange.PrometheusCodec.EncodeResponse(ctx, res)
	}

	proto, ok := res.(*LokiResponse)
	if !ok {
		return nil, httpgrpc.Errorf(http.StatusInternalServerError, "invalid response format")
	}

	streams := make(loghttp.Streams, len(proto.Data.Result))

	for i, stream := range proto.Data.Result {
		s, err := marshal.NewStream(&stream)
		if err != nil {
			return nil, err
		}
		streams[i] = s
	}

	queryRes := loghttp.QueryResponse{
		Status: proto.Status,
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultType(proto.Data.ResultType),
			Result:     streams,
		},
	}

	b, err := json.Marshal(queryRes)
	if err != nil {
		return nil, err
	}

	sp.LogFields(otlog.Int("bytes", len(b)))

	resp := http.Response{
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body:       ioutil.NopCloser(bytes.NewBuffer(b)),
		StatusCode: http.StatusOK,
	}
	return &resp, nil
}

func (codec) MergeResponse(responses ...queryrange.Response) (queryrange.Response, error) {
	// todo we might want to cast this correctly. not sure about the impact yet.
	if len(responses) == 0 {
		return &queryrange.PrometheusResponse{
			Status: loghttp.QueryStatusSuccess,
		}, nil
	}
	if _, ok := responses[0].(*queryrange.PrometheusResponse); ok {
		return queryrange.PrometheusCodec.MergeResponse(responses...)
	}
	protos := make([]*LokiResponse, 0, len(responses))
	for _, p := range responses {
		protos = append(protos, p.(*LokiResponse))
	}
	// merge responses.
	return nil, nil
}

func toProto(m loghttp.Matrix) []queryrange.SampleStream {
	if len(m) == 0 {
		return nil
	}
	res := make([]queryrange.SampleStream, 0, len(m))
	for _, stream := range m {
		samples := make([]client.Sample, 0, len(stream.Values))
		for _, s := range stream.Values {
			samples = append(samples, client.Sample{
				Value:       float64(s.Value),
				TimestampMs: int64(s.Timestamp),
			})
		}
		res = append(res, queryrange.SampleStream{
			Labels:  client.FromMetricsToLabelAdapters(stream.Metric),
			Samples: samples,
		})
	}
	return res
}
