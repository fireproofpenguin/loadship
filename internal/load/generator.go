package load

import (
	"fmt"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

type requestResult struct {
	latency    time.Duration
	err        error
	statusCode int
}

func Run(url string, duration time.Duration, connections int) error {
	fmt.Println("Run load test against:", url, "for duration:", duration)

	startTime := time.Now()

	ch := make(chan []requestResult)

	var wg sync.WaitGroup

	for i := range connections {
		wg.Go(func() {
			RequestLoop(i, url, startTime, duration, ch)
		})
	}

	var results []requestResult

	wg.Go(func() {
		for range connections {
			worker_results := <-ch
			results = append(results, worker_results...)
		}
	})

	wg.Wait()

	close(ch)

	histogram := hdrhistogram.New(1, 60000, 3)

	totalRequests := len(results)
	rps := float64(totalRequests) / duration.Seconds()
	var (
		successfulRequests int
		failedRequests     int
		totalLatency       float64
	)

	var minLatency, maxLatency time.Duration

	if len(results) == 0 {
		fmt.Println("No requests were made.")
		return nil
	}

	minLatency = results[0].latency
	maxLatency = results[0].latency

	for _, result := range results {
		if result.err == nil && result.statusCode >= 200 && result.statusCode < 300 {
			successfulRequests++
			totalLatency += float64(result.latency.Milliseconds())
			histogram.RecordValue(result.latency.Milliseconds())
			if result.latency < minLatency {
				minLatency = result.latency
			}
			if result.latency > maxLatency {
				maxLatency = result.latency
			}
		} else {
			failedRequests++
		}
	}

	var averageLatency float64
	if successfulRequests > 0 {
		averageLatency = totalLatency / float64(successfulRequests)
	}

	p50 := histogram.ValueAtQuantile(50.0)
	p90 := histogram.ValueAtQuantile(90.0)
	p95 := histogram.ValueAtQuantile(95.0)
	p99 := histogram.ValueAtQuantile(99.0)

	fmt.Println("Total Requests:", totalRequests)
	fmt.Println("Successful Requests:", successfulRequests)
	fmt.Println("Failed Requests:", failedRequests)
	fmt.Printf("Requests per Second: %.2f\n", rps)
	if successfulRequests == 0 {
		fmt.Println("------------- No successful requests -------------")
		fmt.Println("--- Be careful using the latency metrics below ---")
	}
	fmt.Printf("Latency Min/Avg/Max: %d / %.2f / %d ms\n", minLatency.Milliseconds(), averageLatency, maxLatency.Milliseconds())
	fmt.Printf("Latency p50/p90/p95/p99: %d / %d / %d / %d ms\n", p50, p90, p95, p99)

	return nil
}
