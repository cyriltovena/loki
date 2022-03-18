package storage

import (
	"context"
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"

	"github.com/grafana/loki/pkg/util/spanlogger"

	"github.com/grafana/loki/pkg/storage/chunk"
	util_log "github.com/grafana/loki/pkg/util/log"
)

type IngesterQuerier interface {
	GetChunkIDs(ctx context.Context, from, through model.Time, matchers ...*labels.Matcher) ([]string, error)
}

// AsyncStore does querying to both ingesters and chunk store and combines the results after deduping them.
// This should be used when using an async store like boltdb-shipper.
// AsyncStore is meant to be used only in queriers or any other service other than ingesters.
// It should never be used in ingesters otherwise it would start spiraling around doing queries over and over again to other ingesters.
type AsyncStore struct {
	chunk.Store
	scfg                 chunk.SchemaConfig
	ingesterQuerier      IngesterQuerier
	queryIngestersWithin time.Duration
}

func NewAsyncStore(store chunk.Store, scfg chunk.SchemaConfig, querier IngesterQuerier, queryIngestersWithin time.Duration) *AsyncStore {
	return &AsyncStore{
		Store:                store,
		scfg:                 scfg,
		ingesterQuerier:      querier,
		queryIngestersWithin: queryIngestersWithin,
	}
}

func (a *AsyncStore) GetChunkRefs(ctx context.Context, userID string, from, through model.Time, matchers ...*labels.Matcher) ([]chunk.LazyChunk, error) {
	spanLogger := spanlogger.FromContext(ctx)

	errs := make(chan error)

	var storeChunks []chunk.LazyChunk
	go func() {
		var err error
		storeChunks, err = a.Store.GetChunkRefs(ctx, userID, from, through, matchers...)
		errs <- err
	}()

	var ingesterChunks []string

	go func() {
		if a.queryIngestersWithin != 0 {
			// don't query ingesters if the query does not overlap with queryIngestersWithin.
			if !through.After(model.Now().Add(-a.queryIngestersWithin)) {
				level.Debug(util_log.Logger).Log("msg", "skipping querying ingesters for chunk ids", "query-from", from, "query-through", through)
				errs <- nil
				return
			}
		}

		var err error
		ingesterChunks, err = a.ingesterQuerier.GetChunkIDs(ctx, from, through, matchers...)

		if err == nil {
			level.Debug(spanLogger).Log("ingester-chunks-count", len(ingesterChunks))
			level.Debug(util_log.Logger).Log("msg", "got chunk ids from ingester", "count", len(ingesterChunks))
		}
		errs <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}

	if len(ingesterChunks) == 0 {
		return storeChunks, nil
	}

	return a.mergeIngesterAndStoreChunks(ctx, userID, storeChunks, ingesterChunks)
}

func (a *AsyncStore) mergeIngesterAndStoreChunks(ctx context.Context, userID string, storeChunks []chunk.LazyChunk, ingesterChunkIDs []string) ([]chunk.LazyChunk, error) {
	ingesterChunkIDs = filterDuplicateChunks(a.scfg, storeChunks, ingesterChunkIDs)
	level.Debug(util_log.Logger).Log("msg", "post-filtering ingester chunks", "count", len(ingesterChunkIDs))

	ingesterChunks, err := a.Store.LazyChunksForKeys(userID, ingesterChunkIDs)
	if err != nil {
		return nil, err
	}
	return append(storeChunks, ingesterChunks...), nil
}

func filterDuplicateChunks(scfg chunk.SchemaConfig, storeChunks []chunk.LazyChunk, ingesterChunkIDs []string) []string {
	filteredChunkIDs := make([]string, 0, len(ingesterChunkIDs))
	seen := make(map[string]struct{}, len(storeChunks))

	for i := range storeChunks {
		seen[storeChunks[i].ExternalKey] = struct{}{}
	}

	for _, chunkID := range ingesterChunkIDs {
		if _, ok := seen[chunkID]; !ok {
			filteredChunkIDs = append(filteredChunkIDs, chunkID)
			seen[chunkID] = struct{}{}
		}
	}

	return filteredChunkIDs
}
