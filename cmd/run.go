package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	duration      string
	connections   int
	containerName string
	jsonFile      string
)

var runCmd = &cobra.Command{
	Use:   "run <target-url>",
	Short: "Run load tests against a target service",
	Long:  `Run a load test against a service, with or without docker.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Please provide a target URL")
			return
		}

		dur, durErr := time.ParseDuration(duration)
		if durErr != nil {
			fmt.Println("Invalid duration:", durErr)
			return
		}

		if connections <= 0 {
			fmt.Println("Must have at least one connection")
			return
		}

		url := args[0]
		testStart := time.Now()

		config := collector.TestConfig{
			URL:           url,
			Timestamp:     testStart,
			Duration:      dur,
			Connections:   connections,
			ContainerName: containerName,
		}

		// Do a preflight HTTP check against the provided URL. Only care about transport issues - valid HTTP responses are fine
		// This prevents us gunking up the output with a bunch of failed requests that resolve almost instantly
		preflightClient := &http.Client{Timeout: 10 * time.Second}
		resp, err := preflightClient.Get(url)
		if err != nil {
			log.Fatalf("Cannot reach %s: %v", url, err)
		}
		resp.Body.Close()

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

		ctx, cancel := context.WithTimeout(context.Background(), dur)
		defer cancel()

		bar := progressbar.NewOptions(int(dur.Seconds()),
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
			total   int = int(dur.Seconds())
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

		metrics := collector.Calculate(httpResults, dockerResults, dur)
		metrics.PrettyPrint()

		if jsonFile != "" {
			metricsJSON, err := collector.OutputJSON(httpResults, dockerResults, config, *metrics)

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
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&duration, "duration", "d", "30s", "Duration of the load test (e.g., 10s, 1m)")
	runCmd.Flags().StringVar(&containerName, "container", "", "Docker container name or id to monitor")
	runCmd.Flags().IntVarP(&connections, "connections", "c", 10, "Number of concurrent connections to use during the load test")
	runCmd.Flags().StringVarP(&jsonFile, "json", "j", "", "Output results to a JSON file")
}
