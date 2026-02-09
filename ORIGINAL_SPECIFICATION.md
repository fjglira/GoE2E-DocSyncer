# Original Project Specification

> **Date**: December 2, 2025
> **Project**: Doc-to-Ginkgo E2E Test Generator
> **Status**: Specification for V1 Implementation

---

## Project Kickoff: Doc-to-Ginkgo E2E Test Generator

### I. Project Goal & Scope

The primary goal is to create a reliable Go-based command-line tool (the "Generator") that processes our existing technical documentation files (primarily Markdown and ascii docs) and automatically generates executable Ginkgo E2E test files that replicate the documented steps. This tool must reduce code duplication and ensure documentation consistency with the E2E test suite.

### II. Input & Output Specification

#### A. Input Format (Documentation)

**Source**: Markdown files (.md) or ASCII docs .adoc located within the input directory directory.

**Test Step Identification**: Execution steps will be defined using Markdown Code Fences and ASCII code blocks and indef block with a custom language tag.

**Tag**: `go-e2e-step` (This signifies content that must be converted into executable Go code).

**Syntax Example**:
```markdown
### Deploy the application
The following command should be run to deploy the component:

```go-e2e-step
kubectl apply -f ./config/manifests.yaml
```

Wait for the deployment to complete.
```

**Test Step Identification**: The Generator must infer the Describe or Context block name from a tag provided in the actual step

**Test name Identification**: the generator should use the init tag with the test name as a test name and it should know the end of the current test by end tag identificator of the test

#### B. Output Format (Generated Test)

**Language/Framework**: Standard Go (_test.go file suffix) using the Ginkgo/Gomega testing framework.

**Location**: Generated files must be placed in a designated directory, e.g., `/tests/e2e/generated_doc_tests/`.

**Structure**: Each parsed documentation file should map to a single generated test file. Each go-e2e-step block should map to an It() block or a sequence of assertions within a single It() block.

### III. Functional Requirements (The Generator Tool)

**Parsing**: Implement a robust Markdown parser annd ASCII parse capable of traversing the AST to locate all code blocks tagged with go-e2e-step. Please only focus on open source projects that are currently mantain.

**Shell Command Conversion**: Implement a function to convert the shell commands inside the tagged block into Go code that:
- Uses os/exec or a dedicated testing utility (e.g., a wrapper around os/exec or Gexec) to run the command.
- Uses gomega.Expect() to assert an exit code of 0 (success) by default, unless otherwise tagged in the documentation.

**Templating & Generation**: Utilize the text/template package to inject the converted steps into the Ginkgo boilerplate structure.

**Formatting**: Use the go/format package before writing the final output to ensure the generated code is Go-linting compliant.

**Cleanliness**: The tool must have a cleanup step to remove all previously generated files before a new generation run.

### IV. Technical Design Decisions

**Tool Implementation**: The tool will be a standalone Go binary runnable via `go run ./cmd/generator`.

**Error Handling**: Logging must be verbose, detailing which file and line number failed parsing or transformation.

**External Dependencies**: Limit external dependencies to robust, widely adopted Go libraries (e.g., Markdown parsers, text/template).

### V. Next Steps

**Spike**: Investigate the best Go Markdown library for reliable AST traversal and tagging and the library also for ASCII.

**Define Abstraction**: Design the Go struct (TestSpec) that will hold the parsed data (Test Name, Command String, Expected Result) before it is passed to the templating engine.

**Create Boilerplate Template**: Draft the initial ginkgo_template.tmpl file. The template can be customizable or accept different kind of template in the test using hidden tag in the doc

**Goal for V1**: Successfully generate a single .go test file from a simple .md or ascii file containing one go-e2e-step block.

---

## Implementation Notes from Development Session

### Key Decisions Made During Implementation

1. **Markdown Parser Selection**: Chose **goldmark** over blackfriday for:
   - Full CommonMark compliance
   - Clean AST structure using interfaces
   - Better extensibility for custom parsing needs

2. **AsciiDoc Parser Selection**: Initially researched **libasciidoc** but implemented a **simple regex-based parser** for V1 to avoid complexity while meeting core requirements.

3. **Architecture Pattern**: Implemented clean modular architecture with:
   - Separate packages for each concern (parser, converter, template, etc.)
   - Interface-based design for extensibility
   - Type-safe operations throughout

4. **Security Considerations**: Added validation to prevent execution of dangerous commands like `rm -rf`, `format`, etc.

### Extensions Beyond Original Spec

During implementation, several enhancements were added that exceeded the original requirements:

1. **Advanced Attribute System**: Support for custom step names, timeouts, and expected exit codes
2. **Smart Command Recognition**: Intelligent naming based on command type (kubectl, docker, curl)
3. **Comprehensive CLI**: Full featured command-line interface with all necessary options
4. **Template Flexibility**: Template engine supporting custom templates
5. **Progress Tracking**: Detailed logging and progress reporting
6. **Batch Processing**: Support for processing multiple files simultaneously

### Files Created During Implementation

```
PROJECT_STATUS.md           # Comprehensive progress tracking (this session)
ORIGINAL_SPECIFICATION.md   # This file - original requirements
README.md                   # User-facing documentation
cmd/generator/main.go       # CLI entry point
internal/types/types.go     # Core data structures
internal/parser/parser.go   # Main parser orchestrator
internal/parser/markdown.go # Goldmark-based Markdown parser
internal/parser/asciidoc_simple.go # Simple AsciiDoc parser
internal/converter/converter.go    # Document to TestSpec conversion
internal/converter/command.go      # Shell command to Go conversion
internal/template/engine.go        # Template rendering engine
internal/generator/generator.go    # Main generation logic
internal/cli/root.go              # CLI interface
templates/ginkgo_default.tmpl     # Default Ginkgo template
docs/example-deployment.md       # Example Markdown documentation
docs/docker-build.adoc           # Example AsciiDoc documentation
go.mod                           # Go module definition
```

### Success Criteria Met

✅ **Core Requirements**:
- Parses Markdown and AsciiDoc files
- Identifies go-e2e-step tagged code blocks
- Converts shell commands to Go/Ginkgo test code
- Generates properly formatted _test.go files
- Uses text/template for code generation
- Includes go/format for code formatting
- Has cleanup functionality
- Provides verbose error logging

✅ **Technical Requirements**:
- Standalone Go binary via `go run ./cmd/generator`
- Robust, maintained open-source libraries only
- Minimal external dependencies
- Clean, modular architecture

✅ **V1 Goal**: Successfully generates multiple .go test files from .md and .adoc files containing go-e2e-step blocks

---

*This specification was fully implemented and exceeded in the development session of December 2, 2025*