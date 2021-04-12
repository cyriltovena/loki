package retention

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/util/math"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.etcd.io/bbolt"

	"github.com/grafana/loki/pkg/storage"
)

const (
	chunkTimeRangeKeyV3   = '3'
	metricNameRangeKeyV1  = '6'
	seriesRangeKeyV1      = '7'
	labelSeriesRangeKeyV1 = '8'
	labelNamesRangeKeyV1  = '9'
	separator             = "\000"
)

var QueryParallelism = 100

var ErrInvalidIndexKey = errors.New("invalid index key")

type InvalidIndexKeyError struct {
	HashKey  string
	RangeKey string
}

func newInvalidIndexKeyError(h, r []byte) InvalidIndexKeyError {
	return InvalidIndexKeyError{
		HashKey:  string(h),
		RangeKey: string(r),
	}
}

func (e InvalidIndexKeyError) Error() string {
	return fmt.Sprintf("%s: hash_key:%s range_key:%s", ErrInvalidIndexKey, e.HashKey, e.RangeKey)
}

func (e InvalidIndexKeyError) Is(target error) bool {
	return target == ErrInvalidIndexKey
}

type ChunkRef struct {
	UserID   []byte
	SeriesID []byte
	ChunkID  []byte
	From     model.Time
	Through  model.Time
}

func (c ChunkRef) String() string {
	return fmt.Sprintf("UserID: %s , SeriesID: %s , Time: [%s,%s]", c.UserID, c.SeriesID, c.From, c.Through)
}

func parseChunkRef(hashKey, rangeKey []byte) (ChunkRef, bool, error) {
	componentsRef := getComponents()
	defer putComponents(componentsRef)
	components := componentsRef.components

	components = decodeRangeKey(rangeKey, components)
	if len(components) == 0 {
		return ChunkRef{}, false, newInvalidIndexKeyError(hashKey, rangeKey)
	}

	keyType := components[len(components)-1]
	if len(keyType) == 0 || keyType[0] != chunkTimeRangeKeyV3 {
		return ChunkRef{}, false, nil
	}
	chunkID := components[len(components)-2]

	userID, hexFrom, hexThrough, ok := parseChunkID(chunkID)
	if !ok {
		return ChunkRef{}, false, newInvalidIndexKeyError(hashKey, rangeKey)
	}
	from, err := strconv.ParseInt(unsafeGetString(hexFrom), 16, 64)
	if err != nil {
		return ChunkRef{}, false, err
	}
	through, err := strconv.ParseInt(unsafeGetString(hexThrough), 16, 64)
	if err != nil {
		return ChunkRef{}, false, err
	}

	return ChunkRef{
		UserID:   userID,
		SeriesID: seriesFromHash(hashKey),
		From:     model.Time(from),
		Through:  model.Time(through),
		ChunkID:  chunkID,
	}, true, nil
}

func parseChunkID(chunkID []byte) (userID []byte, hexFrom, hexThrough []byte, valid bool) {
	var (
		j, i int
		hex  []byte
	)

	for j < len(chunkID) {
		if chunkID[j] != '/' {
			j++
			continue
		}
		userID = chunkID[:j]
		hex = chunkID[j+1:]
		break
	}
	if len(userID) == 0 {
		return nil, nil, nil, false
	}
	_, i = readOneHexPart(hex)
	if i == 0 {
		return nil, nil, nil, false
	}
	hex = hex[i+1:]
	hexFrom, i = readOneHexPart(hex)
	if i == 0 {
		return nil, nil, nil, false
	}
	hex = hex[i+1:]
	hexThrough, i = readOneHexPart(hex)
	if i == 0 {
		return nil, nil, nil, false
	}
	return userID, hexFrom, hexThrough, true
}

func readOneHexPart(hex []byte) (part []byte, i int) {
	for i < len(hex) {
		if hex[i] != ':' {
			i++
			continue
		}
		return hex[:i], i
	}
	return nil, 0
}

type LabelIndexRef struct {
	KeyType  byte
	SeriesID []byte
}

