package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"slices"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
)

//go:embed template.html
var reportTemplate string

func Generate(data ReportData) ([]byte, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)

	if err != nil {
		return nil, fmt.Errorf("failed to parse report template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)

	if err != nil {
		return nil, fmt.Errorf("failed to execute report template: %w", err)
	}

	return buf.Bytes(), nil
}

type ReportData struct {
	Summary     collector.Metrics
	Metadata    collector.TestConfig
	Labels      []string
	RPS         []float64
	Errors      []float64
	Latency     []float64
	Memory      []float64
	CPU         []float64
	DiskReadMB  []float64
	DiskWriteMB []float64
}

func CreateReportData(json *collector.JSONOutput) ReportData {
	labels, rps, errors, latency := bucketHTTP(json.HTTPStats, json.Metadata.Timestamp)

	memory, cpu, diskReadMB, diskWriteMB := bucketDocker(json.DockerStats, json.Metadata.Timestamp)

	return ReportData{
		Summary:     sanitiseSummary(json.Summary),
		Metadata:    json.Metadata,
		Labels:      labels,
		RPS:         rps,
		Errors:      errors,
		Latency:     latency,
		Memory:      memory,
		CPU:         cpu,
		DiskReadMB:  diskReadMB,
		DiskWriteMB: diskWriteMB,
	}
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func sanitiseSummary(summary collector.Metrics) collector.Metrics {
	summary.HTTPMetrics.Latency.Average = roundFloat(summary.HTTPMetrics.Latency.Average, 2)
	summary.HTTPMetrics.Requests.Rps = roundFloat(summary.HTTPMetrics.Requests.Rps, 2)
	return summary
}

func bucketHTTP(stats []load.HTTPStats, testStart time.Time) ([]string, []float64, []float64, []float64) {
	type bucket struct {
		requests int
		errors   int
		latency  []int64
	}

	buckets := make(map[int64]*bucket)

	for _, s := range stats {
		second := int64(s.Timestamp.Sub(testStart).Seconds())
		if buckets[second] == nil {
			buckets[second] = &bucket{}
		}

		buckets[second].requests++
		if s.ErrorType != "" {
			buckets[second].errors++
		} else {
			buckets[second].latency = append(buckets[second].latency, s.Latency.Milliseconds())
		}
	}

	keys := make([]int64, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	labels := make([]string, len(keys))
	rps := make([]float64, len(keys))
	errors := make([]float64, len(keys))
	latency := make([]float64, len(keys))

	for i, k := range keys {
		labels[i] = fmt.Sprintf("%ds", k)
		rps[i] = float64(buckets[k].requests)
		errors[i] = float64(buckets[k].errors)
		var totalLatency int64
		for _, l := range buckets[k].latency {
			totalLatency += l
		}
		if len(buckets[k].latency) > 0 {
			latency[i] = roundFloat(float64(totalLatency)/float64(len(buckets[k].latency)), 2)
		} else {
			latency[i] = 0
		}
	}

	return labels, rps, errors, latency
}

func bucketDocker(stats []docker.DockerStats, testStart time.Time) ([]float64, []float64, []float64, []float64) {
	type bucket struct {
		memoryUsageMB float64
		cpuPercent    float64
		diskReadMB    float64
		diskWriteMB   float64
	}

	buckets := make(map[int64]*bucket)

	for _, s := range stats {
		second := int64(s.Timestamp.Sub(testStart).Seconds())

		buckets[second] = &bucket{
			memoryUsageMB: s.MemoryUsageMB,
			cpuPercent:    s.CPUPercent,
			diskReadMB:    s.DiskReadMB,
			diskWriteMB:   s.DiskWriteMB,
		}
	}

	keys := make([]int64, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	memoryUsage := make([]float64, len(keys))
	cpuPercent := make([]float64, len(keys))
	diskReadMB := make([]float64, len(keys))
	diskWriteMB := make([]float64, len(keys))

	var previousReadMB, previousWriteMB float64
	if len(keys) > 0 {
		previousReadMB = buckets[keys[0]].diskReadMB
		previousWriteMB = buckets[keys[0]].diskWriteMB
	}

	for i, k := range keys {
		memoryUsage[i] = buckets[k].memoryUsageMB
		cpuPercent[i] = roundFloat(buckets[k].cpuPercent, 2)
		diskReadMB[i] = buckets[k].diskReadMB - previousReadMB
		diskWriteMB[i] = buckets[k].diskWriteMB - previousWriteMB
		previousReadMB = buckets[k].diskReadMB
		previousWriteMB = buckets[k].diskWriteMB
	}

	return memoryUsage, cpuPercent, diskReadMB, diskWriteMB
}
