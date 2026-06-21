# logstats

Package `logstats` provides utilities for collecting, aggregating, and analysing
statistics about log streams in Loki. It is designed to be embedded in ingestion
and querying paths to give operators visibility into per-stream behaviour without
requiring an external metrics pipeline.

## Features

* **Stream counters** – lightweight, goroutine-safe per-stream byte / line counters.
* **Rate calculator** – sliding-window rates (bytes/s and lines/s) over configurable
  durations, backed by a ring-buffer to avoid heap pressure.
* **Label cardinality analyser** – detects high-cardinality label sets before they
  reach the index, helping prevent the "cardinality explosion" anti-pattern.
* **Top-N selector** – returns the N streams producing the most volume over a window,
  useful for dashboards and auto-alerting.
* **Aggregator** – orchestrates all of the above and exposes a single, consistent
  snapshot that can be serialised to JSON or converted to Prometheus metrics.

## Quick start

```go
package main

import (
    "fmt"
    "time"

    "github.com/grafana/loki/pkg/logstats"
)

func main() {
    agg := logstats.NewAggregator(logstats.DefaultAggregatorConfig())

    // Record some log activity.
    agg.Record("app=nginx,env=prod", 1024, 10)
    agg.Record("app=mysql,env=prod", 512, 5)
    agg.Record("app=nginx,env=prod", 2048, 20)

    time.Sleep(time.Second)

    snap := agg.Snapshot()
    fmt.Printf("Total streams : %d\n", snap.TotalStreams)
    fmt.Printf("Total bytes   : %d\n", snap.TotalBytes)
    fmt.Printf("Total lines   : %d\n", snap.TotalLines)

    for _, s := range snap.TopStreams {
        fmt.Printf("  %-40s  %6d bytes  %4d lines\n", s.Labels, s.Bytes, s.Lines)
    }
}
```

## Design notes

* All public types are safe for concurrent use.
* The ring-buffer inside `RateCalculator` is pre-allocated to `WindowSize/Resolution`
  slots; choose `Resolution` carefully for memory vs. accuracy trade-offs.
* Label cardinality is tracked via a simple HLL-inspired counting approach (exact
  counting with a configurable high-water mark) so it stays O(1) per record call.
