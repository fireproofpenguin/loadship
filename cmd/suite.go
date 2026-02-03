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
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := args[0]

		bytes, err := os.ReadFile(configFile)

		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}

		config := suite.Config{}
		err = yaml.Unmarshal(bytes, &config)

		if err != nil {
			return fmt.Errorf("error parsing config file: %w", err)
		}

		err = config.Validate()

		if err != nil {
			return fmt.Errorf("suite config contains 1 or more errors: %w", err)
		}

		err = suite.Start(config)

		if err != nil {
			return fmt.Errorf("error running test suite: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(suiteCmd)
}
