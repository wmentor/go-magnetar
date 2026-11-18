package shell

import (
	"os/exec"
	"strings"
)

// CheckCommand validates if a string is a valid shell command.
// Returns true if the command is valid, false otherwise.
func CheckCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}

	// First check: does it contain shell-specific syntax?
	if hasShellSyntax(command) {
		return true
	}

	// If no shell syntax, check if it looks like a command with valid arguments
	return looksLikeCommand(trimmed)
}

// looksLikeCommand checks if a string looks like a command with valid arguments.
func looksLikeCommand(cmd string) bool {
	// Extract the command name (first word)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	// Check if it's a shell builtin
	if isShellBuiltin(cmdName) {
		return true
	}

	// Check if it's a valid command in PATH
	if _, err := exec.LookPath(cmdName); err != nil {
		return false
	}

	// Command exists, now validate arguments
	if len(parts) == 1 {
		// No arguments - valid
		return true
	}

	// Has arguments - check if they look like valid shell arguments
	return validArguments(parts[1:])
}

func validArguments(args []string) bool {
	if len(args) == 0 {
		return true
	}

	// Check each argument
	for _, arg := range args {
		if !isValidArgument(arg) {
			return false
		}
	}

	return true
}

func isValidArgument(arg string) bool {
	// Empty argument is invalid
	if arg == "" {
		return false
	}

	// Check if it contains backticks - validate command substitution
	if strings.Contains(arg, "`") {
		return containsValidCommandSubstitution(arg)
	}

	// Check if it contains $() - validate command substitution
	if strings.Contains(arg, "$(") {
		return containsValidCommandSubstitution(arg)
	}

	// Check for special directory paths (. and ..)
	if arg == "." || arg == ".." {
		return true
	}

	// Check if it's a path (contains / or starts with ./ or ~)
	if strings.Contains(arg, "/") || strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "~/") || strings.HasPrefix(arg, "~") {
		return true
	}

	// Check if it's a flag (starts with -)
	if strings.HasPrefix(arg, "-") {
		return true
	}

	// Check if it's a quoted string
	if (strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"")) ||
		(strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'")) {
		return true
	}

	// Check if it contains shell special chars
	if strings.ContainsAny(arg, "|;&<>()${}[]*?") {
		return true
	}

	// Check if it looks like a number (integer or float)
	if isNumber(arg) {
		return true
	}

	// Check if it's a variable reference
	if strings.HasPrefix(arg, "$") {
		return true
	}

	// Check if it looks like an environment variable assignment (VAR=value)
	if strings.Contains(arg, "=") && !strings.Contains(arg, "http") {
		return true
	}

	// Check if it's a known shell keyword
	if isShellKeyword(arg) {
		return true
	}

	// Check if it contains @ (email, host@user, etc.)
	if strings.Contains(arg, "@") {
		return true
	}

	// Check if it contains uppercase letters (likely a proper noun, URL, etc.)
	if containsUpper(arg) {
		return true
	}

	// Check if it contains digits (file names, paths, etc.)
	if containsDigit(arg) {
		return true
	}

	// Check if it's a single word without spaces
	// This allows for file names, subcommands, etc.
	// But we need to filter out plain English text
	if !containsSpace(arg) && len(arg) >= 2 {
		// For single words, check if it looks like English text
		// by checking for common English words
		if isEnglishText(arg) {
			return false
		}
		return true
	}

	return false
}

func isNumber(s string) bool {
	// Simple number check
	if s == "" {
		return false
	}

	// Remove leading minus for negative numbers
	if strings.HasPrefix(s, "-") {
		s = s[1:]
		if s == "" {
			return false
		}
	}

	// Check if all characters are digits, dots (for floats), or 'e'/'E' (for scientific notation)
	hasDigit := false
	seenDot := false
	seenE := false

	for i, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
		} else if r == '.' && !seenDot && !seenE {
			seenDot = true
		} else if (r == 'e' || r == 'E') && !seenE && hasDigit {
			seenE = true
			// Check next char is digit or minus
			if i+1 < len(s) {
				next := s[i+1]
				if next == '-' || next == '+' {
					// skip
				} else if next < '0' || next > '9' {
					return false
				}
			}
		} else {
			return false
		}
	}

	return hasDigit
}

func isShellKeyword(word string) bool {
	keywords := map[string]bool{
		"if": true, "then": true, "else": true, "elif": true, "fi": true,
		"case": true, "esac": true, "for": true, "while": true, "until": true,
		"do": true, "done": true, "in": true, "function": true,
		"select": true,
		"and":    true, "or": true, "not": true, "xor": true,
		"break": true, "continue": true, "return": true,
		"exit": true, "shift": true, "unset": true,
	}
	return keywords[word]
}

