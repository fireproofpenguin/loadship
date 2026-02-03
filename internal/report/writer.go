package report

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fireproofpenguin/loadship/internal/collector"
)

func Write(json *collector.JSONOutput, reportName string) {
	reportData := CreateReportData(json)

	reportBytes, err := Generate(reportData)

	if err != nil {
		log.Fatalf("Error generating report: %v\n", err)
	}

	outputPath, err := filepath.Abs(fmt.Sprintf("%s.html", reportName))

	if err != nil {
		fmt.Println("Error determining absolute path for report:", err)
		return
	}

	err = os.WriteFile(outputPath, reportBytes, 0644)

	if err != nil {
		fmt.Println("Error writing report:", err)
		return
	}

	fmt.Printf("\n✓ Report saved to %s\n", outputPath)
}
