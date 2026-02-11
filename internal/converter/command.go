package converter

import (
	"fmt"
	"strings"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
)

// GenerateGoCode converts a shell command string into Go code using os/exec.
func GenerateGoCode(command string, expectedExit int, timeout string, retryCount int, retryInterval string, cmdCfg *config.CommandConfig) string {
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

	// Handle expected exit code
	if expectedExit != 0 {
		goCode = wrapWithExpectedExit(goCode, expectedExit)
	}

	// Wrap with retry if specified
	if retryCount > 0 {
		goCode = wrapWithRetry(goCode, retryCount, retryInterval)
	}

	// Wrap with timeout if non-default (outermost — timeout applies across all retry attempts)
	if timeout != "" && timeout != "0" && timeout != "0s" {
		goCode = wrapWithTimeout(goCode, timeout)
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

// wrapWithRetry wraps Go code with a retry loop.
// retryCount is the number of retries (e.g. 3 means 4 total attempts: 1 initial + 3 retries).
func wrapWithRetry(goCode string, retryCount int, retryInterval string) string {
	totalAttempts := retryCount + 1

	// Extract the assertion line and the command setup lines
	// Replace "output, err :=" with "lastOutput, lastErr =" and wrap in loop
	retryCode := goCode

	// Replace variable declarations to use pre-declared vars
	retryCode = strings.Replace(retryCode, "output, err := cmd.CombinedOutput()", "lastOutput, lastErr = cmd.CombinedOutput()", 1)

	// Replace assertion to check lastErr/lastOutput
	retryCode = strings.Replace(retryCode, "Expect(err).ToNot(HaveOccurred(), string(output))", "", 1)

	// Also handle expected exit code assertion pattern
	retryCode = strings.Replace(retryCode,
		"Expect(err).ToNot(HaveOccurred(), string(lastOutput))", "", 1)

	// For expected exit code, replace the exit code check block too
	hasExitCheck := strings.Contains(goCode, "exitErr, ok := err.(*exec.ExitError)")
	if hasExitCheck {
		// Replace the exit error check that uses output
		retryCode = strings.Replace(retryCode, "if exitErr, ok := err.(*exec.ExitError); ok {", "if exitErr, ok := lastErr.(*exec.ExitError); ok {", 1)
		// The exitErr check block uses output, replace it
		retryCode = strings.Replace(retryCode, "string(output)", "string(lastOutput)", -1)
		// Extract the assertion block - we'll put it after the loop
		// Find and remove the exit check block from the loop body
		exitCheckStart := strings.Index(retryCode, "if exitErr, ok := lastErr.(*exec.ExitError)")
		if exitCheckStart >= 0 {
			exitBlock := retryCode[exitCheckStart:]
			retryCode = retryCode[:exitCheckStart]
			// Build the retry loop with the exit check after
			return fmt.Sprintf(`{
			var lastOutput []byte
			var lastErr error
			for attempt := 1; attempt <= %d; attempt++ {
				%s
				if lastErr == nil {
					break
				}
				if attempt <= %d {
					time.Sleep(%s)
				}
			}
			%s
		}`, totalAttempts, strings.TrimSpace(retryCode), retryCount, formatDuration(retryInterval), strings.TrimSpace(exitBlock))
		}
	}

	// Standard case (no expected exit code)
	return fmt.Sprintf(`{
			var lastOutput []byte
			var lastErr error
			for attempt := 1; attempt <= %d; attempt++ {
				%s
				if lastErr == nil {
					break
				}
				if attempt <= %d {
					time.Sleep(%s)
				}
			}
			Expect(lastErr).ToNot(HaveOccurred(), string(lastOutput))
		}`, totalAttempts, strings.TrimSpace(retryCode), retryCount, formatDuration(retryInterval))
}

// formatDuration converts a duration string like "5s" into a Go expression like "5 * time.Second".
func formatDuration(d string) string {
	// Parse simple duration formats: Ns, Nm, Nms
	d = strings.TrimSpace(d)
	if strings.HasSuffix(d, "ms") {
		num := strings.TrimSuffix(d, "ms")
		return fmt.Sprintf("%s * time.Millisecond", num)
	}
	if strings.HasSuffix(d, "s") {
		num := strings.TrimSuffix(d, "s")
		return fmt.Sprintf("%s * time.Second", num)
	}
	if strings.HasSuffix(d, "m") {
		num := strings.TrimSuffix(d, "m")
		return fmt.Sprintf("%s * time.Minute", num)
	}
	// Fallback: use time.ParseDuration at runtime
	return fmt.Sprintf("func() time.Duration { d, _ := time.ParseDuration(%q); return d }()", d)
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
