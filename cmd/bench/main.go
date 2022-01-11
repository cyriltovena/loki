package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/dustin/go-humanize"
	"github.com/grafana/loki/pkg/storage/chunk/gcp"
	"github.com/grafana/loki/pkg/storage/chunk/hedging"
	"go.uber.org/atomic"
)

// kubectl run -i --tty benchttp --image=golang --restart=Never
// GOOGLE_APPLICATION_CREDENTIALS=/go/bench/credentials.json go run main.go
func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:3100", nil))
	}()
	client, err := gcp.NewGCSObjectClient(context.Background(), gcp.GCSConfig{
		BucketName:       "ops-tools-tempo-dev",
		EnableOpenCensus: false,
		EnableHTTP2:      false,
	}, hedging.Config{})
	if err != nil {
		panic(err)
	}
	size := 2 * 1024 * 1024
	name := "test/2mb"
	createFakeChunkFile(client, name, size)
	defer client.DeleteObject(context.Background(), name)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	var (
		start     = time.Now()
		requests  = atomic.NewInt64(0)
		bytesRead = atomic.NewInt64(0)
		wg        sync.WaitGroup
	)
	go func() {
		<-sigs
		cancel()
	}()
	go func() {
		for ctx.Err() == nil {
			time.Sleep(time.Second)
			elapsed := time.Since(start).Seconds()
			log.Printf("RPS %f Throughput %s", float64(requests.Load())/elapsed, humanize.IBytes(uint64(float64(bytesRead.Load())/elapsed)))
		}
	}()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			// client, err := gcp.NewGCSObjectClient(context.Background(), gcp.GCSConfig{
			// 	BucketName:       "ops-tools-tempo-dev",
			// 	EnableOpenCensus: false,
			// }, hedging.Config{})
			// if err != nil {
			// 	panic(err)
			// }
			defer wg.Done()
			buffer := bytes.NewBuffer(make([]byte, 0, size))
			for ctx.Err() == nil {
				buffer.Reset()
				reader, _, err := client.GetObject(ctx, name)
				if err != nil {
					log.Fatal(fmt.Errorf("err Get:%w", err))
				}
				n, err := buffer.ReadFrom(reader)
				if err != nil {
					log.Fatal(fmt.Errorf("err Read:%w", err))
					reader.Close()
					continue
				}
				reader.Close()
				requests.Inc()
				bytesRead.Add(n)
			}
		}()
	}
	wg.Wait()
	log.Println("all goroutines done")
}

func createFakeChunkFile(client *gcp.GCSObjectClient, name string, size int) {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}
	if err := client.PutObject(context.Background(), name, bytes.NewReader(data)); err != nil {
		panic(err)
	}
}
