package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/fireproofpenguin/loadship/internal/comparison"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare test results",
	Long: `Compare multiple test results and show the differences.

Example usage: loadship compare baseline.json test1.json`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("Please provide at least two test result files to compare")
		}

		filenames := make(map[string]bool)
		for _, arg := range args {
			if filepath.Ext(arg) != ".json" {
				return fmt.Errorf("All files must be JSON files with .json extension: %s", arg)
			}

			if filenames[arg] {
				return fmt.Errorf("Duplicate file provided: %s. Please provide different test result files to compare", arg)
			}

			filenames[arg] = true
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		var outputs []*collector.JSONOutput
		for _, arg := range args {
			b, err := os.ReadFile(arg)
			if err != nil {
				return fmt.Errorf("Error reading file: %v\n", err)
			}
			jsonOutput, err := collector.ReadFromJSON(b)
			if err != nil {
				return fmt.Errorf("Error parsing JSON from file: %v\n", err)
			}
			outputs = append(outputs, jsonOutput)
		}

		comparisons := comparison.Compare(outputs)
		fmt.Println("=== Comparing Test Results ===")
		for i, file := range args {
			if i == 0 {
				fmt.Printf("Baseline: %s (%v)\n", file, outputs[i].Metadata.Timestamp)
			} else {
				fmt.Printf("Test %d: %s (%v)\n", i, file, outputs[i].Metadata.Timestamp)
			}
		}

		var hasPrintedWarning bool
		baseline := outputs[0]
		for i, test := range outputs[1:] {
			if !baseline.Metadata.IsSimilar(test.Metadata) {
				if !hasPrintedWarning {
					fmt.Println("\n=== Warning! ===")
					hasPrintedWarning = true
				}
				fmt.Printf("%s does not have similar config to baseline - comparison results may not be valid.\n", args[i+1])
			}
		}

		comparison.PrintComparisonReports(outputs[0], comparisons)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)
}
