package queryrange

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	strings "strings"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/ingester/client"
	"github.com/cortexproject/cortex/pkg/querier/queryrange"
	"github.com/grafana/loki/pkg/loghttp"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/stretchr/testify/require"
)

func init() {
	time.Local = nil // for easier tests comparison
}

var (
	start = testTime //  Marshalling the time drops the monotonic clock so we can't use time.Now
	end   = start.Add(1 * time.Hour)
)

func Test_codec_DecodeRequest(t *testing.T) {
	tests := []struct {
		name       string
		reqBuilder func() (*http.Request, error)
		want       queryrange.Request
		wantErr    bool
	}{
		{"wrong", func() (*http.Request, error) { return http.NewRequest(http.MethodGet, "/bad?step=bad", nil) }, nil, true},
		{"ok", func() (*http.Request, error) {
			return http.NewRequest(http.MethodGet,
				fmt.Sprintf(`/query_range?start=%d&end=%d&query={foo="bar"}&step=1&limit=200&direction=FORWARD`, start.UnixNano(), end.UnixNano()), nil)
		}, &LokiRequest{
			Query:     `{foo="bar"}`,
			Limit:     200,
			Step:      1000, // step is expected in ms.
			Direction: logproto.FORWARD,
			Path:      "/query_range",
			StartTs:   start,
			EndTs:     end,
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := tt.reqBuilder()
			if err != nil {
				t.Fatal(err)
			}
			got, err := lokiCodec.DecodeRequest(context.TODO(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("codec.DecodeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, got, tt.want)
		})
	}
}

func Test_codec_DecodeResponse(t *testing.T) {
	tests := []struct {
		name    string
		res     *http.Response
		req     queryrange.Request
		want    queryrange.Response
		wantErr bool
	}{
		{"500", &http.Response{StatusCode: 500, Body: ioutil.NopCloser(strings.NewReader("some error"))}, nil, nil, true},
		{"no body", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(badReader{})}, nil, nil, true},
		{"bad json", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(""))}, nil, nil, true},
		{"not success", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(`{"status":"fail"}`))}, nil, nil, true},
		{"unknown", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(`{"status":"success"}`))}, nil, nil, true},
		{"matrix", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(matrixString))}, nil,
			&queryrange.PrometheusResponse{
				Status: loghttp.QueryStatusSuccess,
				Data: queryrange.PrometheusData{
					ResultType: loghttp.ResultTypeMatrix,
					Result:     sampleStreams,
				},
			}, false},
		{"streams v1", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(streamsString))},
			&LokiRequest{Direction: logproto.FORWARD, Limit: 100, Path: "/loki/api/v1/query_range"},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     100,
				Version:   uint32(loghttp.VersionV1),
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result:     logStreams,
				},
			}, false},
		{"streams legacy", &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(streamsString))},
			&LokiRequest{Direction: logproto.FORWARD, Limit: 100, Path: "/api/prom/query_range"},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     100,
				Version:   uint32(loghttp.VersionLegacy),
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result:     logStreams,
				},
			}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lokiCodec.DecodeResponse(context.TODO(), tt.res, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("codec.DecodeResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_codec_EncodeRequest(t *testing.T) {

	// we only accept LokiRequest.
	got, err := lokiCodec.EncodeRequest(context.TODO(), &queryrange.PrometheusRequest{})
	require.Error(t, err)
	require.Nil(t, got)

	ctx := context.Background()
	toEncode := &LokiRequest{
		Query:     `{foo="bar"}`,
		Limit:     200,
		Step:      1010,
		Direction: logproto.FORWARD,
		Path:      "/query_range",
		StartTs:   start,
		EndTs:     end,
	}
	got, err = lokiCodec.EncodeRequest(ctx, toEncode)
	require.NoError(t, err)
	require.Equal(t, ctx, got.Context())
	require.Equal(t, "/loki/api/v1/query_range", got.URL.Path)
	require.Equal(t, fmt.Sprintf("%d", start.UnixNano()), got.URL.Query().Get("start"))
	require.Equal(t, fmt.Sprintf("%d", end.UnixNano()), got.URL.Query().Get("end"))
	require.Equal(t, `{foo="bar"}`, got.URL.Query().Get("query"))
	require.Equal(t, fmt.Sprintf("%d", 200), got.URL.Query().Get("limit"))
	require.Equal(t, `FORWARD`, got.URL.Query().Get("direction"))
	require.Equal(t, "1.010000", got.URL.Query().Get("step"))

	// testing a full roundtrip
	req, err := lokiCodec.DecodeRequest(context.TODO(), got)
	require.NoError(t, err)
	require.Equal(t, toEncode.Query, req.(*LokiRequest).Query)
	require.Equal(t, toEncode.Step, req.(*LokiRequest).Step)
	require.Equal(t, toEncode.StartTs, req.(*LokiRequest).StartTs)
	require.Equal(t, toEncode.EndTs, req.(*LokiRequest).EndTs)
	require.Equal(t, toEncode.Direction, req.(*LokiRequest).Direction)
	require.Equal(t, toEncode.Limit, req.(*LokiRequest).Limit)
	require.Equal(t, "/loki/api/v1/query_range", req.(*LokiRequest).Path)
}

func Test_codec_EncodeResponse(t *testing.T) {

	tests := []struct {
		name    string
		res     queryrange.Response
		body    string
		wantErr bool
	}{
		{"error", &badResponse{}, "", true},
		{"prom", &queryrange.PrometheusResponse{
			Status: loghttp.QueryStatusSuccess,
			Data: queryrange.PrometheusData{
				ResultType: loghttp.ResultTypeMatrix,
				Result:     sampleStreams,
			},
		}, matrixString, false},
		{"loki v1",
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     100,
				Version:   uint32(loghttp.VersionV1),
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result:     logStreams,
				},
			}, streamsString, false},
		{"loki legacy",
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     100,
				Version:   uint32(loghttp.VersionLegacy),
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result:     logStreams,
				},
			}, streamsStringLegacy, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lokiCodec.EncodeResponse(context.TODO(), tt.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("codec.EncodeResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				require.Equal(t, 200, got.StatusCode)
				body, err := ioutil.ReadAll(got.Body)
				require.Nil(t, err)
				bodyString := string(body)
				require.JSONEq(t, tt.body, bodyString)
			}
		})
	}
}

