# logsampling

Package `logsampling` provides rate-based log sampling middleware for Loki pipelines.

## Overview

A `Sampler` evaluates each log line against an ordered list of `Rule`s. The first matching rule determines the sampling rate. If no rule matches, the line is always kept.

## Usage

```go
sampler := logsampling.New([]logsampling.Rule{
    {LabelMatcher: "app=nginx", Rate: 10},  // keep 10% of nginx logs
    {LabelMatcher: "app=mysql", Rate: 2},   // keep 50% of mysql logs
})

if sampler.ShouldKeep(labels, line) {
    // forward line to storage
}
```

## Rule format

- `LabelMatcher`: comma-separated `key=value` pairs (e.g. `app=nginx,env=prod`)
- `Rate`: keep 1 in N lines. `1` = keep all, `10` = keep ~10%.
