package converter

import (
	"fmt"
	"strings"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
)

// GenerateGoCode converts a shell command string into Go code using os/exec.
func GenerateGoCode(command string, expectedExit int, timeout string, cmdCfg *config.CommandConfig) string {
	command = strings.TrimSpace(command)
	lines := strings.Split(command, "\n")

	// Multi-line commands are joined with &&
	if len(lines) > 1 {
		var trimmed []string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" {
				trimmed = append(trimmed, l)
			}
		}
		command = strings.Join(trimmed, " && ")
	}

	var goCode string
	if isComplexCommand(command) {
		goCode = generateShellCommand(command, cmdCfg.Shell, cmdCfg.ShellFlag)
	} else {
		goCode = generateSimpleCommand(command)
	}

	// Wrap with timeout if non-default
	if timeout != "" && timeout != "0" && timeout != "0s" {
		goCode = wrapWithTimeout(goCode, timeout)
	}

	// Handle expected exit code
	if expectedExit != 0 {
		goCode = wrapWithExpectedExit(goCode, expectedExit)
	}

	return goCode
}

// isComplexCommand determines if a command needs shell execution (pipes, redirects, etc.).
func isComplexCommand(cmd string) bool {
	complexChars := []string{"|", "&&", "||", ";", ">", "<", ">>", "$(", "`", "&"}
	for _, c := range complexChars {
		if strings.Contains(cmd, c) {
			return true
		}
	}
	return false
}

// generateSimpleCommand generates exec.Command for simple commands.
func generateSimpleCommand(command string) string {
	parts := shellSplit(command)
	if len(parts) == 0 {
		return ""
	}

	if len(parts) == 1 {
		return fmt.Sprintf(`cmd := exec.Command(%q)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), string(output))`, parts[0])
	}

	// Format args
	args := make([]string, len(parts))
	for i, p := range parts {
		args[i] = fmt.Sprintf("%q", p)
	}

	return fmt.Sprintf(`cmd := exec.Command(%s)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), string(output))`, strings.Join(args, ", "))
}

// generateShellCommand generates exec.Command using a shell for complex commands.
func generateShellCommand(command, shell, shellFlag string) string {
	return fmt.Sprintf(`cmd := exec.Command(%q, %q, %q)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), string(output))`, shell, shellFlag, command)
}

// wrapWithTimeout wraps Go code with a context timeout.
func wrapWithTimeout(goCode, timeout string) string {
	return fmt.Sprintf(`dur, err := time.ParseDuration(%q)
			Expect(err).ToNot(HaveOccurred())
			ctx, cancel := context.WithTimeout(context.Background(), dur)
			defer cancel()
			%s`, timeout, strings.Replace(goCode, "exec.Command(", "exec.CommandContext(ctx, ", 1))
}

// wrapWithExpectedExit modifies the assertion to check for a specific exit code.
func wrapWithExpectedExit(goCode string, expectedExit int) string {
	// Replace the standard assertion with exit code check
	return strings.Replace(goCode,
		"Expect(err).ToNot(HaveOccurred(), string(output))",
		fmt.Sprintf(`if exitErr, ok := err.(*exec.ExitError); ok {
				Expect(exitErr.ExitCode()).To(Equal(%d), string(output))
			} else {
				Expect(err).ToNot(HaveOccurred(), string(output))
			}`, expectedExit),
		1)
}

// shellSplit splits a command string into arguments, respecting quotes.
func shellSplit(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else {
			if c == '"' || c == '\'' {
				inQuote = true
				quoteChar = c
			} else if c == ' ' || c == '\t' {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
