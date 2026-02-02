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

	indicator := ""
	if math.Abs(m.Percent) > 5 {
		if m.Better {
			indicator = "✅"
		} else {
			indicator = "❌"
		}
	}

	deltaStr := fmt.Sprintf(m.Format, m.Delta)
	return fmt.Sprintf("%s%s (%.2f%%) %s", sign, deltaStr, m.Percent, indicator)
}

type ComparisonReport struct {
	HTTPChanges   []MetricChange
	DockerChanges []MetricChange
}

func (r *ComparisonReport) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Println("\n=== HTTP Metrics ===")
	fmt.Fprintln(w, "Metric\tBaseline\tTest\tChange")
	fmt.Fprintln(w, "------\t------\t------\t------")

	for _, change := range r.HTTPChanges {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", change.Name, change.BaselineString(), change.TestString(), change.ChangeString())
	}

	w.Flush()
}

func Compare(baseline, test *collector.JSONOutput) *ComparisonReport {
	return &ComparisonReport{
		HTTPChanges: []MetricChange{
			CalculateMetricChange("Total Requests", float64(baseline.Summary.HTTPMetrics.Requests.Total), float64(test.Summary.HTTPMetrics.Requests.Total), false, "%.0f"),
			CalculateMetricChange("Failed Requests", float64(baseline.Summary.HTTPMetrics.Requests.Failed), float64(test.Summary.HTTPMetrics.Requests.Failed), false, "%.0f"),
			CalculateMetricChange("RPS", baseline.Summary.HTTPMetrics.Requests.Rps, test.Summary.HTTPMetrics.Requests.Rps, false, "%.2f"),
			CalculateMetricChange("Latency (Avg)", float64(baseline.Summary.HTTPMetrics.Latency.Average), float64(test.Summary.HTTPMetrics.Latency.Average), true, "%.0f"),
			CalculateMetricChange("Latency (p50)", float64(baseline.Summary.HTTPMetrics.Latency.P50), float64(test.Summary.HTTPMetrics.Latency.P50), true, "%.0f"),
			CalculateMetricChange("Latency (p90)", float64(baseline.Summary.HTTPMetrics.Latency.P90), float64(test.Summary.HTTPMetrics.Latency.P90), true, "%.0f"),
			CalculateMetricChange("Latency (p95)", float64(baseline.Summary.HTTPMetrics.Latency.P95), float64(test.Summary.HTTPMetrics.Latency.P95), true, "%.0f"),
			CalculateMetricChange("Latency (p99)", float64(baseline.Summary.HTTPMetrics.Latency.P99), float64(test.Summary.HTTPMetrics.Latency.P99), true, "%.0f"),
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