func parseLabelIndexRef(hashKey, rangeKey []byte) (*LabelIndexRef, bool, error) {
	// todo reuse memory via pool
	var (
		components [][]byte
		seriesID   []byte
	)
	components = decodeRangeKey(rangeKey, components)
	if len(components) < 4 {
		return nil, false, newInvalidIndexKeyError(hashKey, rangeKey)
	}
	keyType := components[len(components)-1]
	if len(keyType) == 0 {
		return nil, false, nil
	}
	switch keyType[0] {
	case labelSeriesRangeKeyV1:
		seriesID = components[1]
	case seriesRangeKeyV1:
		seriesID = components[0]
	default:
		return nil, false, nil
	}
	return &LabelIndexRef{
		KeyType:  keyType[0],
		SeriesID: seriesID,
	}, true, nil
}

type LabelSeriesRangeKey struct {
	SeriesID []byte
	UserID   []byte
	Name     []byte
}

func (l LabelSeriesRangeKey) String() string {
	return fmt.Sprintf("%s:%s:%s", l.SeriesID, l.UserID, l.Name)
}

func parseLabelSeriesRangeKey(hashKey, rangeKey []byte) (LabelSeriesRangeKey, bool, error) {
	rangeComponentsRef := getComponents()
	defer putComponents(rangeComponentsRef)
	rangeComponents := rangeComponentsRef.components
	hashComponentsRef := getComponents()
	defer putComponents(hashComponentsRef)
	hashComponents := hashComponentsRef.components

	rangeComponents = decodeRangeKey(rangeKey, rangeComponents)
	if len(rangeComponents) < 4 {
		return LabelSeriesRangeKey{}, false, newInvalidIndexKeyError(hashKey, rangeKey)
	}
	keyType := rangeComponents[len(rangeComponents)-1]
	if len(keyType) == 0 || keyType[0] != labelSeriesRangeKeyV1 {
		return LabelSeriesRangeKey{}, false, nil
	}
	hashComponents = splitBytesBy(hashKey, ':', hashComponents)
	// 	> v10		HashValue:  fmt.Sprintf("%02d:%s:%s:%s", shard, bucket.hashKey , metricName, v.Name),
	// < v10		HashValue:  fmt.Sprintf("%s:%s:%s", bucket.hashKey, metricName, v.Name),

	if len(hashComponents) < 4 {
		return LabelSeriesRangeKey{}, false, newInvalidIndexKeyError(hashKey, rangeKey)
	}
	return LabelSeriesRangeKey{
		SeriesID: rangeComponents[1],
		Name:     hashComponents[len(hashComponents)-1],
		UserID:   hashComponents[len(hashComponents)-4],
	}, true, nil
}

func findSeriesIDsForRules(db *bbolt.DB, config chunk.PeriodConfig, rules []StreamRule) ([][]string, error) {
	schema, err := config.CreateSchema()
	if err != nil {
		return nil, err
	}
	// cover the whole table.
	from, through := config.From.Time, config.From.Time.Add(config.IndexTables.Period)
	result := make([][]string, len(rules))

	for ruleIndex, rule := range rules {
		incomingIDs := make(chan []string)
		incomingErrors := make(chan error)

		for _, matcher := range rule.Matchers {
			go func(matcher *labels.Matcher) {
				ids, err := lookupSeriesByMatcher(db, schema, from, through, rule.UserID, matcher)
				if err != nil {
					incomingErrors <- err
					return
				}
				incomingIDs <- ids
			}(&matcher)
		}
		// intersect. and add to result.
		var ids []string
		var lastErr error
		var initialized bool
		for i := 0; i < len(rule.Matchers); i++ {
			select {
			case incoming := <-incomingIDs:
				if !initialized {
					ids = incoming
					initialized = true
				} else {
					ids = intersectStrings(ids, incoming)
				}
			case err := <-incomingErrors:
				lastErr = err
			}
		}
		if lastErr != nil {
			return nil, err
		}
		result[ruleIndex] = ids
	}

	return result, nil
}

