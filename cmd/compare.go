package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fireproofpenguin/loadship/comparison"
	"github.com/fireproofpenguin/loadship/internal/collector"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare test results",
	Long:  `Compare 2 test results and show the differences.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("Please provide two test result files to compare")
			return
		}

		file1 := args[0]

		file2 := args[1]

		if file1 == file2 {
			fmt.Println("Please provide two different test result files to compare")
			return
		}

		file1Path, err := filepath.Abs(file1)

		if err != nil {
			fmt.Println("Error getting absolute path for file 1:", err)
			return
		}

		file2Path, err := filepath.Abs(file2)
		if err != nil {
			fmt.Println("Error getting absolute path for file 2:", err)
			return
		}

		if filepath.Ext(file2Path) != ".json" || filepath.Ext(file1Path) != ".json" {
			fmt.Println("Both files must be JSON files with .json extension")
			return
		}

		b, err := os.ReadFile(file1Path)

		if err != nil {
			log.Fatalf("Error reading file: %v\n", err)
		}

		baseline, err := collector.ReadFromJSON(b)

		if err != nil {
			log.Fatalf("Error parsing JSON from file: %v\n", err)
		}

		b, err = os.ReadFile(file2Path)

		test, err := collector.ReadFromJSON(b)

		if err != nil {
			log.Fatalf("Error parsing JSON from file: %v\n", err)
		}

		comparison := comparison.Compare(baseline, test)

		fmt.Println("=== Comparing Test Results ===")
		fmt.Printf("Baseline: %s (%v)\n", file1, baseline.Metadata.Timestamp)
		fmt.Printf("Test: %s (%v)\n", file2, test.Metadata.Timestamp)

		if !baseline.Metadata.IsSimilar(test.Metadata) {
			fmt.Println("\n=== Warning ===")
			fmt.Println("The two test configurations are not similar. Comparison results may not be valid.")
		}

		comparison.Print()
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)
}
