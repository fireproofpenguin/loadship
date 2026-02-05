package comparison

import (
	"fmt"
	"math"
	"os"
	"text/tabwriter"

	"github.com/fireproofpenguin/loadship/internal/collector"
)

type MetricChange struct {
	Name     string
	Baseline float64
	Test     float64
	Delta    float64
	Percent  float64
	Better   bool
	Format   string // "%.2f", "%.0f", "%d", etc.
}

func (m MetricChange) BaselineString() string {
	return fmt.Sprintf(m.Format, m.Baseline)
}

func (m MetricChange) TestString() string {
	return fmt.Sprintf(m.Format, m.Test)
}

func (m MetricChange) ChangeString() string {
	sign := ""
	if m.Delta > 0 {
		sign = "+"
	}

	percentStr := fmt.Sprintf("%.2f%%", m.Percent)
	indicator := ""
	if m.Baseline == 0 {
		percentStr = "n/a"
	} else if math.Abs(m.Percent) > 5 {
		if m.Better {
			indicator = "✓"
		} else {
			indicator = "✗"
		}
	}

	deltaStr := fmt.Sprintf(m.Format, m.Delta)
	return fmt.Sprintf("%s%s (%s) %s", sign, deltaStr, percentStr, indicator)
}

type DockerChanges struct {
	Memory []MetricChange
	CPU    []MetricChange
	DiskIO []MetricChange
}

type ComparisonReport struct {
	HTTPChanges   []MetricChange
	DockerChanges DockerChanges
}

func (r *ComparisonReport) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Println("\n=== HTTP Metrics ===")
	fmt.Fprintln(w, "Metric\tBaseline\tTest\tChange")
	fmt.Fprintln(w, "------\t------\t------\t------")

	for _, change := range r.HTTPChanges {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", change.Name, change.BaselineString(), change.TestString(), change.ChangeString())
	}

	if len(r.DockerChanges.Memory) > 0 {
		fmt.Fprintln(w, "\n=== Docker Metrics ===")

		fmt.Fprintln(w, "Memory\tBaseline\tTest\tChange")
		fmt.Fprintln(w, "------\t------\t------\t------")
		for _, change := range r.DockerChanges.Memory {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", change.Name, change.BaselineString(), change.TestString(), change.ChangeString())
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "CPU\tBaseline\tTest\tChange")
		fmt.Fprintln(w, "------\t------\t------\t------")
		for _, change := range r.DockerChanges.CPU {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", change.Name, change.BaselineString(), change.TestString(), change.ChangeString())
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Disk Op\tBaseline\tTest\tChange")
		fmt.Fprintln(w, "------\t------\t------\t------")
		for _, change := range r.DockerChanges.DiskIO {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", change.Name, change.BaselineString(), change.TestString(), change.ChangeString())
		}
	}

	w.Flush()
}

func Compare(baseline, test *collector.JSONOutput) *ComparisonReport {
	baselineHasDockerMetrics := len(baseline.DockerStats) > 0
	testHasDockerMetrics := len(test.DockerStats) > 0

	var memoryChanges, cpuChanges, diskIOChanges []MetricChange

	if baselineHasDockerMetrics && testHasDockerMetrics {
		memoryChanges = []MetricChange{
			CalculateMetricChange("Average Memory (MB)", baseline.Summary.DockerMetrics.Memory.Average, test.Summary.DockerMetrics.Memory.Average, true, "%.2f"),
			CalculateMetricChange("Min Memory (MB)", baseline.Summary.DockerMetrics.Memory.Min, test.Summary.DockerMetrics.Memory.Min, true, "%.2f"),
			CalculateMetricChange("Max Memory (MB)", baseline.Summary.DockerMetrics.Memory.Max, test.Summary.DockerMetrics.Memory.Max, true, "%.2f"),
		}
		cpuChanges = []MetricChange{
			CalculateMetricChange("Average CPU (%)", baseline.Summary.DockerMetrics.CPU.Average, test.Summary.DockerMetrics.CPU.Average, true, "%.2f"),
			CalculateMetricChange("Peak CPU (%)", baseline.Summary.DockerMetrics.CPU.Peak, test.Summary.DockerMetrics.CPU.Peak, true, "%.2f"),
		}
		diskIOChanges = []MetricChange{
			CalculateMetricChange("Read (MB)", baseline.Summary.DockerMetrics.DiskIO.ReadMB, test.Summary.DockerMetrics.DiskIO.ReadMB, true, "%.2f"),
			CalculateMetricChange("Write (MB)", baseline.Summary.DockerMetrics.DiskIO.WriteMB, test.Summary.DockerMetrics.DiskIO.WriteMB, true, "%.2f"),
		}
	} else if baselineHasDockerMetrics || testHasDockerMetrics {
		fmt.Println("Warning: Only one of the test results contains Docker metrics. Docker metrics will be skipped in the comparison.")
	}

	return &ComparisonReport{
		HTTPChanges: []MetricChange{
			CalculateMetricChange("Total Requests", float64(baseline.Summary.HTTPMetrics.Requests.Total), float64(test.Summary.HTTPMetrics.Requests.Total), false, "%.0f"),
			CalculateMetricChange("Failed Requests", float64(baseline.Summary.HTTPMetrics.Requests.Failed), float64(test.Summary.HTTPMetrics.Requests.Failed), true, "%.0f"),
			CalculateMetricChange("RPS", baseline.Summary.HTTPMetrics.Requests.Rps, test.Summary.HTTPMetrics.Requests.Rps, false, "%.2f"),
			CalculateMetricChange("Latency (Avg)", float64(baseline.Summary.HTTPMetrics.Latency.Average), float64(test.Summary.HTTPMetrics.Latency.Average), true, "%.0f"),
			CalculateMetricChange("Latency (p50)", float64(baseline.Summary.HTTPMetrics.Latency.P50), float64(test.Summary.HTTPMetrics.Latency.P50), true, "%.0f"),
			CalculateMetricChange("Latency (p90)", float64(baseline.Summary.HTTPMetrics.Latency.P90), float64(test.Summary.HTTPMetrics.Latency.P90), true, "%.0f"),
			CalculateMetricChange("Latency (p95)", float64(baseline.Summary.HTTPMetrics.Latency.P95), float64(test.Summary.HTTPMetrics.Latency.P95), true, "%.0f"),
			CalculateMetricChange("Latency (p99)", float64(baseline.Summary.HTTPMetrics.Latency.P99), float64(test.Summary.HTTPMetrics.Latency.P99), true, "%.0f"),
		},
		DockerChanges: DockerChanges{
			Memory: memoryChanges,
			CPU:    cpuChanges,
			DiskIO: diskIOChanges,
		},
	}
}

func CalculateMetricChange(name string, baseline, test float64, lowerIsBetter bool, format string) MetricChange {
	delta := test - baseline
	percent := 0.0

	if baseline != 0 {
		percent = (delta / baseline) * 100
	}

	better := false
	if lowerIsBetter {
		better = delta < 0
	} else {
		better = delta > 0
	}

	return MetricChange{
		Name:     name,
		Baseline: baseline,
		Test:     test,
		Delta:    delta,
		Percent:  percent,
		Better:   better,
		Format:   format,
	}
}
