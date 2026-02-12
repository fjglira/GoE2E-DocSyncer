# GoE2E-DocSyncer

A Go CLI tool that reads documentation files (Markdown, AsciiDoc) and generates executable [Ginkgo](https://onsi.github.io/ginkgo/)/[Gomega](https://onsi.github.io/gomega/) E2E test files.

Everything is driven by a single YAML configuration file (`docsyncer.yaml`). The tool makes no assumptions about your project structure.

## Features

- **Multi-format support** — Markdown (via goldmark AST) and AsciiDoc
- **YAML-driven** — All behavior controlled by `docsyncer.yaml`; no hard-coded tag names or paths
- **Pluggable parsers** — Add new formats by implementing the `Parser` interface and registering it
- **Test file boundaries** — `<!-- test-start: NAME -->` / `<!-- test-end -->` markers produce **separate output files** (one per pair)
- **Step grouping** — `<!-- test-step-start: NAME -->` / `<!-- test-step-end -->` markers group steps into separate `It()` blocks within a test file
- **Smart code generation** — Shell commands are converted to `exec.Command` / `exec.CommandContext` with timeout and exit code handling
- **Security validation** — Configurable blocked-command patterns prevent dangerous commands in generated tests
- **Embedded default template** — Works with `go run` out of the box; no local `templates/` directory needed
- **Configurable build tags** — Add `//go:build` constraints to generated files via `output.build_tag`
- **go/format compliant** — All generated code passes `gofmt`
- **Dry-run mode** — Preview generated output without writing files

## Installation

```bash
go install github.com/frherrer/GoE2E-DocSyncer/cmd/docsyncer@latest
```

Or build from source:

```bash
git clone https://github.com/frherrer/GoE2E-DocSyncer.git
cd GoE2E-DocSyncer
make build
# Binary: bin/docsyncer
```

## Quick Start

```bash
# 1. Initialize a config file in your project
docsyncer init

# 2. Edit docsyncer.yaml to match your docs structure
#    - Set input.directories to where your docs live
#    - Set tags.step_tags to match your code fence tags
#    - Set output.directory to where tests should go

# 3. Generate tests
docsyncer generate

# 4. Preview without writing files
docsyncer generate --dry-run --verbose
```

## How It Works

1. **Scan** — Discovers documentation files matching your configured patterns
2. **Parse** — Extracts tagged code blocks using format-aware parsers
3. **Convert** — Transforms blocks into `TestSpec` domain models with generated Go code
4. **Render** — Applies Go templates to produce `_test.go` source
5. **Write** — Outputs formatted test files to your configured directory

### Supported Documentation Formats

| Format | Extensions | Parser | Code Block Detection |
|--------|-----------|--------|---------------------|
| Markdown | `.md`, `.markdown` | goldmark AST | Fenced code blocks with tag in info string |
| AsciiDoc | `.adoc`, `.asciidoc` | Regex-based | `[source,<tag>]` blocks between `----` |

### Tagging Code Blocks

In **Markdown**, use a fenced code block with your configured tag as the language:

````markdown
```go-e2e-step step-name="Deploy app" timeout=60s
kubectl apply -f deploy.yaml
```
````

In **AsciiDoc**:

```asciidoc
[source,go-e2e-step,step-name="Deploy app"]
----
kubectl apply -f deploy.yaml
----
```

### Test File Boundaries

Use `test-start` / `test-end` markers to define separate output files. Each pair produces its own `_test.go` file, named after the marker:

```markdown
<!-- test-start: Infrastructure setup -->

```go-e2e-step
helm install postgres bitnami/postgresql
```

```go-e2e-step
kubectl wait --for=condition=ready pod -l app=postgresql
```

<!-- test-end -->

<!-- test-start: Application deployment -->

```go-e2e-step
kubectl apply -f ./k8s/
```

<!-- test-end -->
```

This generates **two** output files: `generated_infrastructure_setup_test.go` and `generated_application_deployment_test.go`.

### Step Grouping

Use `test-step-start` / `test-step-end` markers inside a `test-start` / `test-end` block to split steps into separate `It()` blocks within that test file:

```markdown
<!-- test-start: Database tests -->

<!-- test-step-start: Setup -->

```go-e2e-step
helm install postgres bitnami/postgresql
```

<!-- test-step-end -->

<!-- test-step-start: Verify -->

```go-e2e-step
kubectl get pods -l app=postgresql
```

<!-- test-step-end -->

<!-- test-end -->
```

This generates one file `generated_database_tests_test.go` with two `It()` blocks: "Setup" and "Verify".

When no `test-step-start/end` is used inside a `test-start/end` block, all steps go into a single `It()` named after the test-start name. Blocks without any markers fall back to using the source filename.

## CLI Commands

| Command | Description |
|---------|-------------|
| `docsyncer init` | Create a default `docsyncer.yaml` in the current directory |
| `docsyncer generate` | Scan docs, extract blocks, generate test files |
| `docsyncer validate` | Validate your `docsyncer.yaml` for errors |

### Global Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Config file path (default: `docsyncer.yaml`) |
| `--verbose`, `-v` | Enable debug-level logging |
| `--dry-run` | Parse and convert but don't write files |

## Configuration Reference

The full configuration schema is documented in [`PLAN.md`](PLAN.md). Here's a minimal example:

```yaml
input:
  directories: ["docs"]
  include: ["*.md"]

tags:
  step_tags: ["go-e2e-step"]

output:
  directory: "tests/e2e/generated"
  package_name: "e2e_generated"
  file_prefix: "generated_"
  file_suffix: "_test.go"
  build_tag: "e2e"              # adds //go:build e2e to generated files (optional)
```

### Key Configuration Sections

| Section | Purpose |
|---------|---------|
| `input` | Directories to scan, include/exclude patterns, recursive flag |
| `tags` | Step tags, test-start/end markers, step-start/end markers, attribute name mappings |
| `output` | Output directory, file naming, package name, build tag, clean-before-generate |
| `templates` | Template directory, default template, override support. Leave `directory` empty to use the embedded default |
| `commands` | Default timeout, expected exit code, blocked patterns, shell config |
| `logging` | Log level (`debug`, `info`, `warn`, `error`) |

## Generated Output Example

Given this Markdown:

```markdown
# API Gateway Setup

<!-- test-start: Redis deployment E2E -->

```go-e2e-step step-name="Deploy Redis via Helm"
helm install redis bitnami/redis --set auth.enabled=false
```

```go-e2e-step timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=120s
```

<!-- test-end -->
```

DocSyncer generates `generated_redis_deployment_e2e_test.go`:

```go
package e2e_generated

import (
    "context"
    "os/exec"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

// Auto-generated by docsyncer from: docs/redis.md
// Source type: markdown
// DO NOT EDIT — this file is regenerated on every run.

var _ = Describe("Redis deployment E2E", func() {
    It("Redis deployment E2E", func() {
        {
            By("Deploy Redis via Helm")
            cmd := exec.Command("helm", "install", "redis", "bitnami/redis", "--set", "auth.enabled=false")
            output, err := cmd.CombinedOutput()
            Expect(err).ToNot(HaveOccurred(), string(output))
        }
        {
            By("kubectl wait")
            dur, err := time.ParseDuration("60s")
            Expect(err).ToNot(HaveOccurred())
            ctx, cancel := context.WithTimeout(context.Background(), dur)
            defer cancel()
            cmd := exec.CommandContext(ctx, "kubectl", "wait", "--for=condition=ready", "pod",
                "-l", "app.kubernetes.io/name=redis", "--timeout=120s")
            output, err := cmd.CombinedOutput()
            Expect(err).ToNot(HaveOccurred(), string(output))
        }
    })
})
```

## Project Structure

```
GoE2E-DocSyncer/
├── cmd/docsyncer/          # CLI entry point
├── internal/
│   ├── domain/             # Core types (ParsedDocument, TestSpec, errors)
│   ├── config/             # YAML config loading, defaults, validation
│   ├── scanner/            # File discovery
│   ├── parser/             # Parser interface, registry, format parsers
│   ├── converter/          # Block → TestStep conversion, Go code generation
│   ├── template/           # Template engine with custom functions
│   │   └── embedded/       # Embedded default template (for go run support)
│   ├── generator/          # Pipeline orchestrator
│   └── cli/                # Cobra CLI commands
├── templates/              # Default Ginkgo template (also embedded at build time)
├── testdata/               # Test fixtures (markdown, asciidoc)
├── docsyncer.yaml          # Example configuration
└── PLAN.md                 # Architecture and design document
```

## Development

```bash
make test           # Run all tests
make test-verbose   # Run with verbose Ginkgo output
make vet            # Run go vet
make lint           # Run golangci-lint
make check          # vet + test + build
make tidy           # go mod tidy
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [goldmark](https://github.com/yuin/goldmark) | CommonMark Markdown AST parsing |
| [cobra](https://github.com/spf13/cobra) | CLI framework |
| [log/slog](https://pkg.go.dev/log/slog) | Structured logging (Go standard library) |
| [yaml.v3](https://gopkg.in/yaml.v3) | YAML configuration |
| [ginkgo/v2](https://github.com/onsi/ginkgo) | BDD test framework |
| [gomega](https://github.com/onsi/gomega) | Assertion library |

## License

See [LICENSE](LICENSE) for details.
