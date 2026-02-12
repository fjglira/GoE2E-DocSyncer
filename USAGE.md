# Usage Guide

This guide covers two things:

1. **Trying out docsyncer** — run it against the included test fixtures to see what it produces
2. **Using docsyncer in your own project** — step-by-step integration instructions

---

## Part 1: Try It Out (This Repo)

### 1.1 Build the binary

```bash
make build
# Output: bin/docsyncer
```

### 1.2 Run against the included test fixtures

The repo ships with example documentation files in `testdata/`. A ready-to-use config is included at `docsyncer-demo.yaml`.

**Dry run** (preview output without writing files):

```bash
bin/docsyncer generate --config docsyncer-demo.yaml --dry-run --verbose
```

This will show you:
- Which files are discovered
- How many tagged blocks are extracted from each file
- The full generated Go test code for each file

**Actual generation** (writes test files to `tests/e2e/generated/`):

```bash
bin/docsyncer generate --config docsyncer-demo.yaml --verbose
```

Check the generated output:

```bash
ls tests/e2e/generated/
# generated_simple_deployment_test_test.go   (from simple.md, named after test-start)
# generated_infrastructure_provisioning_test.go  (from multi-step.md, 1st test-start)
# generated_application_deployment_test.go       (from multi-step.md, 2nd test-start)
# generated_sample_test.go     (from AsciiDoc — no test-start, uses filename)

cat tests/e2e/generated/generated_infrastructure_provisioning_test.go
cat tests/e2e/generated/generated_application_deployment_test.go
```

### 1.3 What to look for

**`generated_simple_deployment_test_test.go`** — Single `It()` block with 3 steps from `testdata/markdown/simple.md`, file named after `test-start: Simple deployment test`

**`generated_infrastructure_provisioning_test.go`** — Two `It()` blocks ("Setup Database" and "Wait for Ready") from the `test-step-start/end` groups within `test-start: Infrastructure provisioning` in `testdata/markdown/multi-step.md`

**`generated_application_deployment_test.go`** — Single `It()` block with 3 steps from `test-start: Application deployment` (no step groups, so all steps in one `It()`)

**Things to verify:**
- [ ] Each `test-start`/`test-end` pair produces a **separate output file** named after the test-start name
- [ ] Output files contain `package e2e_generated`
- [ ] `Describe()` block uses the test-start name (or document heading if no test-start)
- [ ] `test-step-start`/`test-step-end` pairs within a test-start block produce separate `It()` blocks
- [ ] Each step has a `By()` with the step name
- [ ] Commands with `timeout` use `context.WithTimeout` and `exec.CommandContext`
- [ ] Simple commands use `exec.Command("cmd", "arg1", "arg2")`
- [ ] Complex commands (pipes) use `exec.Command("/bin/sh", "-c", "...")`

### 1.4 Validate the config

```bash
bin/docsyncer validate --config docsyncer-demo.yaml
```

### 1.5 Run the unit tests

```bash
make check          # vet + test + build (all 76 specs)
make test-verbose   # verbose Ginkgo output
```

### 1.6 Clean up generated files

```bash
make clean
# or: rm -rf tests/e2e/generated
```

---

## Part 2: Use in Your Own Go Project

### 2.1 Install docsyncer

```bash
go install github.com/frherrer/GoE2E-DocSyncer/cmd/docsyncer@latest
```

Or copy the binary from a local build:

```bash
# In the GoE2E-DocSyncer repo:
make build
cp bin/docsyncer /usr/local/bin/
```

### 2.2 Initialize configuration

In your Go project's root:

```bash
cd /path/to/your-go-project
docsyncer init
# Creates docsyncer.yaml with defaults
```

### 2.3 Write documentation with test tags

Create a doc file (e.g., `docs/deployment.md`) using tagged code blocks:

````markdown
# Deployment Guide

<!-- test-start: Smoke test -->

## Deploy the application

```go-e2e-step step-name="Apply manifests"
kubectl apply -f ./k8s/
```

## Verify it works

```go-e2e-step step-name="Check pods are ready" timeout=60s
kubectl wait --for=condition=ready pod -l app=myapp --timeout=120s
```

```go-e2e-step step-name="Health check" timeout=10s
curl -f http://localhost:8080/healthz
```

<!-- test-end -->
````

