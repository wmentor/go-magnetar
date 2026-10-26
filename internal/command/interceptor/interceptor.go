package interceptor

import (
	"regexp"
	"runtime"
	"slices"
	"strings"
)

// PatternType identifies the source of a pattern.
type PatternType int

const (
	PatternBlock     PatternType = iota // From blockList (internal/tools/generic/generic.go)
	PatternDangerous                    // From dangerousPatterns (internal/plugins/chatcmd/exec/plugin.go)
)

// Pattern holds a single security pattern with its type flag.
type Pattern struct {
	Regexp *regexp.Regexp
	Type   PatternType
}

// Interceptor holds the combined security patterns for command validation.
type Interceptor struct {
	patterns []Pattern
}

// New creates a new Interceptor instance with combined security patterns.
func New() *Interceptor {
	// Add blockList patterns with PatternBlock flag
	// For Linux/MacOS commands: case-sensitive (command names are lowercase)
	// For dangerous patterns: case-insensitive where needed (bash, sh, zsh, curl, wget)
	patterns := []Pattern{ //nolint:prealloc // ok.
		{Regexp: regexp.MustCompile(`\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)*(/\s*$|/\s+)`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\|\s*bash\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\|\s*sh\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\|\s*zsh\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bsudo\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bsu\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bmkfs\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bchmod\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bchown\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bdscl\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bfdisk\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bformat\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bdseditgroup\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bbrew\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bdpkg\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bapt\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bcargo\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\brpm\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bnpm\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bssh\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bapt\-get\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bgroupadd\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\busermod\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bgpasswd\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\buseradd\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\buserdel\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\badduser\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bdeluser\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bpasswd\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bsystemctl\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bsysadminctl\b`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bdd\s+.*of=/dev/`), Type: PatternBlock},
		{Regexp: regexp.MustCompile(`(?i)\bgit\s+(commit|push|rebase|pull|cherry-pick|amend|reset|stash|clean|reflog|clone)`), Type: PatternBlock},
	}

	// Add dangerous commands from issue #205: rd, rmdir, reboot, shutdown, del, assoc
	// Case-insensitive on Windows (where these are native), case-sensitive on Unix.
	prefix := ""
	if runtime.GOOS == "windows" {
		prefix = "(?i)"
	}
	for _, cmd := range []string{"rd", "rmdir", "reboot", "shutdown", "del", "assoc"} {
		patterns = append(patterns, Pattern{Regexp: regexp.MustCompile(prefix + `\b` + cmd + `\b`), Type: PatternBlock})
	}

	// Add dangerousPatterns with PatternDangerous flag
	// These patterns check for dangerous combinations that need case-insensitive matching
	patterns = append(patterns,
		Pattern{Regexp: regexp.MustCompile(`(?i)\brm\s+-rf\s+[/$]|\brm\s+-rf\s+\*`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bdd\b.*\bif=\s*`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bmkfs\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bfdisk\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bbash\s+-c\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bsh\s+-c\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)\bzsh\s+-c\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`(?i)(curl|wget)\b.*\|\s*(bash|sh|zsh)\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\brd\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\brmdir\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\breboot\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\bshutdown\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\bdel\b`), Type: PatternDangerous},
		Pattern{Regexp: regexp.MustCompile(`\bassoc\b`), Type: PatternDangerous},
	)

	return &Interceptor{
		patterns: patterns,
	}
}

// splitShellCommand splits a shell command into individual command segments.
// It handles semicolons, &&, ||, and pipes as separators.
// Note: We split on | to check each command in a pipeline, but this has limitations
// with complex commands. A more sophisticated parser would be needed for full accuracy.
func splitShellCommand(command string) []string {
	// Split on ;, &&, || first
	re1 := regexp.MustCompile(`;\s*|\|\||&&`)
	parts := re1.Split(command, -1)

	var segments []string
	for _, part := range parts {
		// Then split each part on | (for pipelines within a command segment)
		re2 := regexp.MustCompile(`\|`)
		subParts := re2.Split(part, -1)
		for _, subPart := range subParts {
			trimmed := strings.TrimSpace(subPart)
			if trimmed != "" {
				segments = append(segments, trimmed)
			}
		}
	}

	return segments
}

// IsBlocked checks if the command matches any security pattern (blockList or dangerousPatterns).
// Returns true if the command should be blocked.
func (i *Interceptor) IsBlocked(command string) bool {
	segments := splitShellCommand(command)

	// Check against all patterns
	for _, p := range i.patterns {
		if p.Type == PatternBlock { //nolint:staticcheck // ok.
			if slices.ContainsFunc(segments, p.Regexp.MatchString) {
				return true
			}
		} else if p.Type == PatternDangerous {
			if p.Regexp.MatchString(command) {
				return true
			}
		}
	}

	return false
}

// IsDangerous checks if the command matches any dangerous pattern (only dangerousPatterns).
// Returns true if the command contains dangerous pipeline patterns (e.g., curl | bash).
func (i *Interceptor) IsDangerous(command string) bool {
	// Only check dangerousPatterns, not blockList
	for _, p := range i.patterns {
		if p.Type == PatternDangerous {
			if p.Regexp.MatchString(command) {
				return true
			}
		}
	}

	return false
}
