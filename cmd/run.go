package cmd

import (
	"fmt"
	"time"

	"github.com/fireproofpenguin/loadship/internal/load"
	"github.com/spf13/cobra"
)

var duration string

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

		err := load.Run(args[0], dur)

		if err != nil {
			fmt.Println("Error running load test:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&duration, "duration", "d", "30s", "Duration of the load test (e.g., 10s, 1m)")
}
