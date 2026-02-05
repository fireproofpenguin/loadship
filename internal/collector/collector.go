package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
)

type RequestMetrics struct {
	Total      int     `json:"total"`
	Failed     int     `json:"failed"`
	Successful int     `json:"successful"`
	Rps        float64 `json:"rps"`
}

type LatencyMetrics struct {
	Average float64 `json:"average"`
	Min     int64   `json:"min"`
	Max     int64   `json:"max"`
	P50     int64   `json:"p50"`
	P90     int64   `json:"p90"`
	P95     int64   `json:"p95"`
	P99     int64   `json:"p99"`
}

type HTTPMetrics struct {
	Requests RequestMetrics `json:"requests"`
	Latency  LatencyMetrics `json:"latency"`
}

type MemoryMetrics struct {
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

type CPUMetrics struct {
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
}

type DiskIOMetrics struct {
	ReadMB  float64 `json:"disk_read_mb"`
	WriteMB float64 `json:"disk_write_mb"`
}

type PIDMetrics struct {
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
}

type DockerMetrics struct {
	collected bool
	Memory    MemoryMetrics `json:"memory,omitempty"`
	CPU       CPUMetrics    `json:"cpu,omitempty"`
	DiskIO    DiskIOMetrics `json:"disk_io,omitempty"`
	PIDs      PIDMetrics    `json:"pids,omitempty"`
}

type Metrics struct {
	HTTPMetrics   HTTPMetrics   `json:"http_metrics"`
	DockerMetrics DockerMetrics `json:"docker_metrics,omitempty"`
}

func (m *Metrics) PrettyPrint() {
	fmt.Println("=== Request Metrics ===")
	fmt.Println("Total Requests:", m.HTTPMetrics.Requests.Total)
	fmt.Println("Successful Requests:", m.HTTPMetrics.Requests.Successful)
	fmt.Println("Failed Requests:", m.HTTPMetrics.Requests.Failed)
	fmt.Printf("Requests per Second: %.2f\n", m.HTTPMetrics.Requests.Rps)
	if m.HTTPMetrics.Requests.Successful == 0 {
		fmt.Println("------------- No successful requests -------------")
		fmt.Println("--- Be careful using the latency metrics below ---")
	}
	fmt.Printf("Latency Min/Avg/Max: %d / %.2f / %d ms\n", m.HTTPMetrics.Latency.Min, m.HTTPMetrics.Latency.Average, m.HTTPMetrics.Latency.Max)
	fmt.Printf("Latency p50/p90/p95/p99: %d / %d / %d / %d ms\n", m.HTTPMetrics.Latency.P50, m.HTTPMetrics.Latency.P90, m.HTTPMetrics.Latency.P95, m.HTTPMetrics.Latency.P99)
	if m.DockerMetrics.collected {
		fmt.Println("=== Docker Metrics ===")
		fmt.Printf("Average memory: %.2f MB\n", m.DockerMetrics.Memory.Average)
		fmt.Printf("Min memory: %.2f MB\n", m.DockerMetrics.Memory.Min)
		fmt.Printf("Max memory: %.2f MB\n", m.DockerMetrics.Memory.Max)
		fmt.Printf("CPU:\tAverage: %.2f %%\tPeak: %.2f %%\n", m.DockerMetrics.CPU.Average, m.DockerMetrics.CPU.Peak)
		fmt.Printf("DiskIO:\tRead: %.2f MB\tWrite: %.2f MB\n", m.DockerMetrics.DiskIO.ReadMB, m.DockerMetrics.DiskIO.WriteMB)
		fmt.Printf("PIDs:\tAverage: %.0f\tPeak: %.0f\n", m.DockerMetrics.PIDs.Average, m.DockerMetrics.PIDs.Peak)
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
		if result.ErrorType == "" && result.StatusCode >= 200 && result.StatusCode < 300 {
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
		Requests: RequestMetrics{
			Total:      totalRequests,
			Failed:     failedRequests,
			Successful: successfulRequests,
			Rps:        rps,
		},
		Latency: LatencyMetrics{
			Average: averageLatency,
			Min:     minLatency.Milliseconds(),
			Max:     maxLatency.Milliseconds(),
			P50:     p50,
			P90:     p90,
			P95:     p95,
			P99:     p99,
		},
	}

	// Docker metrics
	if len(dockerStats) > 0 {
		if runtime.GOOS == "windows" {
			fmt.Println("Windows detected, cannot calculate CPU / DiskIO usage reliably - at this time!")
		}

		var (
			totalMemory float64
			minMemory   float64
			maxMemory   float64
			totalCPU    float64
			peakCPU     float64
			totalPids   float64
			peakPids    float64
		)

		minMemory = dockerStats[0].MemoryUsageMB
		maxMemory = dockerStats[0].MemoryUsageMB

		baselineRead := dockerStats[0].DiskReadMB
		baselineWrite := dockerStats[0].DiskWriteMB

		totalWrite := dockerStats[len(dockerStats)-1].DiskWriteMB - baselineWrite
		totalRead := dockerStats[len(dockerStats)-1].DiskReadMB - baselineRead

		for _, result := range dockerStats {
			totalMemory += result.MemoryUsageMB
			if result.MemoryUsageMB < minMemory {
				minMemory = result.MemoryUsageMB
			}
			if result.MemoryUsageMB > maxMemory {
				maxMemory = result.MemoryUsageMB
			}
			totalCPU += result.CPUPercent
			if result.CPUPercent > peakCPU {
				peakCPU = result.CPUPercent
			}
			totalPids += float64(result.PIDs)
			if float64(result.PIDs) > peakPids {
				peakPids = float64(result.PIDs)
			}
		}

		averageMemory := float64(totalMemory) / float64(len(dockerStats))
		averageCPU := float64(totalCPU) / float64(len(dockerStats))
		averagePids := totalPids / float64(len(dockerStats))

		metrics.DockerMetrics = DockerMetrics{
			collected: true,
			Memory: MemoryMetrics{
				Average: averageMemory,
				Min:     minMemory,
				Max:     maxMemory,
			},
			CPU: CPUMetrics{
				Average: averageCPU,
				Peak:    peakCPU,
			},
			DiskIO: DiskIOMetrics{
				ReadMB:  totalRead,
				WriteMB: totalWrite,
			},
			PIDs: PIDMetrics{
				Average: averagePids,
				Peak:    peakPids,
			},
		}
	}

	return metrics
}

type JSONOutput struct {
	Metadata    TestConfig           `json:"metadata"`
	HTTPStats   []load.HTTPStats     `json:"http_stats"`
	DockerStats []docker.DockerStats `json:"docker_stats,omitempty"`
	Summary     Metrics              `json:"summary"`
}

func (jo *JSONOutput) SaveToFile(filename string) error {
	data, err := json.Marshal(jo)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

type TestConfig struct {
	Timestamp     time.Time     `json:"timestamp"`
	URL           string        `json:"url"`
	Duration      time.Duration `json:"duration"`
	Connections   int           `json:"connections"`
	ContainerName string        `json:"container_name,omitempty"`
}

func (tc *TestConfig) IsSimilar(other TestConfig) bool {
	if tc.URL != other.URL {
		return false
	}
	if tc.Connections != other.Connections {
		return false
	}
	if tc.Duration != other.Duration {
		return false
	}
	return true
}

func ToJSONOutput(httpStats []load.HTTPStats, dockerStats []docker.DockerStats, config TestConfig, metrics Metrics) JSONOutput {
	return JSONOutput{
		Metadata:    config,
		HTTPStats:   httpStats,
		DockerStats: dockerStats,
		Summary:     metrics,
	}
}

func ReadFromJSON(data []byte) (*JSONOutput, error) {
	var output JSONOutput
	err := json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}
