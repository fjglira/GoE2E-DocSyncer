package config

import (
	"fmt"
	"strings"

	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
)

// Validate checks the Config for required fields and valid values.
func Validate(cfg *Config) error {
	var errs []string

	// Input validation
	if len(cfg.Input.Directories) == 0 {
		errs = append(errs, "input.directories must not be empty — add at least one directory (e.g. \"docs\")")
	}
	if len(cfg.Input.Include) == 0 {
		errs = append(errs, "input.include must not be empty — add file patterns (e.g. \"*.md\")")
	}

	// Tags validation
	if len(cfg.Tags.StepTags) == 0 {
		errs = append(errs, "tags.step_tags must not be empty — add at least one tag (e.g. \"go-e2e-step\")")
	}

	// Output validation
	if cfg.Output.Directory == "" {
		errs = append(errs, "output.directory must not be empty — set to e.g. \"tests/e2e/generated\"")
	}
	if cfg.Output.PackageName == "" {
		errs = append(errs, "output.package_name must not be empty — set to a valid Go package name (e.g. \"e2e_generated\")")
	}
	if cfg.Output.FileSuffix == "" {
		errs = append(errs, "output.file_suffix must not be empty — set to e.g. \"_test.go\"")
	}
	if cfg.Output.FileSuffix != "" && !strings.HasSuffix(cfg.Output.FileSuffix, ".go") {
		errs = append(errs, fmt.Sprintf("output.file_suffix must end with .go (got %q) — use e.g. \"_test.go\"", cfg.Output.FileSuffix))
	}

	// Validate logging level
	if cfg.Logging.Level != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[cfg.Logging.Level] {
			errs = append(errs, fmt.Sprintf("logging.level must be one of: debug, info, warn, error (got %q)", cfg.Logging.Level))
		}
	}

	if len(errs) > 0 {
		return domain.NewErrorWithSuggestion("config", "", 0,
			fmt.Sprintf("validation failed:\n  - %s", strings.Join(errs, "\n  - ")),
			"run 'docsyncer init' to generate a valid default configuration",
			nil)
	}

	return nil
}
