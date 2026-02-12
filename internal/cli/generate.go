package cli

import (
	"fmt"

	"github.com/fjglira/GoE2E-DocSyncer/internal/config"
	"github.com/fjglira/GoE2E-DocSyncer/internal/converter"
	"github.com/fjglira/GoE2E-DocSyncer/internal/generator"
	"github.com/fjglira/GoE2E-DocSyncer/internal/parser"
	"github.com/fjglira/GoE2E-DocSyncer/internal/scanner"
	tmpl "github.com/fjglira/GoE2E-DocSyncer/internal/template"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate E2E test files from documentation",
	Long:  `Scans documentation files, extracts tagged code blocks, and generates Ginkgo test files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		if dryRun {
			cfg.DryRun = true
		}

		log.Info("Configuration loaded successfully")
		log.Info("Scanning directories", "directories", cfg.Input.Directories)
		log.Info("Output directory", "path", cfg.Output.Directory)

		return runGenerate(cfg)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

// runGenerate wires all components and runs the generator.
func runGenerate(cfg *config.Config) error {
	// Create scanner
	recursive := true
	if cfg.Input.Recursive != nil {
		recursive = *cfg.Input.Recursive
	}
	s := scanner.NewScanner(recursive)

	// Create parser registry
	registry := parser.NewRegistry()
	registry.Register(parser.NewMarkdownParser())
	registry.Register(parser.NewAsciiDocParser())

	// Create converter
	conv := converter.NewConverter(&cfg.Commands)

	// Create template engine
	engine, err := tmpl.NewEngine(cfg.Templates.Directory, cfg.Templates.Default, cfg.Output.BuildTag)
	if err != nil {
		return fmt.Errorf("failed to create template engine: %w", err)
	}

	// Create and run generator
	gen := generator.NewGenerator(s, registry, conv, engine, log)
	return gen.Generate(cfg)
}
