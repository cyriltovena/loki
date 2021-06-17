// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tsdb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/loki/pkg/logql"
)

var ErrInvalidTimes = fmt.Errorf("max time is lesser than min time")

// CreateBlock creates a chunkrange block from the samples passed to it, and writes it to disk.
func CreateBlock(series []logproto.Stream, dir string, chunkRange int64, logger log.Logger) (string, error) {
	if chunkRange == 0 {
		chunkRange = DefaultBlockDuration
	}
	if chunkRange < 0 {
		return "", ErrInvalidTimes
	}

	w, err := NewBlockWriter(logger, dir, chunkRange)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := w.Close(); err != nil {
			logger.Log("err closing blockwriter", err.Error())
		}
	}()

	ctx := context.Background()
	app := w.Appender(ctx)

	for _, s := range series {
		ref := uint64(0)

		lset, err := logql.ParseLabels(s.Labels)
		if err != nil {
			return "", err
		}
		for _, e := range s.Entries {
			ref, err = app.Append(ref, lset, e.Timestamp.UnixNano(), []byte(e.Line))
			if err != nil {
				return "", err
			}
		}
	}

	if err = app.Commit(); err != nil {
		return "", err
	}

	ulid, err := w.Flush(ctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ulid.String()), nil
}
