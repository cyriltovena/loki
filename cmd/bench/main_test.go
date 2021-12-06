package main

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunks"
	"github.com/prometheus/prometheus/tsdb/index"
)

var chks []chunks.Meta

func Benchmark_IndexRead(b *testing.B) {
	ir, err := index.NewFileReader("/Users/ctovena/go/src/github.com/grafana/loki/cmd/bench/tsdb-ops")
	if err != nil {
		b.Fatal(err)
	}
	defer ir.Close()

	for i := 0; i < b.N; i++ {
		p, err := tsdb.PostingsForMatchers(ir, labels.MustNewMatcher(labels.MatchEqual, "namespace", "loki-prod"))
		if err != nil {
			b.Fatal(err)
		}

		var (
			bufChks []chunks.Meta
			bufLbls labels.Labels
		)

		for p.Next() {
			if err := ir.Series(p.At(), &bufLbls, &bufChks); err != nil {
				panic(err)
			}

			if len(bufChks) == 0 {
				continue
			}

			chks = append(chks, bufChks...)
		}
	}
	b.Log(len(chks))
}
