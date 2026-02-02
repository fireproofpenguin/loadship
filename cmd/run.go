package cmd

import (
	"context"
	"fmt"
	"log"
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

		url := args[0]

		dur, durErr := time.ParseDuration(duration)
		if durErr != nil {
			fmt.Println("Invalid duration:", durErr)
			return
		}

		if connections <= 0 {
			fmt.Println("Must have at least one connection")
			return
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
				dockerResults = docker.RunDockerMonitor(ctx, containerName)
			})
		}

		wg.Wait()

		fmt.Printf("\nLoad test complete. Processing results...\n")

		metrics := collector.Calculate(httpResults, dockerResults, dur)
		metrics.PrettyPrint()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&duration, "duration", "d", "30s", "Duration of the load test (e.g., 10s, 1m)")
	runCmd.Flags().StringVar(&containerName, "container", "", "Docker container name or id to monitor")
	runCmd.Flags().IntVarP(&connections, "connections", "c", 10, "Number of concurrent connections to use during the load test")
}