**Key syntax:**
- The code fence language must match one of your `tags.step_tags` (default: `go-e2e-step`)
- Attributes go in the info string: `step-name="..."`, `timeout=60s`, `exit-code=1`
- `<!-- test-start: Name -->` / `<!-- test-end -->` — each pair produces a **separate output file** named after the test-start name
- `<!-- test-step-start: Name -->` / `<!-- test-step-end -->` — within a test-start block, each pair produces a separate `It()` block
- Without `test-step-start/end`, all steps in a `test-start/end` block go into a single `It()`
- Blocks outside any test-start/test-end pair are grouped by source filename

### 2.4 Configure docsyncer.yaml

Edit the generated config to match your project:

```yaml
input:
  directories:
    - "docs"                  # Where your docs live
    - "documentation"         # Add more directories as needed
  include:
    - "*.md"
  exclude:
    - "**/CHANGELOG.md"

tags:
  step_tags:
    - "go-e2e-step"           # Must match your code fence language tags

output:
  directory: "tests/e2e/generated"
  package_name: "e2e_generated"
  file_prefix: "generated_"
  file_suffix: "_test.go"
  build_tag: "e2e"               # adds //go:build e2e to generated files (optional)
  clean_before_generate: true     # Wipes old generated files first

templates:
  directory: ""                   # empty = use embedded default template
  default: "ginkgo_default"

commands:
  default_timeout: "30s"
  blocked_patterns:
    - "rm -rf /"
```

### 2.5 Using `go run` (no install needed)

You can run docsyncer directly from another project without installing it. The embedded default template means no local `templates/` directory is required:

```bash
go run github.com/frherrer/GoE2E-DocSyncer/cmd/docsyncer@latest generate \
    --config docsyncer.yaml --verbose
```

Set `templates.directory: ""` in your config to use the embedded template.

### 2.6 Generate tests

```bash
# Preview first:
docsyncer generate --dry-run --verbose

# Generate for real:
docsyncer generate --verbose
```

### 2.7 Run the generated tests

The generated files are standard Ginkgo test files. Run them with:

```bash
# If you have ginkgo CLI:
ginkgo ./tests/e2e/generated/

# Or with go test:
go test ./tests/e2e/generated/ -v
```

**Important:** The generated tests execute real shell commands (`kubectl`, `helm`, `curl`, etc.), so they require:
- A running Kubernetes cluster (for kubectl/helm commands)
- Network access (for curl commands)
- Any tools referenced in the docs to be installed

These are E2E tests — they are meant to run against a real environment.

### 2.8 Integrate into your workflow

Add to your `Makefile`:

```makefile
# Generate E2E tests from documentation
generate-e2e:
	docsyncer generate --verbose

# Validate docsyncer config
validate-docs:
	docsyncer validate

# Run generated E2E tests (requires a running cluster)
test-e2e: generate-e2e
	go test ./tests/e2e/generated/ -v -count=1
```

Add to `.gitignore` (optional — some teams prefer committing generated tests):

```
tests/e2e/generated/
```

---

## Part 3: Supported Formats Quick Reference

### Markdown (.md)

````markdown
```go-e2e-step step-name="My step" timeout=30s
kubectl get pods
```
````

### AsciiDoc (.adoc)

```asciidoc
[source,go-e2e-step,step-name="My step"]
----
kubectl get pods
----
```

---

## Part 4: Troubleshooting

### "No documentation files found"

- Check that `input.directories` in `docsyncer.yaml` points to directories that exist
- Check that `input.include` patterns match your file extensions
- Use `--verbose` to see which directories are being scanned

### "No tagged blocks found"

- Verify your code fence language matches one of your `tags.step_tags`
- For Markdown: the tag goes right after the triple backticks (`` ```go-e2e-step ``)

### "template not found"

- If using a custom templates directory, check that `templates.directory` points to a directory with `.tmpl` files
- Set `templates.directory` to `""` (empty) to use the built-in embedded template — this is recommended when running via `go run`
- The `templates.default` value should match a filename (without `.tmpl` extension)

### "command blocked by security policy"

- The command contains a pattern from `commands.blocked_patterns`
- If intentional, remove the pattern from your config

### Generated code doesn't compile

- Run with `--dry-run --verbose` to inspect the raw output
- Check that the template produces valid Go syntax
- Ensure `output.package_name` is a valid Go identifier
