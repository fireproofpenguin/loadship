package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed template.html
var reportTemplate string

func Generate() ([]byte, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)

	if err != nil {
		return nil, fmt.Errorf("failed to parse report template: %w", err)
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, nil)
	return buf.Bytes(), nil
}
