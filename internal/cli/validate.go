package cli

import (
	"fmt"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the docsyncer.yaml configuration file",
	Long:  `Loads the configuration file and checks for errors, missing required fields, and invalid values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		fmt.Printf("Configuration file %q is valid.\n", cfgFile)
		log.Debugf("Loaded config: %+v", cfg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
