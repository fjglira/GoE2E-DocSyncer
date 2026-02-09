package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// Config is the top-level configuration struct.
type Config struct {
	Input             InputConfig            `yaml:"input"`
	Tags              TagConfig              `yaml:"tags"`
	PlaintextPatterns PlaintextPatternsConfig `yaml:"plaintext_patterns"`
	Output            OutputConfig           `yaml:"output"`
	Templates         TemplateConfig         `yaml:"templates"`
	Commands          CommandConfig          `yaml:"commands"`
	Logging           LoggingConfig          `yaml:"logging"`
	DryRun            bool                   `yaml:"dry_run"`
}

type InputConfig struct {
	Directories []string `yaml:"directories"`
	Include     []string `yaml:"include"`
	Exclude     []string `yaml:"exclude"`
	Recursive   *bool    `yaml:"recursive"` // pointer to distinguish unset from false
}

type TagConfig struct {
	StepTags   []string            `yaml:"step_tags"`
	TestStart  TestMarkerConfig    `yaml:"test_start"`
	TestEnd    TestMarkerConfig    `yaml:"test_end"`
	Attributes map[string][]string `yaml:"attributes"`
}

type TestMarkerConfig struct {
	CommentMarkers []string `yaml:"comment_markers"`
	AttributeKey   string   `yaml:"attribute_key"`
}

type PlaintextPatternsConfig struct {
	BlockStart string `yaml:"block_start"`
	BlockEnd   string `yaml:"block_end"`
	LinePrefix string `yaml:"line_prefix"`
}

type OutputConfig struct {
	Directory           string `yaml:"directory"`
	FilePrefix          string `yaml:"file_prefix"`
	FileSuffix          string `yaml:"file_suffix"`
	PackageName         string `yaml:"package_name"`
	CleanBeforeGenerate bool   `yaml:"clean_before_generate"`
}

type TemplateConfig struct {
	Directory     string `yaml:"directory"`
	Default       string `yaml:"default"`
	AllowOverride bool   `yaml:"allow_override"`
}

type CommandConfig struct {
	DefaultTimeout          string   `yaml:"default_timeout"`
	DefaultExpectedExitCode int      `yaml:"default_expected_exit_code"`
	BlockedPatterns         []string `yaml:"blocked_patterns"`
	Shell                   string   `yaml:"shell"`
	ShellFlag               string   `yaml:"shell_flag"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Load reads a YAML configuration file and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, domain.NewError("config", path, 0, "failed to read config file", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, domain.NewError("config", path, 0, "failed to parse config file", err)
	}

	return cfg, nil
}
