package chunk

import (
	"context"
	"sort"
	"sync"

	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql"

	"github.com/grafana/loki/pkg/storage/chunk/cache"
	util_log "github.com/grafana/loki/pkg/util/log"
	"github.com/grafana/loki/pkg/util/spanlogger"
)

var (
	errAsyncBufferFull = errors.New("the async buffer is full")
	skipped            = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loki_chunk_fetcher_cache_skipped_buffer_full_total",
		Help: "Total number of operations against cache that have been skipped.",
	})
	chunkFetcherCacheQueueEnqueue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loki_chunk_fetcher_cache_enqueued_total",
		Help: "Total number of chunks enqueued to a buffer to be asynchronously written back to the chunk cache.",
	})
	chunkFetcherCacheQueueDequeue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loki_chunk_fetcher_cache_dequeued_total",
		Help: "Total number of chunks asynchronously dequeued from a buffer and written back to the chunk cache.",
	})
)

const chunkDecodeParallelism = 16

func filterLazyChunksByTime(from, through model.Time, chunks []*LazyChunk) []*LazyChunk {
	filtered := make([]*LazyChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.Through < from || through < chunk.From {
			continue
		}
		filtered = append(filtered, chunk)
	}
	return filtered
}

func labelNamesFromChunks(refs []*LazyChunk) []string {
	var result UniqueStrings
	for _, c := range refs {
		for _, l := range c.Chunk().Metric {
			result.Add(l.Name)
		}
	}
	return result.Strings()
}

func filterLazyChunksByUniqueFingerprint(chunks []*LazyChunk) []*LazyChunk {
	filtered := make([]*LazyChunk, 0, len(chunks))
	uniqueFp := map[model.Fingerprint]struct{}{}

	for _, chunk := range chunks {
		if _, ok := uniqueFp[chunk.FingerprintModel()]; ok {
			continue
		}
		filtered = append(filtered, chunk)
		uniqueFp[chunk.FingerprintModel()] = struct{}{}
	}
	return filtered
}

// Fetcher deals with fetching chunk contents from the cache/store,
// and writing back any misses to the cache.  Also responsible for decoding
// chunks from the cache, in parallel.
type Fetcher struct {
	pcfg       PeriodConfig
	storage    Client
	cache      cache.Cache
	cacheStubs bool

	wait           sync.WaitGroup
	decodeRequests chan decodeRequest

	maxAsyncConcurrency int
	maxAsyncBufferSize  int

	asyncQueue chan []*LazyChunk
	stop       chan struct{}
}

type decodeRequest struct {
	chunk     *Chunk
	buf       []byte
	responses chan decodeResponse
}

type decodeResponse struct {
	err error
}

// NewChunkFetcher makes a new ChunkFetcher.
func NewChunkFetcher(cacher cache.Cache, cacheStubs bool, pcfg PeriodConfig, storage Client, maxAsyncConcurrency int, maxAsyncBufferSize int) (*Fetcher, error) {
	c := &Fetcher{
		pcfg:                pcfg,
		storage:             storage,
		cache:               cacher,
		cacheStubs:          cacheStubs,
		decodeRequests:      make(chan decodeRequest),
		maxAsyncConcurrency: maxAsyncConcurrency,
		maxAsyncBufferSize:  maxAsyncBufferSize,
		stop:                make(chan struct{}),
	}

	c.wait.Add(chunkDecodeParallelism)
	for i := 0; i < chunkDecodeParallelism; i++ {
		go c.worker()
	}

	// Start a number of goroutines - processing async operations - equal
	// to the max concurrency we have.
	c.asyncQueue = make(chan []*LazyChunk, c.maxAsyncBufferSize)
	for i := 0; i < c.maxAsyncConcurrency; i++ {
		go c.asyncWriteBackCacheQueueProcessLoop()
	}

	return c, nil
}

func (c *Fetcher) writeBackCacheAsync(refs []*LazyChunk) error {
	select {
	case c.asyncQueue <- refs:
		chunkFetcherCacheQueueEnqueue.Add(float64(len(refs)))
		return nil
	default:
		return errAsyncBufferFull
	}
}

func (c *Fetcher) asyncWriteBackCacheQueueProcessLoop() {
	for {
		select {
		case fromStorage := <-c.asyncQueue:
			chunkFetcherCacheQueueDequeue.Add(float64(len(fromStorage)))
			cacheErr := c.writeBackCache(context.Background(), fromStorage)
			if cacheErr != nil {
				level.Warn(util_log.Logger).Log("msg", "could not write fetched chunks from storage into chunk cache", "err", cacheErr)
			}
		case <-c.stop:
			return
		}
	}
}

