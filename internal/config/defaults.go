package config

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	recursive := true
	return &Config{
		Input: InputConfig{
			Directories: []string{"docs"},
			Include:     []string{"*.md"},
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
			Attributes: map[string][]string{
				"step_name":          {"step-name", "name"},
				"timeout":           {"timeout"},
				"expected_exit_code": {"expected", "exit-code"},
				"describe":          {"describe"},
				"context":           {"context"},
				"skip_on_failure":   {"skip-on-failure"},
				"template":         {"template"},
			},
		},
		PlaintextPatterns: PlaintextPatternsConfig{
			BlockStart: `^\s*@begin\((\S+)(?:\s+(.*))?\)\s*$`,
			BlockEnd:   `^\s*@end\s*$`,
		},
		Output: OutputConfig{
			Directory:           "tests/e2e/generated",
			FilePrefix:          "generated_",
			FileSuffix:          "_test.go",
			PackageName:         "e2e_generated",
			CleanBeforeGenerate: true,
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
