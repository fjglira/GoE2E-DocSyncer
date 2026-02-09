package converter

import (
	"fmt"
	"strings"
)

// ValidateCommand checks if a command matches any blocked patterns.
func ValidateCommand(command string, blockedPatterns []string) error {
	for _, pattern := range blockedPatterns {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("command blocked by security policy: contains %q — if this is intentional, remove it from commands.blocked_patterns in docsyncer.yaml", pattern)
		}
	}
	return nil
}
