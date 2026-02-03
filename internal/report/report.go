package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

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
	Summary collector.Metrics
}

func CreateReportData(json *collector.JSONOutput) ReportData {
	return ReportData{
		Summary: json.Summary,
	}
}
