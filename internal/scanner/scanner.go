package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
)

// Scanner discovers documentation files in the project tree.
type Scanner interface {
	Scan(rootDir string, patterns []string, excludes []string) ([]string, error)
}

// FileScanner implements Scanner using filepath.WalkDir.
type FileScanner struct {
	Recursive bool
}

// NewScanner creates a new FileScanner.
func NewScanner(recursive bool) *FileScanner {
	return &FileScanner{Recursive: recursive}
}

// Scan walks rootDir and returns sorted file paths matching any of the given
// glob patterns while excluding paths that match any exclude pattern.
func (s *FileScanner) Scan(rootDir string, patterns []string, excludes []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get path relative to rootDir for pattern matching
		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			relPath = path
		}

		if d.IsDir() {
			// Skip non-root directories if not recursive
			if !s.Recursive && relPath != "." {
				return filepath.SkipDir
			}
			// Check if directory matches any exclude pattern
			for _, exc := range excludes {
				matched, _ := filepath.Match(exc, relPath)
				if matched {
					return filepath.SkipDir
				}
				// Also check with trailing separator patterns
				if matchGlob(relPath, exc) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches any exclude pattern
		for _, exc := range excludes {
			if matchGlob(relPath, exc) {
				return nil
			}
		}

		// Check if file matches any include pattern
		for _, pattern := range patterns {
			if matchGlob(relPath, pattern) {
				files = append(files, path)
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return nil, domain.NewError("scan", rootDir, 0, "failed to scan directory", err)
	}

	sort.Strings(files)
	return files, nil
}

// matchGlob matches a path against a glob pattern, supporting ** for recursive matching.
func matchGlob(path, pattern string) bool {
	// Handle ** patterns by splitting and matching parts
	if strings.Contains(pattern, "**") {
		// Split pattern on **
		parts := strings.SplitN(pattern, "**", 2)
		prefix := strings.TrimSuffix(parts[0], string(filepath.Separator))
		suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))

		if prefix != "" {
			if !strings.HasPrefix(path, prefix) {
				return false
			}
			// Remove prefix from path for suffix matching
			path = strings.TrimPrefix(path, prefix)
			path = strings.TrimPrefix(path, string(filepath.Separator))
		}

		if suffix == "" {
			return true
		}

		// Try matching suffix against each possible subpath
		pathParts := strings.Split(path, string(filepath.Separator))
		for i := range pathParts {
			subPath := strings.Join(pathParts[i:], string(filepath.Separator))
			matched, _ := filepath.Match(suffix, subPath)
			if matched {
				return true
			}
		}
		return false
	}

	// Simple glob match
	matched, _ := filepath.Match(pattern, filepath.Base(path))
	if matched {
		return true
	}
	// Also try against the full relative path
	matched, _ = filepath.Match(pattern, path)
	return matched
}
