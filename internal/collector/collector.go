package collector

import (
	"fmt"
	"runtime"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
)

type RequestMetrics struct {
	total      int
	failed     int
	successful int
	rps        float64
}

type LatencyMetrics struct {
	average float64
	min     int64
	max     int64
	p50     int64
	p90     int64
	p95     int64
	p99     int64
}

type HTTPMetrics struct {
	requests RequestMetrics
	latency  LatencyMetrics
}

type MemoryMetrics struct {
	average float64
	min     float64
	max     float64
}

type DockerMetrics struct {
	collected bool
	memory    MemoryMetrics
}

type Metrics struct {
	HTTPMetrics   HTTPMetrics
	DockerMetrics DockerMetrics
}

func (m *Metrics) PrettyPrint() {
	fmt.Println("=== Request Metrics ===")
	fmt.Println("Total Requests:", m.HTTPMetrics.requests.total)
	fmt.Println("Successful Requests:", m.HTTPMetrics.requests.successful)
	fmt.Println("Failed Requests:", m.HTTPMetrics.requests.failed)
	fmt.Printf("Requests per Second: %.2f\n", m.HTTPMetrics.requests.rps)
	if m.HTTPMetrics.requests.successful == 0 {
		fmt.Println("------------- No successful requests -------------")
		fmt.Println("--- Be careful using the latency metrics below ---")
	}
	fmt.Printf("Latency Min/Avg/Max: %d / %.2f / %d ms\n", m.HTTPMetrics.latency.min, m.HTTPMetrics.latency.average, m.HTTPMetrics.latency.max)
	fmt.Printf("Latency p50/p90/p95/p99: %d / %d / %d / %d ms\n", m.HTTPMetrics.latency.p50, m.HTTPMetrics.latency.p90, m.HTTPMetrics.latency.p95, m.HTTPMetrics.latency.p99)
	if m.DockerMetrics.collected {
		fmt.Println("=== Docker Metrics ===")
		fmt.Printf("Average memory: %.2f MB\n", m.DockerMetrics.memory.average)
		fmt.Printf("Min memory: %.2f MB\n", m.DockerMetrics.memory.min)
		fmt.Printf("Max memory: %.2f MB\n", m.DockerMetrics.memory.max)
	}
}

func Calculate(httpStats []load.HTTPStats, dockerStats []docker.DockerStats, duration time.Duration) *Metrics {
	var metrics = &Metrics{}

	// HTTP metrics
	histogram := hdrhistogram.New(1, 60000, 3)

	totalRequests := len(httpStats)
	rps := float64(totalRequests) / duration.Seconds()
	var (
		successfulRequests int
		failedRequests     int
		totalLatency       float64
	)

	var minLatency, maxLatency time.Duration
	var latencyInitialised bool

	for _, result := range httpStats {
		if result.Err == nil && result.StatusCode >= 200 && result.StatusCode < 300 {
			successfulRequests++
			totalLatency += float64(result.Latency.Milliseconds())
			histogram.RecordValue(result.Latency.Milliseconds())
			if !latencyInitialised {
				minLatency = result.Latency
				maxLatency = result.Latency
				latencyInitialised = true
				continue
			}
			if result.Latency < minLatency {
				minLatency = result.Latency
			}
			if result.Latency > maxLatency {
				maxLatency = result.Latency
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

	metrics.HTTPMetrics = HTTPMetrics{
		requests: RequestMetrics{
			total:      totalRequests,
			failed:     failedRequests,
			successful: successfulRequests,
			rps:        rps,
		},
		latency: LatencyMetrics{
			average: averageLatency,
			min:     minLatency.Milliseconds(),
			max:     maxLatency.Milliseconds(),
			p50:     p50,
			p90:     p90,
			p95:     p95,
			p99:     p99,
		},
	}

	// Docker metrics
	if len(dockerStats) > 0 {
		if runtime.GOOS == "windows" {
			fmt.Println("Windows detected, cannot calculate CPU usage reliably - at this time!")
		}

		var (
			totalMemory float64
			minMemory   float64
			maxMemory   float64
		)

		minMemory = dockerStats[0].MemoryUsageMB
		maxMemory = dockerStats[0].MemoryUsageMB

		for _, result := range dockerStats {
			totalMemory += result.MemoryUsageMB
			if result.MemoryUsageMB < minMemory {
				minMemory = result.MemoryUsageMB
			}
			if result.MemoryUsageMB > maxMemory {
				maxMemory = result.MemoryUsageMB
			}
		}

		averageMemory := float64(totalMemory) / float64(len(dockerStats))

		metrics.DockerMetrics = DockerMetrics{
			collected: true,
			memory: MemoryMetrics{
				average: averageMemory,
				min:     minMemory,
				max:     maxMemory,
			},
		}
	}

	return metrics
}
