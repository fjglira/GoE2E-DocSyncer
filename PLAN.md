# GoE2E-DocSyncer v0.1 — Implementation Plan

> **Date**: February 9, 2026
> **Version**: 0.1
> **Status**: Architecture & Implementation Plan
> **Goal**: Universal doc-to-Ginkgo E2E test generator for any Go project

---

## 1. Vision & Scope

**GoE2E-DocSyncer** is a standalone Go CLI tool that reads **any text-based documentation file** (Markdown, AsciiDoc, RTF, reStructuredText, plain text, etc.) and generates executable **Ginkgo/Gomega E2E test files** for any Go project.

Everything is driven by a **YAML configuration file** (`docsyncer.yaml`) placed in the target Go project's root. The config defines:
- Which tags/markers identify test steps inside documents
- Which file types and paths to scan
- How to structure the generated test output
- Template selection, package names, timeout defaults, etc.

### Design Pillars

| Pillar | Description |
|--------|-------------|
| **Any text format** | Supports Markdown, AsciiDoc, and any generic text file via a pluggable parser registry |
| **YAML-driven** | A single `docsyncer.yaml` config file + optional CLI overrides control all behavior |
| **Pluggable parsers** | Interface + Registry pattern — add a new format by implementing one interface |
| **Portable** | Works in any Go project; zero assumptions about project structure |
| **Configurable tags** | Tag names, markers, and patterns are all defined in YAML — nothing is hard-coded |
| **Interface-driven DI** | All core components behind interfaces for testability and extensibility |
| **BDD tested** | Ginkgo/Gomega BDD tests from day one |
| **Structured errors** | Every error carries phase, file, line number, and context chain |

---

## 2. Architecture

### 2.1 High-Level Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│              │     │              │     │              │     │              │
│  YAML Config │────▶│   Scanner    │────▶│   Parser     │────▶│  Converter   │
│  (docsyncer  │     │  (discover   │     │  (extract    │     │  (commands   │
│   .yaml)     │     │   files)     │     │   blocks)    │     │   → Go code) │
│              │     │              │     │              │     │              │
└──────────────┘     └──────────────┘     └──────────────┘     └──────┬───────┘
                                                                      │
                                                                      ▼
                     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
                     │              │     │              │     │              │
                     │  Formatter   │◀────│  Template    │◀────│  TestSpec    │
                     │  (go/format) │     │  Engine      │     │  (domain     │
                     │              │     │  (render)    │     │   model)     │
                     │              │     │              │     │              │
                     └──────┬───────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  Output      │
                     │  *_test.go   │
                     │  files       │
                     └──────────────┘
