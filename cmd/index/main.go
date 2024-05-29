package main

import (
	"context"
	"math"

	"github.com/grafana/loki/v3/pkg/storage/stores/shipper/indexshipper/tsdb/index"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

func main() {
	reader, err := index.NewFileReader("/Users/cyril/Downloads/index_loki_ops_tsdb_index_19845_29_1714739725605204519-compactor-1714575565280-1714739165631-311e4749.tsdb")
	if err != nil {
		panic(err)
	}
	out, err := index.NewWriter(context.TODO(), index.FormatV3, "out.tsdb")
	if err != nil {
		panic(err)
	}

	// copy symbols
	symbolsIter := reader.Symbols()
	for symbolsIter.Next() {
		if err := out.AddSymbol(symbolsIter.At()); err != nil {
			panic(err)
		}
	}

	// copy series
	k, v := index.AllPostingsKey()
	postings, err := reader.Postings(k, nil, v)
	if err != nil {
		panic(err)
	}

	var (
		lbls labels.Labels
		chks []index.ChunkMeta
		fp   uint64
	)
	totalSeries := 0
	totalchunks := 0
	// iterate over all the symbols
	for postings.Next() {
		seriesRef := postings.At()
		// get the labels for the series
		fp, err = reader.Series(seriesRef, 0, math.MaxInt64, &lbls, &chks)
		if err != nil {
			panic(err)
		}
		out.AddSeries(seriesRef, lbls, model.Fingerprint(fp))
		totalSeries++
		totalchunks += len(chks)

	}
	if err := postings.Err(); err != nil {
		panic(err)
	}
	if err := out.Close(); err != nil {
		panic(err)
	}
	println("success!")
	println("total series:", totalSeries)
	println("total chunks:", totalchunks)
}
