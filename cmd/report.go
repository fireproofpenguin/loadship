package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/report"
	"github.com/spf13/cobra"
)

var reportName string

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate an HTML report",
	Long: `Generate an HTML report using 1 or more JSON files from previous runs.
	
	loadship report baseline.json new_build.json`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatalf("Must provide at least 1 JSON file")
		}

		for _, arg := range args {
			filePath, err := filepath.Abs(arg)

			if err != nil {
				fmt.Println("Error getting absolute path:", err)
				return
			}

			if filepath.Ext(filePath) != ".json" {
				log.Fatalf("Must provide results in JSON format only")
			}

			b, err := os.ReadFile(filePath)

			if err != nil {
				log.Fatalf("Error reading file: %v\n", err)
			}

			output, err := collector.ReadFromJSON(b)

			if err != nil {
				log.Fatalf("Error parsing JSON from file: %v\n", err)
			}

			output.Summary.PrettyPrint()
		}

		reportBytes, err := report.Generate()

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
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&reportName, "report", "r", "report", "Name for the report")
}
