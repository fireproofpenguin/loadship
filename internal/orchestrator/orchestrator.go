package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/docker"
	"github.com/fireproofpenguin/loadship/internal/load"
	"github.com/schollz/progressbar/v3"
)

func Orchestrate(config collector.TestConfig) ([]load.HTTPStats, []docker.DockerStats, error) {
	preflightChecks(config)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	bar := progressbar.NewOptions(int(config.Duration.Seconds()),
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
		total   int = int(config.Duration.Seconds())
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
		httpResults = load.RunHTTPTest(ctx, config.URL, config.Connections)
	})

	if config.ContainerName != "" {
		wg.Go(func() {
			var dockerErr error
			dockerResults, dockerErr = docker.RunDockerMonitor(ctx, config.ContainerName)
			if dockerErr != nil {
				fmt.Println("Docker monitoring failed:", dockerErr)
			}
		})
	}

	wg.Wait()

	return httpResults, dockerResults, nil
}

func preflightChecks(config collector.TestConfig) error {
	// Do a preflight HTTP check against the provided URL. Only care about transport issues - valid HTTP responses are fine
	// This prevents us gunking up the output with a bunch of failed requests that resolve almost instantly
	preflightClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := preflightClient.Get(config.URL)
	if err != nil {
		return fmt.Errorf("Preflight HTTP check failed: %v", err)
	}
	resp.Body.Close()

	if config.ContainerName != "" {
		isRunning, err := docker.CheckContainerRunning(config.ContainerName)

		if err != nil {
			return fmt.Errorf("Preflight container check failed: %v", err)
		}
		if !isRunning {
			return fmt.Errorf("Preflight container check failed: Container %s is not running", config.ContainerName)
		}
	}

	return nil
}