```

### 2.2 Package Layout

```
GoE2E-DocSyncer/
├── cmd/
│   └── docsyncer/
│       └── main.go                    # CLI entry point
│
├── internal/
│   ├── domain/                        # Core domain types (no external deps)
│   │   ├── types.go                   # TestSpec, TestStep, CodeBlock, ParsedDocument
│   │   └── errors.go                  # Structured domain errors
│   │
│   ├── config/                        # YAML configuration loading
│   │   ├── config.go                  # Config struct + loader
│   │   ├── defaults.go               # Default values
│   │   ├── validate.go               # Config validation logic
│   │   └── config_test.go
│   │
│   ├── scanner/                       # File discovery
│   │   ├── scanner.go                 # File scanner interface + implementation
│   │   └── scanner_test.go
│   │
│   ├── parser/                        # Document parsing (pluggable)
│   │   ├── parser.go                  # Parser interface definition
│   │   ├── registry.go               # Parser registry (maps extensions → parsers)
│   │   ├── markdown.go               # Goldmark-based Markdown parser
│   │   ├── asciidoc.go               # AsciiDoc parser (regex-based)
│   │   ├── plaintext.go              # Generic regex-based parser (fallback)
│   │   ├── markdown_test.go
│   │   ├── asciidoc_test.go
│   │   └── plaintext_test.go
│   │
│   ├── converter/                     # Shell command → Go code conversion
│   │   ├── converter.go               # Converter interface + implementation
│   │   ├── command.go                 # Command parsing and Go code generation
│   │   ├── security.go               # Command security validation
│   │   └── converter_test.go
│   │
│   ├── template/                      # Template rendering
│   │   ├── engine.go                  # Template engine interface + implementation
│   │   ├── functions.go              # Custom template functions
│   │   └── engine_test.go
│   │
│   ├── generator/                     # Orchestration layer
│   │   ├── generator.go               # Main orchestrator (wires all components)
│   │   └── generator_test.go
│   │
│   └── cli/                           # CLI interface
│       ├── root.go                    # Root command
│       ├── generate.go               # `generate` subcommand
│       ├── init.go                   # `init` subcommand (scaffold config)
│       └── validate.go              # `validate` subcommand (check config)
│
├── templates/                         # Shipped default templates
│   └── ginkgo_default.tmpl
│
├── testdata/                          # Test fixtures
│   ├── markdown/
│   │   ├── simple.md
│   │   └── multi-step.md
│   ├── asciidoc/
│   │   └── sample.adoc
│   ├── plaintext/
│   │   └── generic.txt
│   └── configs/
│       ├── minimal.yaml
│       └── full.yaml
│
├── docsyncer.yaml                     # Example / reference configuration
├── go.mod
├── go.sum
├── .cursorrules                       # Cursor IDE rules for AI-assisted dev
└── PLAN.md                            # This document
```

### 2.3 Core Interfaces

These are the primary contracts that all implementations must satisfy:

```go
// Parser extracts code blocks from a document.
type Parser interface {
    // Parse reads raw document content and returns extracted blocks.
    Parse(filePath string, content []byte, tags []string) (*domain.ParsedDocument, error)
    // SupportedExtensions returns the file extensions this parser handles.
    SupportedExtensions() []string
}

// ParserRegistry maps file extensions to parsers.
type ParserRegistry interface {
    Register(parser Parser)
    ParserFor(extension string) (Parser, error)
}

// Converter transforms parsed documents into TestSpec domain models.
type Converter interface {
    Convert(doc *domain.ParsedDocument, cfg *config.TagConfig) ([]domain.TestSpec, error)
}

// TemplateEngine renders TestSpec into Go source code strings.
type TemplateEngine interface {
    Render(spec domain.TestSpec, packageName string) (string, error)
    ListTemplates() []string
}

// Scanner discovers documentation files in the project tree.
type Scanner interface {
    Scan(rootDir string, patterns []string, excludes []string) ([]string, error)
}

// Generator is the top-level orchestrator.
type Generator interface {
    Generate(cfg *config.Config) error
}
```

---

## 3. YAML Configuration Schema

The configuration file `docsyncer.yaml` is the heart of the tool. Here is the full schema:

```yaml
# docsyncer.yaml — GoE2E-DocSyncer configuration
# Place this file in the root of your Go project.

# =============================================================================
# Input Configuration
# =============================================================================
input:
  # Directories to scan for documentation files (relative to project root)
  directories:
    - "docs"
    - "documentation"

  # File patterns to include (glob syntax)
  include:
    - "*.md"
    - "*.adoc"
    - "*.asciidoc"
    - "*.txt"
    - "*.rst"

  # Patterns/directories to exclude
  exclude:
    - "vendor/**"
    - "node_modules/**"
    - "**/CHANGELOG.md"

  # Recursive search (default: true)
  recursive: true