// hasShellSyntax checks if the command contains shell-specific syntax.
func hasShellSyntax(cmd string) bool {
	cmdLower := strings.ToLower(cmd)

	// Check for variable assignments (but not at the start)
	trimmedCmd := strings.TrimSpace(cmd)
	if strings.Contains(cmd, "=") && !strings.HasPrefix(trimmedCmd, "=") {
		return true
	}

	// Check for quoted strings (excluding backticks - they are checked via containsValidCommandSubstitution)
	if strings.Contains(cmd, "'") || strings.Contains(cmd, "\"") {
		return true
	}

	// Check for wildcards
	if strings.Contains(cmd, "*") || strings.Contains(cmd, "?") {
		return true
	}

	// Check for paths (contains / but not just /)
	if strings.Contains(cmd, "/") && len(cmd) > 1 {
		return true
	}

	// Check for command substitution and validate content
	if containsValidCommandSubstitution(cmd) {
		return true
	}

	// Check for shell operators - need to ensure they are standalone (not part of a word)
	operators := []string{
		"||", "&&", "|", ";", "&",
		"<", ">", ">>", "<>", "<<",
		// Note: ` is removed - command substitution is checked separately via containsValidCommandSubstitution
		// $() is also checked separately via containsValidCommandSubstitution
		"(", ")", "[[", "]]", "$((", "))",
		"-a", "-o", "-z", "-n", "-f", "-d", "-e", "-s", "!",
		"fi", "then", "else", "esac", "do", "done", "in",
	}

	for _, op := range operators {
		if containsOperator(cmdLower, op) {
			return true
		}
	}

	// Check for environment variable references
	if strings.Contains(cmd, "$") {
		return true
	}

	return false
}

// containsValidCommandSubstitution checks if command substitution contains a valid command.
func containsValidCommandSubstitution(cmd string) bool {
	// Check for backticks
	if strings.Contains(cmd, "`") {
		return isValidBacktickSubstitution(cmd)
	}
	// Check for $() syntax
	if strings.Contains(cmd, "$(") {
		return isValidParenSubstitution(cmd)
	}
	return false
}

// isValidBacktickSubstitution checks if backtick substitution contains a valid command.
func isValidBacktickSubstitution(cmd string) bool {
	// Find all backtick-enclosed sections
	start := 0
	for {
		open := strings.Index(cmd[start:], "`")
		if open == -1 {
			break
		}
		open += start
		closeIdx := strings.Index(cmd[open+1:], "`")
		if closeIdx == -1 {
			return false // Unmatched backtick
		}
		closeIdx += open + 1

		// Extract content between backticks
		content := cmd[open+1 : closeIdx]

		// Check if content looks like a valid command
		if !isValidCommandInSubstitution(content) {
			return false
		}

		start = closeIdx + 1
	}
	return true
}

// isValidParenSubstitution checks if $(...) substitution contains a valid command.
func isValidParenSubstitution(cmd string) bool {
	// Find all $() sections
	start := 0
	for {
		open := strings.Index(cmd[start:], "$(")
		if open == -1 {
			break
		}
		open += start
		// Find matching closing parenthesis
		closeIdx := findMatchingParen(cmd, open+2)
		if closeIdx == -1 {
			return false // Unmatched $
		}

		// Extract content between $(
		content := cmd[open+2 : closeIdx]

		// Check if content looks like a valid command
		if !isValidCommandInSubstitution(content) {
			return false
		}

		start = closeIdx + 1
	}
	return true
}

// findMatchingParen finds the matching closing parenthesis for $( at pos.
func findMatchingParen(s string, start int) int {
	depth := 1
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '"', '\'':
			// Skip quoted content
			quote := s[i]
			for i+1 < len(s) && s[i+1] != quote {
				i++
			}
			i++
		}
	}
	return -1
}

// isValidCommandInSubstitution checks if content inside command substitution looks like a valid command.
func isValidCommandInSubstitution(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}

	// Extract the command name (first word)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	// Check if it's a shell builtin
	if isShellBuiltin(cmdName) {
		return true
	}

	// Check if it's a valid command in PATH
	// For validation, we assume common commands exist
	commonCommands := map[string]bool{
		"date": true, "echo": true, "ls": true, "pwd": true,
		"whoami": true, "cat": true, "grep": true, "find": true,
		"git": true, "curl": true, "wget": true, "make": true,
		"go": true, "node": true, "npm": true, "python3": true,
		"sed": true, "awk": true, "cut": true, "sort": true,
		"tr": true, "wc": true, "head": true, "tail": true,
	}

	// Check常见 commands
	if commonCommands[cmdName] {
		return true
	}

	// Check if it looks like a valid command (has valid chars)
	if isValidCommandName(cmdName) {
		return true
	}

	return false
}

