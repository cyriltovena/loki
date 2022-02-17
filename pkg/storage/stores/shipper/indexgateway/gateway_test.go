package indexgateway

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/grafana/loki/pkg/storage/chunk"
	"github.com/grafana/loki/pkg/storage/chunk/local"
	"github.com/grafana/loki/pkg/storage/stores/shipper/downloads"
	"github.com/grafana/loki/pkg/storage/stores/shipper/indexgateway/indexgatewaypb"
	"github.com/grafana/loki/pkg/storage/stores/shipper/storage"
	"github.com/grafana/loki/pkg/storage/stores/shipper/testutil"
	"github.com/grafana/loki/pkg/storage/stores/shipper/util"
	util_math "github.com/grafana/loki/pkg/util/math"
)

const (
	// query prefixes
	tableNamePrefix        = "table-name"
	hashValuePrefix        = "hash-value"
	rangeValuePrefixPrefix = "range-value-prefix"
	rangeValueStartPrefix  = "range-value-start"
	valueEqualPrefix       = "value-equal"

	// response prefixes
	rangeValuePrefix = "range-value"
	valuePrefix      = "value"
)

type mockBatch struct {
	size int
}

func (r *mockBatch) Iterator() chunk.ReadBatchIterator {
	return &mockBatchIter{
		curr: -1,
		size: r.size,
	}
}

type mockBatchIter struct {
	curr, size int
}

func (b *mockBatchIter) Next() bool {
	b.curr++
	return b.curr < b.size
}

func (b *mockBatchIter) RangeValue() []byte {
	return []byte(fmt.Sprintf("%s%d", rangeValuePrefix, b.curr))
}

func (b *mockBatchIter) Value() []byte {
	return []byte(fmt.Sprintf("%s%d", valuePrefix, b.curr))
}

type mockQueryIndexServer struct {
	grpc.ServerStream
	callback func(resp *indexgatewaypb.QueryIndexResponse)
}

func (m *mockQueryIndexServer) Send(resp *indexgatewaypb.QueryIndexResponse) error {
	m.callback(resp)
	return nil
}

func TestGateway_sendBatch(t *testing.T) {
	var expectedQueryKey string
	type batchRange struct {
		start, end int
	}
	var expectedRanges []batchRange

	// the response should have index entries between start and end from batchRange at index 0 of expectedRanges
	var server indexgatewaypb.IndexGateway_QueryIndexServer = &mockQueryIndexServer{
		callback: func(resp *indexgatewaypb.QueryIndexResponse) {
			require.Equal(t, expectedQueryKey, resp.QueryKey)

			require.True(t, len(expectedRanges) > 0)
			require.Len(t, resp.Rows, expectedRanges[0].end-expectedRanges[0].start)
			i := expectedRanges[0].start
			for _, row := range resp.Rows {
				require.Equal(t, fmt.Sprintf("%s%d", rangeValuePrefix, i), string(row.RangeValue))
				require.Equal(t, fmt.Sprintf("%s%d", valuePrefix, i), string(row.Value))
				i++
			}

			// remove first element for checking the response from next callback.
			expectedRanges = expectedRanges[1:]
		},
	}

	gateway := gateway{}
	responseSizes := []int{0, 99, maxIndexEntriesPerResponse, 2 * maxIndexEntriesPerResponse, 5*maxIndexEntriesPerResponse - 1}
	for i, responseSize := range responseSizes {
		query := chunk.IndexQuery{
			TableName:        fmt.Sprintf("%s%d", tableNamePrefix, i),
			HashValue:        fmt.Sprintf("%s%d", hashValuePrefix, i),
			RangeValuePrefix: []byte(fmt.Sprintf("%s%d", rangeValuePrefixPrefix, i)),
			RangeValueStart:  []byte(fmt.Sprintf("%s%d", rangeValueStartPrefix, i)),
			ValueEqual:       []byte(fmt.Sprintf("%s%d", valueEqualPrefix, i)),
		}

		// build expectedRanges based on maxIndexEntriesPerResponse
		for j := 0; j < responseSize; j += maxIndexEntriesPerResponse {
			expectedRanges = append(expectedRanges, batchRange{
				start: j,
				end:   util_math.Min(j+maxIndexEntriesPerResponse, responseSize),
			})
		}
		expectedQueryKey = util.QueryKey(query)

		err := gateway.sendBatch(server, query, &mockBatch{responseSize})
		require.NoError(t, err)

		// verify that we actually got responses back by checking if expectedRanges got cleared.
		require.Len(t, expectedRanges, 0)
	}
}

var total int

func Benchmark_IndexQueries(b *testing.B) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	s := grpc.NewServer(grpc.ChainStreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return middleware.StreamServerUserHeaderInterceptor(srv, ss, info, handler)
	}))
	conn, _ := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	defer func() {
		s.Stop()
		conn.Close()
	}()

	dir := b.TempDir()
	bclient, err := local.NewBoltDBIndexClient(local.BoltDBConfig{
		Directory: dir + "/boltdb",
	})
	numRecords := 10000
	require.NoError(b, os.MkdirAll(dir+"/index/test_table", 0777))
	require.NoError(b, os.MkdirAll(dir+"/cache/test_table", 0777))
	testutil.AddRecordsToDB(b, dir+"/index/test_table/db1", bclient, 0, numRecords, []byte("index"))
	testutil.AddRecordsToDB(b, dir+"/cache/test_table/db1", bclient, 0, numRecords, []byte("index"))
	require.NoError(b, err)
	fs, err := local.NewFSObjectClient(local.FSConfig{
		Directory: dir,
	})
	require.NoError(b, err)
	tm, err := downloads.NewTableManager(downloads.Config{
		CacheDir:          dir + "/cache",
		SyncInterval:      15 * time.Minute,
		CacheTTL:          15 * time.Minute,
		QueryReadyNumDays: 30,
	}, bclient, storage.NewIndexStorageClient(fs, "index/"), nil)
	require.NoError(b, err)
	gw := NewIndexGateway(tm)
	indexgatewaypb.RegisterIndexGatewayServer(s, gw)
	ctx := user.InjectOrgID(context.Background(), "fooid")
	go func() {
		if err := s.Serve(listener); err != nil {
			panic(err)
		}
	}()
	client := indexgatewaypb.NewIndexGatewayClient(conn)
	queries := []*indexgatewaypb.IndexQuery{}
	for i := 0; i < numRecords; i++ {
		queries = append(queries, &indexgatewaypb.IndexQuery{
			TableName:  "test_table",
			ValueEqual: []byte(strconv.Itoa(i)),
		})
	}
	ctx, _ = user.InjectIntoGRPCRequest(ctx)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.QueryIndex(ctx, &indexgatewaypb.QueryIndexRequest{Queries: queries})
		require.NoError(b, err)
		var resp *indexgatewaypb.QueryIndexResponse
		for {
			resp, err = stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			total += len(resp.Rows)
		}
	}
}
