package main

import (
	"context"
	"flag"
	"log"
	"sort"

	"github.com/grafana/loki/pkg/storage/stores/shipper/compactor/retention"
	shipper_util "github.com/grafana/loki/pkg/storage/stores/shipper/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunks"
	"github.com/prometheus/prometheus/tsdb/index"
	"go.etcd.io/bbolt"
)

var (
	source = flag.String("source", "", "the source boltdb file")
	dest   = flag.String("dest", "", "the dest tsdb file")
)

func main() {
	flag.Parse()

	if source == nil || *source == "" {
		panic("source is required")
	}

	if dest == nil || *dest == "" {
		panic("dest is required")
	}

	db, err := shipper_util.SafeOpenBoltdbFile(*source)
	if err != nil {
		panic(err)
	}

	tsdbindex, err := index.NewWriter(context.Background(), *dest)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := tsdbindex.Close(); err != nil {
			panic(err)
		}
	}()

	symbolsMap := make(map[string]struct{})
	seriesMap := make(map[string]*series)

	log.Print("Loading index into memory")
	// loads everything into memory.
	if err := db.View(func(t *bbolt.Tx) error {
		it, err := retention.NewChunkIndexIterator(t.Bucket([]byte("index")))
		if err != nil {
			return err
		}

		for it.Next() {
			if it.Err() != nil {
				return it.Err()
			}
			entry := it.Entry()
			for _, v := range entry.Labels {
				symbolsMap[v.Name] = struct{}{}
				symbolsMap[v.Value] = struct{}{}
			}
			lbsString := entry.Labels.String()
			var s *series
			var ok bool
			if s, ok = seriesMap[lbsString]; !ok {
				s = &series{
					lbsString: lbsString,
					lbs:       entry.Labels,
				}
				seriesMap[lbsString] = s
			}
			s.chunks = append(s.chunks, chunks.Meta{
				MinTime: int64(entry.From),
				MaxTime: int64(entry.Through),
			})
		}

		return nil
	}); err != nil {
		panic(err)
	}
	log.Print("sorting index")
	// sort
	symbols := make([]string, 0, len(symbolsMap))
	for k := range symbolsMap {
		symbols = append(symbols, k)
	}
	sort.Strings(symbols)

	seriesSet := make([]*series, 0, len(seriesMap))
	for _, s := range seriesMap {
		seriesSet = append(seriesSet, s)
	}
	sort.Slice(seriesSet, func(i, j int) bool {
		return labels.Compare(seriesSet[i].lbs, seriesSet[j].lbs) < 0
	})
	for _, s := range seriesSet {
		sort.Slice(s.chunks, func(i, j int) bool {
			return s.chunks[i].MinTime < s.chunks[j].MinTime
		})
		for i := range s.chunks {
			s.chunks[i].Ref = chunks.ChunkRef(i)
		}
	}

	log.Print("writing index")
	// then flush to disk
	for _, s := range symbols {
		if err := tsdbindex.AddSymbol(s); err != nil {
			panic(err)
		}
	}
	for i, s := range seriesSet {
		if err := tsdbindex.AddSeries(storage.SeriesRef(i), s.lbs, s.chunks...); err != nil {
			panic(err)
		}
	}
}

type series struct {
	lbs       labels.Labels
	lbsString string
	chunks    []chunks.Meta
}