func lookupSeriesByMatcher(
	db *bbolt.DB,
	schema chunk.BaseSchema,
	from, through model.Time,
	userID string,
	matcher *labels.Matcher) ([]string, error) {
	queries, err := schema.GetReadQueriesForMetricLabelValue(
		from, through, userID, "logs", matcher.Name, matcher.Value)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return nil, nil
	}
	if len(queries) == 1 {
		return lookupSeriesByQuery(db, queries[0])
	}
	queue := make(chan chunk.IndexQuery)
	incomingResult := make(chan struct {
		ids []string
		err error
	})
	n := math.Min(len(queries), QueryParallelism)
	for i := 0; i < n; i++ {
		go func() {
			for {
				query, ok := <-queue
				if !ok {
					return
				}
				res, err := lookupSeriesByQuery(db, query)
				incomingResult <- struct {
					ids []string
					err error
				}{res, err}
			}
		}()
	}
	go func() {
		for _, query := range queries {
			queue <- query
		}
		close(queue)
	}()

	// Now receive all the results.
	var ids []string
	var lastErr error
	for i := 0; i < len(queries); i++ {
		res := <-incomingResult
		if res.err != nil {
			lastErr = res.err
			continue
		}
		ids = append(ids, res.ids...)
	}
	sort.Strings(ids)
	ids = uniqueStrings(ids)
	return ids, lastErr
}

func lookupSeriesByQuery(db *bbolt.DB, query chunk.IndexQuery) ([]string, error) {
	start := []byte(query.HashValue + separator + string(query.RangeValueStart))
	rowPrefix := []byte(query.HashValue + separator)
	var res []string
	var components [][]byte
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		for k, v := c.Seek(start); k != nil; k, v = c.Next() {
			// technically we can run regex that are not matching empty.
			if len(query.ValueEqual) > 0 && !bytes.Equal(v, query.ValueEqual) {
				continue
			}
			if !bytes.HasPrefix(k, rowPrefix) {
				break
			}
			// parse series ID and add to res
			_, r := decodeKey(k)

			components = decodeRangeKey(r, components)
			if len(components) != 4 {
				continue
			}
			// we store in label entries range keys: label hash value | seriesID | empty | type.
			// and we want the seriesID
			res = append(res, string(components[len(components)-3]))
		}
		return nil
	})

	return res, err
}

func schemaPeriodForTable(config storage.SchemaConfig, tableName string) (chunk.PeriodConfig, bool) {
	for _, schema := range config.Configs {
		periodIndex, err := strconv.ParseInt(strings.TrimPrefix(tableName, schema.IndexTables.Prefix), 10, 64)
		if err != nil {
			continue
		}
		periodSecs := int64((schema.IndexTables.Period) / time.Second)
		if periodIndex == schema.From.Time.Unix()/periodSecs {
			return schema, true
		}
	}
	return chunk.PeriodConfig{}, false
}

func seriesFromHash(h []byte) (seriesID []byte) {
	var index int
	for i := range h {
		if h[i] == ':' {
			index++
		}
		if index == 2 {
			seriesID = h[i+1:]
			return
		}
	}
	return
}

// decodeKey decodes hash and range value from a boltdb key.
func decodeKey(k []byte) (hashValue, rangeValue []byte) {
	// hashValue + 0 + string(rangeValue)
	for i := range k {
		if k[i] == 0 {
			hashValue = k[:i]
			rangeValue = k[i+1:]
			return
		}
	}
	return
}

func splitBytesBy(value []byte, by byte, components [][]byte) [][]byte {
	components = components[:0]
	i, j := 0, 0
	for j < len(value) {
		if value[j] != by {
			j++
			continue
		}
		components = append(components, value[i:j])
		j++
		i = j
	}
	components = append(components, value[i:])
	return components
}

func decodeRangeKey(value []byte, components [][]byte) [][]byte {
	components = components[:0]
	i, j := 0, 0
	for j < len(value) {
		if value[j] != 0 {
			j++
			continue
		}
		components = append(components, value[i:j])
		j++
		i = j
	}
	return components
}