# =============================================================================
# Tag Configuration — What markers identify test content
# =============================================================================
tags:
  # Code fence language tags that identify test steps.
  # For Markdown: ```<tag>   For AsciiDoc: [source,<tag>]
  # For plaintext/generic: Regex-based detection (see patterns below)
  step_tags:
    - "go-e2e-step"
    - "e2e-test"

  # Markers for test boundaries (test start/end)
  test_start:
    # Comment-based markers (works in any text format)
    comment_markers:
      - "<!-- test-start:"      # HTML comment style
      - "// test-start:"        # Go/C comment style
      - "# test-start:"        # Shell/Python comment style
    # Attribute-based (inside code fence attributes)
    attribute_key: "init"

  test_end:
    comment_markers:
      - "<!-- test-end -->"
      - "// test-end"
      - "# test-end"
    attribute_key: "end"

  # Attribute names recognized inside code fence metadata
  attributes:
    step_name: ["step-name", "name"]
    timeout: ["timeout"]
    expected_exit_code: ["expected", "exit-code"]
    describe: ["describe"]
    context: ["context"]
    skip_on_failure: ["skip-on-failure"]
    template: ["template"]

# =============================================================================
# Plaintext / Generic Parser Patterns
# =============================================================================
# For file formats without native code fence syntax, these regex patterns
# are used to identify tagged blocks.
plaintext_patterns:
  # Block start pattern — capture group 1 = tag, group 2 = attributes
  block_start: '^\s*@begin\((\S+)(?:\s+(.*))?\)\s*$'
  # Block end pattern
  block_end: '^\s*@end\s*$'
  # Alternative: marker-based (lines starting with a prefix)
  line_prefix: ">>> "

# =============================================================================
# Output Configuration
# =============================================================================
output:
  # Output directory for generated test files (relative to project root)
  directory: "tests/e2e/generated"

  # Generated file naming: prefix + <source-filename> + suffix
  file_prefix: "generated_"
  file_suffix: "_test.go"

  # Go package name for generated test files
  package_name: "e2e_generated"

  # Clean output directory before generating (default: true)
  clean_before_generate: true

# =============================================================================
# Template Configuration
# =============================================================================
templates:
  # Directory containing custom templates (relative to project root)
  directory: "templates"

  # Default template to use (filename without .tmpl extension)
  default: "ginkgo_default"

  # Allow per-test template override via tag attributes (default: true)
  allow_override: true

# =============================================================================
# Command Conversion Settings
# =============================================================================
commands:
  # Default timeout for commands (Go duration format)
  default_timeout: "30s"

  # Default expected exit code
  default_expected_exit_code: 0

  # Security: block dangerous command patterns
  blocked_patterns:
    - "rm -rf /"
    - "mkfs"
    - "dd if="
    - "format c:"
    - "> /dev/sd"

  # Shell to use for complex commands (pipes, redirects)
  shell: "/bin/sh"
  shell_flag: "-c"

# =============================================================================
# Logging & Behavior
# =============================================================================
logging:
  # Log level: debug, info, warn, error
  level: "info"
  # Optional log file (in addition to stdout)
  file: ""

# Dry-run mode: parse and convert but don't write files
dry_run: false
```

---

## 4. Domain Model

### 4.1 Core Types

```go
// ParsedDocument holds the result of parsing a single document file.
type ParsedDocument struct {
    FilePath   string
    FileType   string           // "markdown", "asciidoc", "plaintext", etc.
    Blocks     []CodeBlock      // All extracted code blocks (tagged ones)
    Headings   []Heading        // Document structure (for context inference)
    Metadata   map[string]string // Any document-level metadata found
}

// CodeBlock represents a single tagged code block extracted from a document.
type CodeBlock struct {
    Tag        string            // The matched tag (e.g. "go-e2e-step")
    Content    string            // Raw content of the block
    LineNumber int               // 1-based line number in source
    Attributes map[string]string // Key-value attributes from the fence info
    Context    string            // Nearest heading / section title
}

// Heading represents a document heading for context inference.
type Heading struct {
    Level int
    Text  string
    Line  int
}

// TestSpec is the fully converted test specification ready for template rendering.
type TestSpec struct {
    SourceFile    string
    SourceType    string
    TestName      string
    DescribeBlock string
    ContextBlock  string
    Steps         []TestStep
    TemplateName  string
}