// isValidCommandName checks if a string looks like a valid command name.
func isValidCommandName(name string) bool {
	if name == "" {
		return false
	}
	// Command names can contain letters, digits, hyphens, underscores, dots
	// But must start with a letter or underscore
	first := name[0]
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}
	// Rest can be letters, digits, hyphens, underscores, dots
	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// containsOperator checks if the operator appears as a standalone shell keyword.
// (not as part of a larger word.)
func containsOperator(s string, op string) bool {
	pos := strings.Index(s, op)
	if pos == -1 {
		return false
	}

	// Check if operator is at the start of the string or preceded by whitespace/special char
	beforeOK := pos == 0 || isSeparator(s[pos-1])

	// Check if operator is at the end of the string or followed by whitespace/special char
	afterPos := pos + len(op)
	afterOK := afterPos >= len(s) || isSeparator(s[afterPos])

	// For shell keywords, also check that they're followed by valid shell syntax
	// (not just any separator - they should be followed by commands or other shell syntax)
	if afterOK {
		switch op {
		case "do", "done", "in", "then", "else", "fi", "esac":
			return checkShellKeywordContext(s, afterPos, op)
		case "if":
			return checkIfKeywordContext(s, afterPos)
		}
	}

	return beforeOK && afterOK
}

// checkShellKeywordContext checks if shell keywords have valid context.
func checkShellKeywordContext(s string, afterPos int, op string) bool {
	// These keywords should be followed by more shell syntax, not just end of string
	if afterPos >= len(s) {
		// Keyword at end of string - check if it's valid
		// "do" at end is invalid, "done" at end could be valid in a loop
		return op == "done"
	}

	// After the keyword, check if there's more shell syntax
	// Skip separators and check what comes next
	next := afterPos
	for next < len(s) && isSeparator(s[next]) {
		next++
	}

	// Must have more content that looks like shell syntax
	if next >= len(s) {
		return false
	}

	// Valid shell syntax after these keywords
	if s[next] == ';' || s[next] == '|' || s[next] == '&' ||
		s[next] == '(' || s[next] == '[' || s[next] == '$' ||
		s[next] == '`' || s[next] == '{' {
		return true
	}

	// Check for more shell keywords
	remaining := s[next:]
	if strings.HasPrefix(remaining, "do") || strings.HasPrefix(remaining, "fi") ||
		strings.HasPrefix(remaining, "then") || strings.HasPrefix(remaining, "else") ||
		strings.HasPrefix(remaining, "for") || strings.HasPrefix(remaining, "while") ||
		strings.HasPrefix(remaining, "until") || strings.HasPrefix(remaining, "case") {
		return true
	}

	// For "in", check if followed by variable or command
	if op == "in" {
		if strings.HasPrefix(remaining, "$") || strings.HasPrefix(remaining, "`") ||
			(len(remaining) > 0 && isAlpha(remaining[0])) {
			return true
		}
	}

	return false
}

// checkIfKeywordContext checks if "if" keyword has valid context.
func checkIfKeywordContext(s string, afterPos int) bool {
	// "if" should be followed by a test command
	if afterPos >= len(s) {
		return false
	}

	next := afterPos
	for next < len(s) && isSeparator(s[next]) {
		next++
	}

	if next >= len(s) {
		return false
	}

	// "if" should be followed by test command or [
	if s[next] == '[' || s[next] == '$' || s[next] == '`' {
		return true
	}

	return false
}

// isAlpha checks if a character is a letter.
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isSeparator checks if a character is a word separator.
func isSeparator(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
		c == ';' || c == '|' || c == '&' || c == '<' || c == '>' ||
		c == '(' || c == ')' || c == '{' || c == '}' ||
		c == '`' || c == '$' || c == '"' || c == '\'' ||
		c == ',' || c == '='
}

func isShellBuiltin(cmd string) bool {
	builtins := map[string]bool{
		"cd": true, "pwd": true, "echo": true, "export": true,
		"unset": true, "alias": true, "unalias": true, "eval": true,
		"source": true, ".": true, "readonly": true, "declare": true,
		"local": true, "typeset": true, "shift": true,
		"getopts": true, "command": true, "builtin": true, "enable": true,
		"help": true, "type": true, "hash": true, "history": true,
		"ulimit": true, "umask": true, "bind": true, "disown": true, "jobs": true,
		"bg": true, "fg": true, "wait": true, "true": true, "false": true,
		"exit": true, "return": true, "break": true, "continue": true,
		"read": true, "printf": true, "set": true, "shopt": true,
		"pushd": true, "popd": true, "dirs": true, "complete": true,
		"let": true, "array": true, "times": true, "trap": true,
		"logout": true, "suspend": true,
	}

	cmd = strings.TrimPrefix(cmd, "_")
	return builtins[cmd]
}

