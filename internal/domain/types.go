package domain

// ParsedDocument holds the result of parsing a single document file.
type ParsedDocument struct {
	FilePath string
	FileType string            // "markdown", "asciidoc", "plaintext", etc.
	Blocks   []CodeBlock       // All extracted code blocks (tagged ones)
	Headings []Heading         // Document structure (for context inference)
	Metadata map[string]string // Any document-level metadata found
}

// CodeBlock represents a single tagged code block extracted from a document.
type CodeBlock struct {
	Tag        string            // The matched tag (e.g. "go-e2e-step")
	Content    string            // Raw content of the block
	LineNumber int               // 1-based line number in source
	Attributes map[string]string // Key-value attributes from the fence info
	Context    string            // Nearest heading / section title
	TestGroup  string            // test-start group name (empty if ungrouped)
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
	Name          string
	Command       string
	GoCode        string // Generated Go code for this step
	ExpectedExit  int
	Timeout       string
	LineNumber    int
	SkipOnFailure bool
	RetryCount    int    // Number of retries (0 = no retry)
	RetryInterval string // Duration between retries (e.g. "2s")
}
