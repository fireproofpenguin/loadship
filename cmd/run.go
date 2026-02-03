package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/orchestrator"
	"github.com/fireproofpenguin/loadship/internal/report"
	"github.com/spf13/cobra"
)

var (
	duration       time.Duration
	connections    int
	containerName  string
	jsonFile       string
	generateReport bool
)

var runCmd = &cobra.Command{
	Use:   "run <target-url>",
	Short: "Run load tests against a target service",
	Long:  `Run a load test against a service, with or without docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if generateReport && jsonFile == "" {
			return fmt.Errorf("must specify --json when using --report")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if connections <= 0 {
			fmt.Println("Must have at least one connection")
			return
		}

		url := args[0]
		testStart := time.Now()

		config := collector.TestConfig{
			URL:           url,
			Timestamp:     testStart,
			Duration:      duration,
			Connections:   connections,
			ContainerName: containerName,
		}

		httpResults, dockerResults, err := orchestrator.Orchestrate(config)

		if err != nil {
			log.Fatalf("Error during test orchestration: %v", err)
		}

		fmt.Printf("\nLoad test complete. Processing results...\n")

		metrics := collector.Calculate(httpResults, dockerResults, duration)
		metrics.PrettyPrint()

		if jsonFile != "" {
			metricsOutput := collector.ToJSONOutput(httpResults, dockerResults, config, *metrics)

			metricsJSON, err := json.Marshal(metricsOutput)

			if err != nil {
				fmt.Println("Error converting metrics to JSON:", err)
				return
			}

			outputPath, err := filepath.Abs(jsonFile)

			if err != nil {
				fmt.Println("Error determining absolute path for JSON file:", err)
				return
			}

			err = os.WriteFile(outputPath, metricsJSON, 0644)

			if err != nil {
				fmt.Println("Error writing JSON file:", err)
				return
			}

			fmt.Printf("\n✓ Results saved to %s\n", outputPath)

			if generateReport {
				reportName := strings.TrimSuffix(jsonFile, ".json")

				report.Write(&metricsOutput, reportName)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().DurationVarP(&duration, "duration", "d", time.Second*30, "Duration of the load test (e.g., 10s, 1m)")
	runCmd.Flags().StringVar(&containerName, "container", "", "Docker container name or id to monitor")
	runCmd.Flags().IntVarP(&connections, "connections", "c", 10, "Number of concurrent connections to use during the load test")
	runCmd.Flags().StringVarP(&jsonFile, "json", "j", "", "Output results to a JSON file")
	runCmd.Flags().BoolVar(&generateReport, "report", false, "Generate an HTML report")
}
