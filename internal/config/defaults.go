package config

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	recursive := true
	return &Config{
		Input: InputConfig{
			Directories: []string{"docs"},
			Include:     []string{"*.md", "*.adoc"},
			Exclude:     []string{"vendor/**", "node_modules/**"},
			Recursive:   &recursive,
		},
		Tags: TagConfig{
			StepTags: []string{"go-e2e-step"},
			TestStart: TestMarkerConfig{
				CommentMarkers: []string{
					"<!-- test-start:",
					"// test-start:",
					"# test-start:",
				},
				AttributeKey: "init",
			},
			TestEnd: TestMarkerConfig{
				CommentMarkers: []string{
					"<!-- test-end",
					"// test-end",
					"# test-end",
				},
				AttributeKey: "end",
			},
			StepStart: TestMarkerConfig{
				CommentMarkers: []string{
					"<!-- test-step-start:",
					"// test-step-start:",
					"# test-step-start:",
				},
			},
			StepEnd: TestMarkerConfig{
				CommentMarkers: []string{
					"<!-- test-step-end",
					"// test-step-end",
					"# test-step-end",
				},
			},
			Attributes: map[string][]string{
				"step_name":          {"step-name", "name"},
				"timeout":           {"timeout"},
				"expected_exit_code": {"expected", "exit-code"},
				"describe":          {"describe"},
				"context":           {"context"},
				"skip_on_failure":   {"skip-on-failure"},
				"template":         {"template"},
				"retry":            {"retry", "retries", "retry-count"},
				"retry_interval":   {"retry-interval", "retry-delay"},
			},
		},
		Output: OutputConfig{
			Directory:           "tests/e2e/generated",
			FilePrefix:          "generated_",
			FileSuffix:          "_test.go",
			PackageName:         "e2e_generated",
			CleanBeforeGenerate: true,
			DefaultLabels:       []string{"documentation"},
		},
		Templates: TemplateConfig{
			Directory:     "templates",
			Default:       "ginkgo_default",
			AllowOverride: true,
		},
		Commands: CommandConfig{
			DefaultTimeout:          "30s",
			DefaultExpectedExitCode: 0,
			BlockedPatterns: []string{
				"rm -rf /",
				"mkfs",
				"dd if=",
				"format c:",
				"> /dev/sd",
			},
			Shell:     "/bin/sh",
			ShellFlag: "-c",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		DryRun: false,
	}
}
