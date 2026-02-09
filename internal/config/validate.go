package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// Validate checks the Config for required fields and valid values.
func Validate(cfg *Config) error {
	var errs []string

	// Input validation
	if len(cfg.Input.Directories) == 0 {
		errs = append(errs, "input.directories must not be empty")
	}
	if len(cfg.Input.Include) == 0 {
		errs = append(errs, "input.include must not be empty")
	}

	// Tags validation
	if len(cfg.Tags.StepTags) == 0 {
		errs = append(errs, "tags.step_tags must not be empty")
	}

	// Output validation
	if cfg.Output.Directory == "" {
		errs = append(errs, "output.directory must not be empty")
	}
	if cfg.Output.PackageName == "" {
		errs = append(errs, "output.package_name must not be empty")
	}
	if cfg.Output.FileSuffix == "" {
		errs = append(errs, "output.file_suffix must not be empty")
	}
	if !strings.HasSuffix(cfg.Output.FileSuffix, ".go") {
		errs = append(errs, "output.file_suffix must end with .go")
	}

	// Validate plaintext patterns are valid regex (if set)
	if cfg.PlaintextPatterns.BlockStart != "" {
		if _, err := regexp.Compile(cfg.PlaintextPatterns.BlockStart); err != nil {
			errs = append(errs, fmt.Sprintf("plaintext_patterns.block_start is not a valid regex: %v", err))
		}
	}
	if cfg.PlaintextPatterns.BlockEnd != "" {
		if _, err := regexp.Compile(cfg.PlaintextPatterns.BlockEnd); err != nil {
			errs = append(errs, fmt.Sprintf("plaintext_patterns.block_end is not a valid regex: %v", err))
		}
	}

	// Validate logging level
	if cfg.Logging.Level != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[cfg.Logging.Level] {
			errs = append(errs, fmt.Sprintf("logging.level must be one of: debug, info, warn, error (got %q)", cfg.Logging.Level))
		}
	}

	if len(errs) > 0 {
		return domain.NewError("config", "", 0, fmt.Sprintf("validation failed: %s", strings.Join(errs, "; ")), nil)
	}

	return nil
}