// TestStep is a single executable step within a test.
type TestStep struct {
    Name           string
    Command        string
    GoCode         string   // Generated Go code for this step
    ExpectedExit   int
    Timeout        string
    LineNumber     int
    SkipOnFailure  bool
}
```

### 4.2 Error Types

```go
// DocSyncerError is the base error type with context.
type DocSyncerError struct {
    Phase      string // "config", "scan", "parse", "convert", "template", "write"
    File       string
    LineNumber int
    Message    string
    Cause      error
}

func (e *DocSyncerError) Error() string { ... }
func (e *DocSyncerError) Unwrap() error { return e.Cause }
```

---

## 5. Parser Strategy (Multi-Format Support)

### 5.1 Tier 1: Format-Aware Parsers

These use the format's native constructs:

| Format | Library | Code Block Detection |
|--------|---------|---------------------|
| **Markdown** (.md, .markdown) | `github.com/yuin/goldmark` | Fenced code blocks with language tag matching `step_tags` |
| **AsciiDoc** (.adoc, .asciidoc) | Regex-based | `[source,<tag>]` blocks between `----` delimiters |

### 5.2 Tier 2: Generic/Plaintext Parser

For any other text format (`.txt`, `.rst`, `.rtf`, etc.), a **regex-based generic parser** uses patterns from the YAML config to detect blocks:

**Strategy**: The plaintext parser reads the file line-by-line and applies configurable regex patterns:
1. Detect block start using `plaintext_patterns.block_start` regex
2. Capture content until `plaintext_patterns.block_end` regex matches
3. Extract tag and attributes from regex capture groups

**RTF Handling**: RTF files are first stripped to plaintext (removing `\rtf1` control words) before the generic parser processes them. We use a simple built-in RTF stripper (no external dependency needed — RTF control words follow a predictable regex pattern).

### 5.3 Parser Registry

```go
// On startup, parsers register themselves:
registry.Register(markdown.NewParser())   // handles .md, .markdown
registry.Register(asciidoc.NewParser())   // handles .adoc, .asciidoc
registry.Register(plaintext.NewParser())  // handles everything else (fallback)
```

The registry tries a format-aware parser first; if no parser is registered for the extension, it falls back to the plaintext parser.

---

## 6. Implementation Phases

### Phase 1: Foundation (Core Infrastructure)
**Goal**: Project skeleton, domain types, configuration, and basic CLI.

| Task | Description | Package |
|------|-------------|---------|
| 1.1 | Initialize Go module, set up `go.mod` with deps | root |
| 1.2 | Define domain types (`ParsedDocument`, `TestSpec`, `TestStep`, etc.) | `internal/domain` |
| 1.3 | Define domain errors | `internal/domain` |
| 1.4 | Create YAML config struct, loader, validator | `internal/config` |
| 1.5 | Create config defaults | `internal/config` |
| 1.6 | Set up CLI skeleton with cobra (root + subcommands) | `internal/cli` |
| 1.7 | Write Ginkgo tests for config loading & validation | `internal/config` |

**Deliverable**: `docsyncer init` creates a scaffold `docsyncer.yaml`; `docsyncer validate` checks it.

### Phase 2: Parsing Layer
**Goal**: Parse any document format and extract tagged code blocks.

| Task | Description | Package |
|------|-------------|---------|
| 2.1 | Define `Parser` interface and `ParserRegistry` | `internal/parser` |
| 2.2 | Implement `ParserRegistry` | `internal/parser` |
| 2.3 | Implement Markdown parser with goldmark | `internal/parser` |
| 2.4 | Implement AsciiDoc parser (regex-based) | `internal/parser` |
| 2.5 | Implement Plaintext/Generic parser | `internal/parser` |
| 2.6 | Implement RTF text stripper utility | `internal/parser` |
| 2.7 | Implement file scanner | `internal/scanner` |
| 2.8 | Write Ginkgo tests for each parser | `internal/parser` |
| 2.9 | Write Ginkgo tests for scanner | `internal/scanner` |

**Deliverable**: Can parse MD, AsciiDoc, and generic text files; extract tagged blocks.

### Phase 3: Conversion & Command Handling
**Goal**: Convert parsed blocks into TestSpec domain models with generated Go code.

| Task | Description | Package |
|------|-------------|---------|
| 3.1 | Define `Converter` interface | `internal/converter` |
| 3.2 | Implement block → TestStep conversion | `internal/converter` |
| 3.3 | Implement shell command → Go code generator | `internal/converter` |
| 3.4 | Implement command security validation | `internal/converter` |
| 3.5 | Implement smart step naming (kubectl, docker, curl, etc.) | `internal/converter` |
| 3.6 | Implement test grouping (init/end markers) | `internal/converter` |
| 3.7 | Write Ginkgo tests for converter | `internal/converter` |

**Deliverable**: Full pipeline from `ParsedDocument` → `[]TestSpec` with Go code.

### Phase 4: Template Engine & Code Generation
**Goal**: Render TestSpecs into formatted `_test.go` files.

| Task | Description | Package |
|------|-------------|---------|
| 4.1 | Define `TemplateEngine` interface | `internal/template` |
| 4.2 | Create default Ginkgo template | `templates/` |
| 4.3 | Implement template loader + custom function registration | `internal/template` |
| 4.4 | Implement template rendering with `go/format` integration | `internal/template` |
| 4.5 | Write Ginkgo tests for template engine | `internal/template` |

**Deliverable**: Rendered, `go/format`-compliant `_test.go` output.

### Phase 5: Orchestration & CLI Completion
**Goal**: Wire everything together; complete CLI commands.

| Task | Description | Package |
|------|-------------|---------|
| 5.1 | Implement `Generator` orchestrator | `internal/generator` |
| 5.2 | Implement `generate` CLI command (main flow) | `internal/cli` |
| 5.3 | Implement `init` CLI command (scaffold config) | `internal/cli` |
| 5.4 | Implement `validate` CLI command | `internal/cli` |
| 5.5 | Implement cleanup/clean-before-generate | `internal/generator` |
| 5.6 | Implement dry-run mode | `internal/generator` |
| 5.7 | Write integration tests (end-to-end) | `internal/generator` |

**Deliverable**: Fully working `docsyncer generate` command.

### Phase 6: Polish & Documentation
**Goal**: Production-quality output, documentation, examples.

| Task | Description | Package |
|------|-------------|---------|
| 6.1 | Create example documentation files (testdata) | `testdata/` |
| 6.2 | Add comprehensive error messages + suggestions | all |
| 6.3 | Add verbose/debug logging throughout | all |
| 6.4 | Create README.md | root |
| 6.5 | golangci-lint configuration + fixes | root |
| 6.6 | CI pipeline (Makefile, GitHub Actions) | root |

---

## 7. Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/yuin/goldmark` | latest | CommonMark Markdown AST parsing |
| `github.com/spf13/cobra` | v1.8+ | CLI framework |
| `github.com/sirupsen/logrus` | v1.9+ | Structured logging |
| `gopkg.in/yaml.v3` | v3 | YAML configuration parsing |
| `github.com/onsi/ginkgo/v2` | v2.x | BDD testing framework |
| `github.com/onsi/gomega` | v1.x | Assertion library for Ginkgo |