// Stop the ChunkFetcher.
func (c *Fetcher) Stop() {
	close(c.decodeRequests)
	c.wait.Wait()
	c.cache.Stop()
	close(c.stop)
}

func (c *Fetcher) worker() {
	defer c.wait.Done()
	decodeContext := NewDecodeContext()
	for req := range c.decodeRequests {
		err := req.chunk.Decode(decodeContext, req.buf)
		if err != nil {
			cacheCorrupt.Inc()
		}
		req.responses <- decodeResponse{
			err: err,
		}
	}
}

// FetchChunks fetches a set of chunks from cache and store. Note that the keys passed in must be
// lexicographically sorted, while the returned chunks are not in the same order as the passed in chunks.
func (c *Fetcher) FetchChunks(ctx context.Context, refs []*LazyChunk) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	log, ctx := spanlogger.New(ctx, "ChunkStore.FetchChunks")
	defer log.Span.Finish()

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ExternalKey < refs[j].ExternalKey
	})
	keys := make([]string, len(refs))
	for i, ref := range refs {
		keys[i] = ref.ExternalKey
	}

	// Now fetch the actual chunk data from Memcache / S3
	cacheHits, cacheBufs, _, err := c.cache.Fetch(ctx, keys)
	if err != nil {
		level.Warn(log).Log("msg", "error fetching from cache", "err", err)
	}
	missing, err := c.processCacheResponse(ctx, refs, cacheHits, cacheBufs)
	if err != nil {
		level.Warn(log).Log("msg", "error process response from cache", "err", err)
	}

	if len(missing) > 0 {
		err = c.storage.GetChunks(ctx, missing)
	}

	// Always cache any chunks we did get
	if cacheErr := c.writeBackCacheAsync(missing); cacheErr != nil {
		if cacheErr == errAsyncBufferFull {
			skipped.Inc()
		}
		level.Warn(log).Log("msg", "could not store chunks in chunk cache", "err", cacheErr)
	}

	if err != nil {
		// Don't rely on Cortex error translation here.
		return promql.ErrStorage{Err: err}
	}

	return nil
}

func (c *Fetcher) writeBackCache(ctx context.Context, refs []*LazyChunk) error {
	keys := make([]string, 0, len(refs))
	bufs := make([][]byte, 0, len(refs))
	for i := range refs {
		var encoded []byte
		var err error
		if !c.cacheStubs {
			encoded, err = refs[i].Chunk().Encoded()
			// TODO don't fail, just log and continue?
			if err != nil {
				return err
			}
		}

		keys = append(keys, refs[i].ExternalKey)
		bufs = append(bufs, encoded)
	}

	err := c.cache.Store(ctx, keys, bufs)
	if err != nil {
		level.Warn(util_log.Logger).Log("msg", "writeBackCache cache store fail", "err", err)
	}
	return nil
}

// ProcessCacheResponse decodes the chunks coming back from the cache, separating
// hits and misses.
func (c *Fetcher) processCacheResponse(ctx context.Context, refs []*LazyChunk, keys []string, bufs [][]byte) ([]*LazyChunk, error) {
	var (
		requests  = make([]decodeRequest, 0, len(keys))
		responses = make(chan decodeResponse)
		missing   = make([]*LazyChunk, 0, len(refs))
		logger    = util_log.WithContext(ctx, util_log.Logger)
	)

	i, j := 0, 0
	for i < len(refs) && j < len(keys) {
		chunkKey := refs[i].ExternalKey

		if chunkKey < keys[j] {
			missing = append(missing, refs[i])
			i++
		} else if chunkKey > keys[j] {
			level.Warn(logger).Log("msg", "got chunk from cache we didn't ask for")
			j++
		} else {
			requests = append(requests, decodeRequest{
				chunk:     refs[i].Chunk(),
				buf:       bufs[j],
				responses: responses,
			})
			i++
			j++
		}
	}
	for ; i < len(refs); i++ {
		missing = append(missing, refs[i])
	}
	level.Debug(logger).Log("refs", len(refs), "decodeRequests", len(requests), "missing", len(missing))

	go func() {
		for _, request := range requests {
			c.decodeRequests <- request
		}
	}()

	var err error
	for i := 0; i < len(requests); i++ {
		response := <-responses
		// Don't exit early, as we don't want to block the workers.
		if response.err != nil {
			err = response.err
		}
	}
	return missing, err
}

func (c *Fetcher) IsChunkNotFoundErr(err error) bool {
	return c.storage.IsChunkNotFoundErr(err)
}
