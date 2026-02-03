package suite

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/orchestrator"
	"github.com/fireproofpenguin/loadship/internal/report"
	"github.com/schollz/progressbar/v3"
)

type Run struct {
	Duration    time.Duration
	Connections int
}

type Config struct {
	Name      string
	Url       string
	Container string
	Cooldown  time.Duration
	Report    bool
	Runs      []Run
}

// validates the suite config
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("suite name cannot be empty")
	}
	if strings.TrimSpace(c.Url) == "" {
		return fmt.Errorf("suite URL cannot be empty")
	}
	if c.Cooldown < 0 {
		return fmt.Errorf("cooldown duration cannot be negative")
	}
	if len(c.Runs) == 0 {
		return fmt.Errorf("suite must have at least one run defined")
	}
	for i, run := range c.Runs {
		if run.Connections <= 0 {
			return fmt.Errorf("run %d has invalid connections: must be greater than 0", i+1)
		}
		if run.Duration <= 0 {
			return fmt.Errorf("run %d has invalid duration: must be greater than 0", i+1)
		}
	}
	return nil
}

func Start(config Config) error {
	fmt.Println("Running test suite from config", config.Name)

	totalRuns := len(config.Runs)

	directory := fmt.Sprintf("suite_%s_%s", config.Name, time.Now().Format("20060102_150405"))

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("error creating suite directory: %w", err)
	}

	for currentRun, run := range config.Runs {
		fmt.Printf("Run (%d/%d): %d connections for %s\n", currentRun+1, totalRuns, run.Connections, run.Duration.String())

		testConfig := collector.TestConfig{
			URL:           config.Url,
			Timestamp:     time.Now(),
			Duration:      run.Duration,
			Connections:   run.Connections,
			ContainerName: config.Container,
		}

		httpStats, dockerStats, err := orchestrator.Orchestrate(testConfig)

		if err != nil {
			fmt.Printf("Run %d failed: %v\n", currentRun+1, err)
			continue
		}

		metrics := collector.Calculate(httpStats, dockerStats, run.Duration)
		metricsOutput := collector.ToJSONOutput(httpStats, dockerStats, testConfig, *metrics)
		filename := fmt.Sprintf("%s/run_%d_%dc_%.0fs.json", directory, currentRun+1, run.Connections, run.Duration.Seconds())
		err = metricsOutput.SaveToFile(filename)

		if err != nil {
			fmt.Println("Error saving JSON file:", err)
		}

		if config.Report {
			reportName := strings.TrimSuffix(filename, ".json")

			report.Write(&metricsOutput, reportName)
		}

		if currentRun < totalRuns-1 {
			cooldown(config.Cooldown)
		}
	}

	fmt.Printf("Test suite complete. Results saved to %s/\n", directory)

	return nil
}

func cooldown(duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	bar := progressbar.NewOptions(int(duration.Seconds()),
		progressbar.OptionSetDescription("Cooldown..."),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionClearOnFinish(),
	)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bar.Add(1)
		case <-ctx.Done():
			bar.Finish()
			return
		}
	}
}