func Test_codec_MergeResponse(t *testing.T) {
	tests := []struct {
		name      string
		responses []queryrange.Response
		want      queryrange.Response
		wantErr   bool
	}{
		{"empty", []queryrange.Response{}, nil, true},
		{"unknown response", []queryrange.Response{&badResponse{}}, nil, true},
		{"prom", []queryrange.Response{
			&queryrange.PrometheusResponse{
				Status: loghttp.QueryStatusSuccess,
				Data: queryrange.PrometheusData{
					ResultType: loghttp.ResultTypeMatrix,
					Result:     sampleStreams,
				},
			}},
			&queryrange.PrometheusResponse{
				Status: loghttp.QueryStatusSuccess,
				Data: queryrange.PrometheusData{
					ResultType: loghttp.ResultTypeMatrix,
					Result:     sampleStreams,
				},
			},
			false,
		},
		{
			"loki backward",
			[]queryrange.Response{
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.BACKWARD,
					Limit:     100,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 2), Line: "2"},
									{Timestamp: time.Unix(0, 1), Line: "1"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 6), Line: "6"},
									{Timestamp: time.Unix(0, 5), Line: "5"},
								},
							},
						},
					},
				},
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.BACKWARD,
					Limit:     100,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 10), Line: "10"},
									{Timestamp: time.Unix(0, 9), Line: "9"},
									{Timestamp: time.Unix(0, 9), Line: "9"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 16), Line: "16"},
									{Timestamp: time.Unix(0, 15), Line: "15"},
								},
							},
						},
					},
				},
			},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.BACKWARD,
				Limit:     100,
				Version:   1,
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result: []logproto.Stream{
						{
							Labels: `{foo="bar", level="debug"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 16), Line: "16"},
								{Timestamp: time.Unix(0, 15), Line: "15"},
								{Timestamp: time.Unix(0, 6), Line: "6"},
								{Timestamp: time.Unix(0, 5), Line: "5"},
							},
						},
						{
							Labels: `{foo="bar", level="error"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 10), Line: "10"},
								{Timestamp: time.Unix(0, 9), Line: "9"},
								{Timestamp: time.Unix(0, 9), Line: "9"},
								{Timestamp: time.Unix(0, 2), Line: "2"},
								{Timestamp: time.Unix(0, 1), Line: "1"},
							},
						},
					},
				},
			},
			false,
		},
		{
			"loki backward limited",
			[]queryrange.Response{
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.BACKWARD,
					Limit:     4,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 2), Line: "2"},
									{Timestamp: time.Unix(0, 1), Line: "1"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 6), Line: "6"},
									{Timestamp: time.Unix(0, 5), Line: "5"},
								},
							},
						},
					},
				},
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.BACKWARD,
					Limit:     4,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 10), Line: "10"},
									{Timestamp: time.Unix(0, 9), Line: "9"},
									{Timestamp: time.Unix(0, 9), Line: "9"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 16), Line: "16"},
									{Timestamp: time.Unix(0, 15), Line: "15"},
								},
							},
						},
					},
				},
			},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.BACKWARD,
				Limit:     4,
				Version:   1,
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result: []logproto.Stream{
						{
							Labels: `{foo="bar", level="debug"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 16), Line: "16"},
								{Timestamp: time.Unix(0, 15), Line: "15"},
							},
						},
						{
							Labels: `{foo="bar", level="error"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 10), Line: "10"},
								{Timestamp: time.Unix(0, 9), Line: "9"},
							},
						},
					},
				},
			},
			false,
		},
		{
			"loki forward",
			[]queryrange.Response{
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.FORWARD,
					Limit:     100,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 1), Line: "1"},
									{Timestamp: time.Unix(0, 2), Line: "2"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 5), Line: "5"},
									{Timestamp: time.Unix(0, 6), Line: "6"},
								},
							},
						},
					},
				},
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.FORWARD,
					Limit:     100,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 9), Line: "9"},
									{Timestamp: time.Unix(0, 10), Line: "10"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 15), Line: "15"},
									{Timestamp: time.Unix(0, 15), Line: "15"},
									{Timestamp: time.Unix(0, 16), Line: "16"},
								},
							},
						},
					},
				},
			},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     100,
				Version:   1,
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result: []logproto.Stream{
						{
							Labels: `{foo="bar", level="debug"}`,
							Entries: []logproto.Entry{

								{Timestamp: time.Unix(0, 5), Line: "5"},
								{Timestamp: time.Unix(0, 6), Line: "6"},
								{Timestamp: time.Unix(0, 15), Line: "15"},
								{Timestamp: time.Unix(0, 15), Line: "15"},
								{Timestamp: time.Unix(0, 16), Line: "16"},
							},
						},
						{
							Labels: `{foo="bar", level="error"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 1), Line: "1"},
								{Timestamp: time.Unix(0, 2), Line: "2"},
								{Timestamp: time.Unix(0, 9), Line: "9"},
								{Timestamp: time.Unix(0, 10), Line: "10"},
							},
						},
					},
				},
			},
			false,
		},
		{
			"loki forward limited",
			[]queryrange.Response{
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.FORWARD,
					Limit:     5,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 1), Line: "1"},
									{Timestamp: time.Unix(0, 2), Line: "2"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 5), Line: "5"},
									{Timestamp: time.Unix(0, 6), Line: "6"},
								},
							},
						},
					},
				},
				&LokiResponse{
					Status:    loghttp.QueryStatusSuccess,
					Direction: logproto.FORWARD,
					Limit:     5,
					Version:   1,
					Data: LokiData{
						ResultType: loghttp.ResultTypeStream,
						Result: []logproto.Stream{
							{
								Labels: `{foo="bar", level="error"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 9), Line: "9"},
									{Timestamp: time.Unix(0, 10), Line: "10"},
								},
							},
							{
								Labels: `{foo="bar", level="debug"}`,
								Entries: []logproto.Entry{
									{Timestamp: time.Unix(0, 15), Line: "15"},
									{Timestamp: time.Unix(0, 15), Line: "15"},
									{Timestamp: time.Unix(0, 16), Line: "16"},
								},
							},
						},
					},
				},
			},
			&LokiResponse{
				Status:    loghttp.QueryStatusSuccess,
				Direction: logproto.FORWARD,
				Limit:     5,
				Version:   1,
				Data: LokiData{
					ResultType: loghttp.ResultTypeStream,
					Result: []logproto.Stream{
						{
							Labels: `{foo="bar", level="debug"}`,
							Entries: []logproto.Entry{

								{Timestamp: time.Unix(0, 5), Line: "5"},
								{Timestamp: time.Unix(0, 6), Line: "6"},
							},
						},
						{
							Labels: `{foo="bar", level="error"}`,
							Entries: []logproto.Entry{
								{Timestamp: time.Unix(0, 1), Line: "1"},
								{Timestamp: time.Unix(0, 2), Line: "2"},
								{Timestamp: time.Unix(0, 9), Line: "9"},
							},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lokiCodec.MergeResponse(tt.responses...)
			if (err != nil) != tt.wantErr {
				t.Errorf("codec.MergeResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

type badResponse struct{}

func (badResponse) Reset()         {}
func (badResponse) String() string { return "noop" }
func (badResponse) ProtoMessage()  {}

type badReader struct{}

func (badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("")
}

var (
	matrixString = `{
	"data": {
	  "resultType": "matrix",
	  "result": [
		{
		  "metric": {
			"filename": "\/var\/hostlog\/apport.log",
			"job": "varlogs"
		  },
		  "values": [
			  [
				1568404331.324,
				"0.013333333333333334"
			  ]
			]
		},
		{
		  "metric": {
			"filename": "\/var\/hostlog\/syslog",
			"job": "varlogs"
		  },
		  "values": [
				[
					1568404331.324,
					"3.45"
				],
				[
					1568404331.339,
					"4.45"
				]
			]
		}
	  ]
	},
	"status": "success"
  }`
	sampleStreams = []queryrange.SampleStream{
		{
			Labels:  []client.LabelAdapter{{Name: "filename", Value: "/var/hostlog/apport.log"}, {Name: "job", Value: "varlogs"}},
			Samples: []client.Sample{{Value: 0.013333333333333334, TimestampMs: 1568404331324}},
		},
		{
			Labels:  []client.LabelAdapter{{Name: "filename", Value: "/var/hostlog/syslog"}, {Name: "job", Value: "varlogs"}},
			Samples: []client.Sample{{Value: 3.45, TimestampMs: 1568404331324}, {Value: 4.45, TimestampMs: 1568404331339}},
		},
	}
	streamsString = `{
		"status": "success",
		"data": {
			"resultType": "streams",
			"result": [
				{
					"stream": {
						"test": "test"
					},
					"values":[
						[ "123456789012345", "super line" ]
					]
				}
			]
		}
	}`
	streamsStringLegacy = `{"streams":[{"labels":"{test=\"test\"}","entries":[{"ts":"1970-01-02T10:17:36.789012345Z","line":"super line"}]}]}`
	logStreams          = []logproto.Stream{
		{
			Labels: `{test="test"}`,
			Entries: []logproto.Entry{
				{
					Line:      "super line",
					Timestamp: time.Unix(0, 123456789012345).UTC(),
				},
			},
		},
	}
)

func BenchmarkResponseMerge(b *testing.B) {
	const (
		resps         = 10
		streams       = 100
		logsPerStream = 1000
	)

	for _, tc := range []struct {
		desc  string
		limit uint32
		fn    func([]*LokiResponse, uint32, logproto.Direction) []logproto.Stream
	}{
		{
			"mergeStreams unlimited",
			uint32(streams * logsPerStream),
			mergeStreams,
		},
		{
			"mergeOrderedNonOverlappingStreams unlimited",
			uint32(streams * logsPerStream),
			mergeOrderedNonOverlappingStreams,
		},
		{
			"mergeStreams limited",
			uint32(streams*logsPerStream - 1),
			mergeStreams,
		},
		{
			"mergeOrderedNonOverlappingStreams limited",
			uint32(streams*logsPerStream - 1),
			mergeOrderedNonOverlappingStreams,
		},
	} {
		input := mkResps(resps, streams, logsPerStream, logproto.FORWARD)
		b.Run(tc.desc, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				tc.fn(input, tc.limit, logproto.FORWARD)
			}
		})
	}

}

func mkResps(nResps, nStreams, nLogs int, direction logproto.Direction) (resps []*LokiResponse) {
	for i := 0; i < nResps; i++ {
		r := &LokiResponse{}
		for j := 0; j < nStreams; j++ {
			stream := logproto.Stream{
				Labels: fmt.Sprintf(`{foo="%d"}`, j),
			}
			// split nLogs evenly across all responses
			for k := i * (nLogs / nResps); k < (i+1)*(nLogs/nResps); k++ {
				stream.Entries = append(stream.Entries, logproto.Entry{
					Timestamp: time.Unix(int64(k), 0),
					Line:      fmt.Sprintf("%d", k),
				})

				if direction == logproto.BACKWARD {
					for x, y := 0, len(stream.Entries)-1; x < len(stream.Entries)/2; x, y = x+1, y-1 {
						stream.Entries[x], stream.Entries[y] = stream.Entries[y], stream.Entries[x]
					}
				}
			}
			r.Data.Result = append(r.Data.Result, stream)
		}
		resps = append(resps, r)
	}
	return resps
}
