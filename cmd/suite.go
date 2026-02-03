package cmd

import (
	"fmt"
	"os"

	"github.com/fireproofpenguin/loadship/internal/suite"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var suiteCmd = &cobra.Command{
	Use:   "suite",
	Short: "Run a suite of load tests",
	Long: `Run a suite of load tests based on a configuration file. The test suite will run one after another, collecting results for each.
	loadship suite <config-file>`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("must provide config file")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		configFile := args[0]

		bytes, err := os.ReadFile(configFile)

		if err != nil {
			fmt.Printf("Error reading config file: %v\n", err)
			return
		}

		config := suite.Config{}
		err = yaml.Unmarshal(bytes, &config)

		if err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}

		err = config.Validate()

		if err != nil {
			fmt.Printf("Suite config contains 1 or more errors: %v\n", err)
			return
		}

		suite.Start(config)
	},
}

func init() {
	rootCmd.AddCommand(suiteCmd)
}
