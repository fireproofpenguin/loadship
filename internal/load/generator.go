package load

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

func Run(url string, duration time.Duration) error {
	fmt.Println("Run load test against:", url, "for duration:", duration)

	defaultTimeout := 30 * time.Second

	startTime := time.Now()

	totalRequests := 0
	successfulRequests := 0
	failedRequests := 0
	latencies := []time.Duration{}

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	for time.Since(startTime) < duration {
		reqStart := time.Now()
		resp, err := client.Get(url)

		totalRequests += 1

		if err != nil {
			failedRequests += 1
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		latency := time.Since(reqStart)
		latencies = append(latencies, latency)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successfulRequests += 1
		} else {
			failedRequests += 1
		}
	}

	rps := float64(totalRequests) / duration.Seconds()
	var totalLatency float64
	for _, latency := range latencies {
		totalLatency += float64(latency.Milliseconds())
	}
	averageLatency := totalLatency / float64(len(latencies))
	var minLatency, maxLatency time.Duration
	if len(latencies) > 0 {
		minLatency = slices.Min(latencies)
		maxLatency = slices.Max(latencies)
	}

	fmt.Println("Total Requests:", totalRequests)
	fmt.Println("Successful Requests:", successfulRequests)
	fmt.Println("Failed Requests:", failedRequests)
	fmt.Printf("Requests per Second: %.2f\n", rps)
	fmt.Printf("Latency Min/Avg/Max: %d / %.2f / %d ms\n", minLatency.Milliseconds(), averageLatency, maxLatency.Milliseconds())

	return nil
}