**Principles**:
- Prefer Go standard library (`text/template`, `go/format`, `os/exec`, `regexp`, `path/filepath`)
- No dependency for AsciiDoc (regex-based)
- No dependency for RTF (simple control-word stripper)
- No dependency for generic text (regex from config)

---

## 8. How It Works in Any Go Project

### Installation & Setup

```bash
# Install the tool
go install github.com/frherrer/GoE2E-DocSyncer/cmd/docsyncer@latest

# In your Go project, initialize config:
cd /path/to/your-go-project
docsyncer init          # Creates docsyncer.yaml with sensible defaults

# Edit docsyncer.yaml to match your project's docs structure

# Generate tests:
docsyncer generate      # Reads docsyncer.yaml, scans docs, generates tests

# Validate configuration:
docsyncer validate      # Checks docsyncer.yaml for errors
```

### Project-Agnostic Design

The tool makes **zero assumptions** about the target Go project's structure. Everything is driven by `docsyncer.yaml`:
- Where are the docs? → `input.directories`
- What file types? → `input.include`
- What tags? → `tags.step_tags`
- Where to put tests? → `output.directory`
- What package name? → `output.package_name`

---

## 9. Example Workflow

### Step 1: Documentation Author writes docs

```markdown
# API Gateway Setup

## Deploy Redis Cache

<!-- test-start: Redis deployment E2E -->

### Install Redis

```go-e2e-step step-name="Deploy Redis via Helm"
helm install redis bitnami/redis --set auth.enabled=false
```

### Verify Redis is running

```go-e2e-step timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=120s
```

<!-- test-end -->
```