// isEnglishText checks if a word looks like English text
// Common English words that would indicate plain text vs a command argument.
func isEnglishText(word string) bool {
	// Convert to lowercase for comparison
	wordLower := strings.ToLower(word)

	// Common English words that are unlikely to be command arguments
	// NOTE: This map must not contain duplicate keys
	englishWords := map[string]bool{
		// Articles
		"a": true, "an": true, "the": true,
		// Pronouns
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
		"my": true, "your": true, "his": true, "its": true, "our": true, "their": true,
		"me": true, "him": true, "us": true, "them": true,
		"this": true, "that": true, "these": true, "those": true,
		"what": true, "which": true, "who": true, "whom": true,
		"whose": true, "where": true, "when": true, "why": true, "how": true,
		"if": true, "because": true, "as": true, "until": true, "while": true,
		"but": true, "or": true, "nor": true, "yet": true, "so": true,
		// Be verbs
		"am": true, "is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
		// Other common words
		"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
		"by": true, "from": true, "at": true, "into": true, "through": true,
		"during": true, "before": true, "after": true, "above": true, "below": true,
		"between": true, "under": true, "again": true, "further": true, "then": true,
		"once": true, "here": true, "there": true,
		"all": true, "each": true, "few": true,
		"more": true, "most": true, "other": true, "some": true, "such": true,
		"no": true, "not": true, "only": true, "own": true, "same": true,
		"than": true, "too": true, "very": true,
		"can": true, "will": true, "should": true, "would": true, "could": true,
		"may": true, "might": true, "must": true, "shall": true,
		"do": true, "does": true, "did": true, "doing": true, "done": true,
		"hello": true, "good": true, "morning": true, "afternoon": true, "evening": true,
		"night": true, "thanks": true, "thank": true, "please": true,
		"yes": true, "ok": true, "okay": true, "sure": true,
		"help": true, "helps": true, "helping": true, "helped": true,
		"make": true, "makes": true, "making": true, "made": true,
		"see": true, "sees": true, "seeing": true, "saw": true, "seen": true,
		"want": true, "wants": true, "wanting": true, "wanted": true,
		"keep": true, "keeps": true, "keeping": true, "kept": true,
		"give": true, "gives": true, "giving": true, "gave": true, "given": true,
		"use": true, "uses": true, "using": true, "used": true,
		"find": true, "finds": true, "finding": true, "found": true,
		"tell": true, "tells": true, "telling": true, "told": true,
		"ask": true, "asks": true, "asking": true, "asked": true,
		"work": true, "works": true, "working": true, "worked": true,
		"seem": true, "seems": true, "seeming": true, "seemed": true,
		"look": true, "looks": true, "looking": true, "looked": true,
		"come": true, "comes": true, "coming": true, "came": true,
		"think": true, "thinks": true, "thinking": true, "thought": true,
		"feel": true, "feels": true, "feeling": true, "felt": true,
		"get": true, "gets": true, "getting": true, "got": true, "gotten": true,
		"know": true, "knows": true, "knowing": true, "knew": true, "known": true,
		"take": true, "takes": true, "taking": true, "took": true, "taken": true,
		"go": true, "goes": true, "going": true, "went": true, "gone": true,
		"like": true, "likes": true, "liking": true, "liked": true,
		"love": true, "loves": true, "loving": true, "loved": true,
		"hate": true, "hates": true, "hating": true, "hated": true,
		"need": true, "needs": true, "needing": true, "needed": true,
		"say": true, "says": true, "saying": true, "said": true,
		"hear": true, "hears": true, "hearing": true, "heard": true,
		"listen": true, "listens": true, "listening": true, "listened": true,
		"watch": true, "watches": true, "watching": true, "watched": true,
		"study": true, "studies": true, "studying": true, "studied": true,
		"learn": true, "learns": true, "learning": true, "learnt": true, "learned": true,
		"remember": true, "remembers": true, "remembering": true, "remembered": true,
		"forget": true, "forgets": true, "forgetting": true, "forgot": true, "forgotten": true,
		"understand": true, "understands": true, "understanding": true, "understood": true,
		"believe": true, "believes": true, "believing": true, "believed": true,
		"hope": true, "hopes": true, "hoping": true, "hoped": true,
		"wish": true, "wishes": true, "wishing": true, "wished": true,
		"live": true, "lives": true, "living": true, "lived": true,
		"bye": true, "hi": true, "hey": true,
		"fine": true,
	}
	return englishWords[wordLower]
}

func containsUpper(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}
