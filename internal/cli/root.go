package cli

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool
	log     = logrus.New()
)

// rootCmd is the base command for docsyncer.
var rootCmd = &cobra.Command{
	Use:   "docsyncer",
	Short: "Generate Ginkgo E2E tests from documentation files",
	Long: `GoE2E-DocSyncer reads documentation files (Markdown, AsciiDoc, plain text)
and generates executable Ginkgo/Gomega E2E test files.

Everything is driven by a YAML configuration file (docsyncer.yaml).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(logrus.DebugLevel)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "docsyncer.yaml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "parse and convert but don't write files")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
