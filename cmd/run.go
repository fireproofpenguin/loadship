package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
	"github.com/fireproofpenguin/loadship/internal/report"
	"github.com/schollz/progressbar/v3"
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

		shouldMonitorDocker := containerName != ""

		if shouldMonitorDocker {
			isRunning, err := docker.CheckContainerRunning(containerName)

			if err != nil {
				log.Fatalf("Error checking container: %v", err)
			}
			if !isRunning {
				log.Fatalf("Container %s is not running", containerName)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		bar := progressbar.NewOptions(int(duration.Seconds()),
			progressbar.OptionSetDescription("Running test..."),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowElapsedTimeOnFinish(),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionClearOnFinish(),
		)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		var (
			elapsed int
			total   int = int(duration.Seconds())
		)

		go func() {
			for {
				select {
				case <-ticker.C:
					bar.Add(1)
					elapsed += 1
					bar.Describe(fmt.Sprintf("Running test (%d/%ds)", elapsed, total))
				case <-ctx.Done():
					bar.Finish()
					return
				}
			}
		}()

		var wg sync.WaitGroup

		var httpResults []load.HTTPStats
		var dockerResults []docker.DockerStats

		wg.Go(func() {
			httpResults = load.RunHTTPTest(ctx, url, connections)
		})

		if shouldMonitorDocker {
			wg.Go(func() {
				var dockerErr error
				dockerResults, dockerErr = docker.RunDockerMonitor(ctx, containerName)
				if dockerErr != nil {
					fmt.Println("Docker monitoring failed:", dockerErr)
				}
			})
		}

		wg.Wait()

		fmt.Printf("\nLoad test complete. Processing results...\n")

		metrics := collector.Calculate(httpResults, dockerResults, duration)
		metrics.PrettyPrint()

		if jsonFile != "" {
			// return the struct, then de-sereialise to avoid double work
			metricsJSON := collector.ToJSONOutput(httpResults, dockerResults, config, *metrics)

			json, err := json.Marshal(metricsJSON)

			if err != nil {
				fmt.Println("Error converting metrics to JSON:", err)
				return
			}

			outputPath, err := filepath.Abs(jsonFile)

			if err != nil {
				fmt.Println("Error determining absolute path for JSON file:", err)
				return
			}

			err = os.WriteFile(outputPath, json, 0644)

			if err != nil {
				fmt.Println("Error writing JSON file:", err)
				return
			}

			fmt.Printf("\n✓ Results saved to %s\n", outputPath)

			if generateReport {
				reportName := strings.TrimSuffix(jsonFile, ".json")

				report.Write(&metricsJSON, reportName)
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
