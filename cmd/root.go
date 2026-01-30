package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "loadship",
	Short: "Loadship is a performance and load test runner for docker-based http services",
	Long: `Loadship is a CLI library to perform performance and load testing for docker-based http services.

Loadship runs multiple http clients in parallel containers to generate load against a target service and monitors resource usage on the target service's host.

loadship run http://localhost:8080`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
