package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"math"

	"github.com/fireproofpenguin/loadship/internal/collector"
)

//go:embed template.html
var reportTemplate string

func Generate(data ReportData) ([]byte, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)

	if err != nil {
		return nil, fmt.Errorf("failed to parse report template: %w", err)
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	return buf.Bytes(), nil
}

type ReportData struct {
	Summary  collector.Metrics
	Metadata collector.TestConfig
}

func CreateReportData(json *collector.JSONOutput) ReportData {
	return ReportData{
		Summary:  sanitiseSummary(json.Summary),
		Metadata: json.Metadata,
	}
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func sanitiseSummary(summary collector.Metrics) collector.Metrics {
	summary.HTTPMetrics.Latency.Average = roundFloat(summary.HTTPMetrics.Latency.Average, 2)
	return summary
}
