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
	PIDs   []MetricChange
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

// printMetricSection prints a table of metrics using the provided metric extractor function
func printMetricSection(w *tabwriter.Writer, reports []*ComparisonReport, getMetrics func(*ComparisonReport) []MetricChange) {
	if len(reports) == 0 {
		return
	}

	// Build header
	header := "Metric\tBaseline"
	separator := "------\t--------"
	for i := range reports {
		header += fmt.Sprintf("\tTest %d (Change)", i+1)
		separator += "\t---------------"
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, separator)

	// Build lookup maps for each report for O(1) access
	reportMaps := make([]map[string]MetricChange, len(reports))
	for i, report := range reports {
		reportMaps[i] = make(map[string]MetricChange)
		for _, metric := range getMetrics(report) {
			reportMaps[i][metric.Name] = metric
		}
	}

	// Get all unique metric names from the first report
	firstReportMetrics := getMetrics(reports[0])
	for _, metric := range firstReportMetrics {
		metricName := metric.Name
		row := fmt.Sprintf("%s\t%s", metricName, metric.BaselineString())

		// Add test results for this metric
		for _, reportMap := range reportMaps {
			if change, exists := reportMap[metricName]; exists {
				row += fmt.Sprintf("\t%s (%s)", change.TestString(), change.ChangeString())
			} else {
				row += "\tn/a (n/a)"
			}
		}
		fmt.Fprintln(w, row)
	}
}

// hasDockerMetrics checks if any report contains Docker metrics
func hasDockerMetrics(reports []*ComparisonReport) bool {
	for _, report := range reports {
		if len(report.DockerChanges.Memory) > 0 ||
			len(report.DockerChanges.CPU) > 0 ||
			len(report.DockerChanges.DiskIO) > 0 {
			return true
		}
	}
	return false
}

func Compare(outputs []*collector.JSONOutput) []*ComparisonReport {
	baseline := outputs[0]
	tests := outputs[1:]

	var reports []*ComparisonReport

	for _, test := range tests {
		baselineHasDockerMetrics := len(baseline.DockerStats) > 0
		testHasDockerMetrics := len(test.DockerStats) > 0

		var memoryChanges, cpuChanges, diskIOChanges, pidChanges []MetricChange

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
			pidChanges = []MetricChange{
				CalculateMetricChange("Average PIDs", baseline.Summary.DockerMetrics.PIDs.Average, test.Summary.DockerMetrics.PIDs.Average, true, "%.2f"),
				CalculateMetricChange("Peak PIDs", baseline.Summary.DockerMetrics.PIDs.Peak, test.Summary.DockerMetrics.PIDs.Peak, true, "%.0f"),
			}
		} else if baselineHasDockerMetrics || testHasDockerMetrics {
			fmt.Println("Warning: Only one of the test results contains Docker metrics. Docker metrics will be skipped in the comparison.")
		}

		report := &ComparisonReport{
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
				PIDs:   pidChanges,
			},
		}

		reports = append(reports, report)
	}

	return reports
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

func PrintComparisonReports(baseline *collector.JSONOutput, reports []*ComparisonReport) {
	if len(reports) == 0 {
		fmt.Println("No comparison reports to display")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	// Print HTTP metrics
	fmt.Println("\n=== HTTP Metrics ===")
	printMetricSection(w, reports, func(r *ComparisonReport) []MetricChange { return r.HTTPChanges })

	// Print Docker metrics if available
	if hasDockerMetrics(reports) {
		fmt.Fprintln(w, "\n=== Docker Metrics ===")
		fmt.Fprintln(w, "Memory")
		printMetricSection(w, reports, func(r *ComparisonReport) []MetricChange { return r.DockerChanges.Memory })

		fmt.Fprintln(w)
		fmt.Fprintln(w, "CPU")
		printMetricSection(w, reports, func(r *ComparisonReport) []MetricChange { return r.DockerChanges.CPU })

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Disk I/O")
		printMetricSection(w, reports, func(r *ComparisonReport) []MetricChange { return r.DockerChanges.DiskIO })

		fmt.Fprintln(w)
		fmt.Fprintln(w, "PIDs")
		printMetricSection(w, reports, func(r *ComparisonReport) []MetricChange { return r.DockerChanges.PIDs })
	}

	w.Flush()
}
