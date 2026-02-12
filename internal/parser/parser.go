package parser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
)

// Parser extracts code blocks from a document.
type Parser interface {
	Parse(filePath string, content []byte, tags []string) (*domain.ParsedDocument, error)
	SupportedExtensions() []string
}

// ParserRegistry maps file extensions to parsers.
type ParserRegistry interface {
	Register(parser Parser)
	ParserFor(extension string) (Parser, error)
}

// DefaultRegistry is a thread-safe parser registry with fallback support.
type DefaultRegistry struct {
	mu       sync.RWMutex
	parsers  map[string]Parser
	fallback Parser
}

// NewRegistry creates a new DefaultRegistry.
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser to the registry for each of its supported extensions.
func (r *DefaultRegistry) Register(p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ext := range p.SupportedExtensions() {
		ext = strings.TrimPrefix(ext, ".")
		r.parsers[ext] = p
	}
}

// SetFallback sets the fallback parser for unregistered extensions.
func (r *DefaultRegistry) SetFallback(p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = p
}

// ParserFor returns the parser registered for the given file extension.
// If no parser is found, it returns the fallback parser if set.
func (r *DefaultRegistry) ParserFor(extension string) (Parser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := strings.TrimPrefix(extension, ".")
	if p, ok := r.parsers[ext]; ok {
		return p, nil
	}
	if r.fallback != nil {
		return r.fallback, nil
	}
	return nil, fmt.Errorf("no parser registered for extension %q", extension)
}