### Step 2: Run the generator

```bash
docsyncer generate --verbose
```

### Step 3: Generated test file

```go
package e2e_generated

import (
    "os/exec"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("API Gateway Setup", func() {
    It("Redis deployment E2E", func() {
        By("Deploy Redis via Helm")
        cmd := exec.Command("helm", "install", "redis", "bitnami/redis", "--set", "auth.enabled=false")
        output, err := cmd.CombinedOutput()
        Expect(err).ToNot(HaveOccurred(), string(output))

        By("Verify Redis is running")
        cmd = exec.Command("kubectl", "wait", "--for=condition=ready", "pod",
            "-l", "app.kubernetes.io/name=redis", "--timeout=120s")
        output, err = cmd.CombinedOutput()
        Expect(err).ToNot(HaveOccurred(), string(output))
    })
})
```

---

## 10. Decision Log

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | **Goldmark** for Markdown | CommonMark compliant, clean AST, actively maintained, no CGo |
| 2 | **Regex-based** AsciiDoc | Avoids heavy `libasciidoc` dependency; covers 95% of use cases |
| 3 | **Generic plaintext parser** | Enables support for any text format via configurable regex |
| 4 | **Built-in RTF stripper** | RTF control words are simple regex; no external dep needed |
| 5 | **YAML over TOML/JSON** | Most human-readable for config; `yaml.v3` is mature and trusted |
| 6 | **Interface-driven DI** | Testability, extensibility, clean architecture |
| 7 | **Parser registry** | New format support = implement interface + register; zero changes to core |
| 8 | **`go/format`** for output | Guarantees generated code passes `gofmt` / `goimports` |
| 9 | **Ginkgo BDD tests** | Aligns with user's testing framework preference; consistent style |
| 10 | **Cobra CLI** | Industry standard Go CLI framework; subcommand support |
| 11 | **No global state** | All state via constructor injection; aligns with Go best practices |
| 12 | **Binary name `docsyncer`** | Short, descriptive, easy to type |

---

## 11. Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| RTF files with complex formatting | Medium | Strip to plaintext first; document limitation |
| Regex patterns don't cover edge cases | Medium | Allow multiple patterns per format in YAML |
| Generated code doesn't compile | High | Validate with `go/format`; integration tests |
| Large documents slow parsing | Low | Single-pass streaming; benchmark early |
| Goldmark API changes | Low | Pin version in `go.mod` |

---

## 12. Success Criteria

- [ ] `docsyncer init` scaffolds a valid `docsyncer.yaml`
- [ ] `docsyncer validate` catches config errors with clear messages
- [ ] `docsyncer generate` produces valid `_test.go` from `.md` files
- [ ] `docsyncer generate` produces valid `_test.go` from `.adoc` files
- [ ] `docsyncer generate` produces valid `_test.go` from `.txt` files (generic)
- [ ] Generated tests compile with `go build`
- [ ] All components have Ginkgo BDD test coverage
- [ ] Tool works when installed in any Go project (not just this repo)
- [ ] YAML config is the single source of truth for all behavior
- [ ] `--dry-run` mode shows what would be generated without writing files

---

*This plan will be updated as implementation progresses.*
