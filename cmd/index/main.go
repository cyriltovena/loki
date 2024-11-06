package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/grafana/loki/v3/pkg/storage/stores/shipper/indexshipper/tsdb/index"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "file", "", "Path to TSDB file")
	flag.Parse()

	if filePath == "" {
		fmt.Println("Error: Please provide a TSDB file path using -file flag")
		os.Exit(1)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File %s does not exist\n", filePath)
		os.Exit(1)
	}

	// Open the TSDB file
	reader, err := index.NewFileReader(filePath)
	if err != nil {
		fmt.Printf("Error opening TSDB file: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	// Print basic file information
	fmt.Printf("File: %s\n", filepath.Base(filePath))
	printIndexStats(reader)

	// Print TSDB size for each 6-hour range
	// _, maxTime := reader.Bounds()
	// sixHours := int64(6 * 60 * 60 * 1000) // 6 hours in milliseconds
	startTime := time.Date(2024, 10, 30, 7, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 10, 30, 19, 0, 0, 0, time.UTC).UnixMilli()
	shards := []int{16, 32, 64, 128}
	for _, numShards := range shards {
		printTsdbSizeRange(reader, startTime, endTime, numShards)
		fmt.Printf("\nPress Enter to continue...")
		fmt.Scanln()
	}

	namesCount := map[string]int64{}
	names, err := reader.LabelNames()
	if err != nil {
		fmt.Printf("Error getting label names: %v\n", err)
		os.Exit(1)
	}
	for _, name := range names {
		values, err := reader.LabelValues(name)
		if err != nil {
			fmt.Printf("Error getting label values: %v\n", err)
			os.Exit(1)
		}
		namesCount[name] = int64(len(values))
	}
	// Create slice of name/count pairs for sorting
	type labelCount struct {
		name  string
		count int64
	}
	counts := make([]labelCount, 0, len(namesCount))
	for name, count := range namesCount {
		counts = append(counts, labelCount{name, count})
	}

	// Sort by count descending
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	// Create and configure a new tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print table header
	fmt.Fprintln(w, "Label Name\tUnique Values\t")
	fmt.Fprintln(w, "----------\t-------------\t")

	// Print each row
	for _, lc := range counts {
		fmt.Fprintf(w, "%s\t%d\t\n", lc.name, lc.count)
	}

	// Flush the writer to output the formatted table
	w.Flush()
}

func printIndexStats(reader *index.Reader) {
	fmt.Println("----------------------------------------")
	fmt.Printf("Index Size: %s\n", formatBytes(int64(reader.Size())))
	fmt.Printf("Symbols Size: %s\n", formatBytes(int64(reader.SymbolTableSize())))
	fmt.Printf("Series Size: %s\n", formatBytes(int64(reader.SeriesSize())))
	fmt.Printf("Series Labels Size: %s\n", formatBytes(int64(reader.SeriesLabelsSize())))
	fmt.Printf("Series Chunks Size: %s\n", formatBytes(int64(reader.SeriesChunksSize())))
	fmt.Printf("Label Indices Table Size: %s\n", formatBytes(int64(reader.LabelIndicesTableSize())))
	fmt.Printf("Label Indices Size: %s\n", formatBytes(int64(reader.LabelIndicesSize())))
	fmt.Printf("Postings Table Size: %s\n", formatBytes(int64(reader.PostingsTableSize())))
	fmt.Printf("Postings Size: %s\n", formatBytes(int64(reader.PostingsSize())))
	// Get chunks information
	minTime, maxTime := reader.Bounds()
	k, v := index.AllPostingsKey()
	var (
		ls          labels.Labels
		chks        []index.ChunkMeta
		totalSeries int
		totalChunks int
	)
	postings, err := reader.Postings(k, nil, v)
	if err != nil {
		fmt.Printf("Error getting postings: %v\n", err)
		os.Exit(1)
	}
	for postings.Next() {
		currentPosting := postings.At()
		_, err := reader.Series(currentPosting, 0, math.MaxInt64, &ls, &chks)
		if err != nil {
			fmt.Printf("Error getting series: %v\n", err)
			os.Exit(1)
		}
		totalSeries++
		totalChunks += len(chks)
	}
	if err := postings.Err(); err != nil {
		fmt.Printf("Error reading postings: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total Series: %d\n", totalSeries)
	fmt.Printf("Total Chunks: %d\n", totalChunks)
	fmt.Printf("Time Range: %s to %s\n",
		time.Unix(minTime/1000, 0).UTC().Format(time.RFC3339),
		time.Unix(maxTime/1000, 0).UTC().Format(time.RFC3339))
	fmt.Println("----------------------------------------")
}

func printTsdbSizeRange(reader *index.Reader, from int64, to int64, numShards int) {
	// Create a map to hold writers for each shard
	writers := make(map[model.Fingerprint]*index.Writer)
	tmpFiles := make(map[model.Fingerprint]*os.File)
	defer func() {
		// Close and cleanup all temp files
		for _, w := range writers {
			w.Close()
		}
		for _, f := range tmpFiles {
			os.Remove(f.Name())
		}
	}()

	// Copy series with chunks using postings
	allPostingsName, allPostingsValue := index.AllPostingsKey()
	postings, err := reader.Postings(allPostingsName, nil, allPostingsValue)
	if err != nil {
		fmt.Printf("Error getting all postings: %v\n", err)
		os.Exit(1)
	}

	var labels labels.Labels
	var chks []index.ChunkMeta
	for postings.Next() {
		ref := postings.At()
		fp, err := reader.Series(ref, from, to, &labels, &chks)
		if err != nil {
			fmt.Printf("Error reading series: %v\n", err)
			os.Exit(1)
		}

		// Filter chunks within time range
		filteredChks := make([]index.ChunkMeta, 0, len(chks))
		for _, chk := range chks {
			if chk.MaxTime >= from && chk.MinTime <= to {
				filteredChks = append(filteredChks, chk)
			}
		}

		if len(filteredChks) > 0 {
			// Get or create writer for this fingerprint's shard
			fingerprint := model.Fingerprint(fp)
			shardNum := fingerprint % model.Fingerprint(numShards)
			writer, exists := writers[shardNum]
			if !exists {
				// Create new temp file and writer for this shard
				tmpFile, err := os.CreateTemp("", fmt.Sprintf("series-with-chunks-shard-%d", shardNum))
				if err != nil {
					fmt.Printf("Error creating temp file for shard %d: %v\n", shardNum, err)
					os.Exit(1)
				}
				tmpFiles[shardNum] = tmpFile

				writer, err = index.NewWriter(context.Background(), reader.Version(), tmpFile.Name())
				if err != nil {
					fmt.Printf("Error creating writer for shard %d: %v\n", shardNum, err)
					os.Exit(1)
				}
				writers[shardNum] = writer

				// Copy symbols to new shard
				symbolsiter := reader.Symbols()
				for symbolsiter.Next() {
					if err := writer.AddSymbol(symbolsiter.At()); err != nil {
						fmt.Printf("Error adding symbol to shard %d: %v\n", shardNum, err)
						os.Exit(1)
					}
				}
				if err := symbolsiter.Err(); err != nil {
					fmt.Printf("Error reading symbols for shard %d: %v\n", shardNum, err)
					os.Exit(1)
				}
			}

			if err := writer.AddSeries(ref, labels, fingerprint, filteredChks...); err != nil {
				fmt.Printf("Error adding series to shard %d: %v\n", shardNum, err)
				os.Exit(1)
			}
		}
	}
	if err := postings.Err(); err != nil {
		fmt.Printf("Error iterating postings: %v\n", err)
		os.Exit(1)
	}

	// Close all writers
	for shardNum, writer := range writers {
		if err := writer.Close(); err != nil {
			fmt.Printf("Error closing writer for shard %d: %v\n", shardNum, err)
			os.Exit(1)
		}
	}

	// Print stats for each shard
	fmt.Printf("Series size from %s to %s\n",
		time.Unix(from/1000, 0).Format("2006-01-02 15:04:05"),
		time.Unix(to/1000, 0).Format("2006-01-02 15:04:05"))

	for shardNum, tmpFile := range tmpFiles {
		newReader, err := index.NewFileReader(tmpFile.Name())
		if err != nil {
			fmt.Printf("Error opening TSDB file for shard %d: %v\n", shardNum, err)
			os.Exit(1)
		}
		fmt.Printf("\nShard %d of %d stats:\n", shardNum+1, numShards)
		printIndexStats(newReader)
		newReader.Close()
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
