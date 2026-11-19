---
name: make-issue
description: Create GitHub issues from user requests
disable: false
---

**YOU MUST FOLLOW THESE INSTRUCTIONS. NO EXCEPTIONS.**

STEP 1: **ANALYZE CODEBASE**
- Search the codebase to find relevant files
- Find the exact file path and function name
- Understand what the code currently does
- Identify what should be changed

STEP 2: **DETERMINE ISSUE TYPE**
- Bug: user reports problem/error
- Task: user requests new feature

STEP 3: **CREATE ISSUE**
- First line: `Bug: <title>` or `Task: <title>`
- Include file paths and function names in the issue
- Explain what code currently does vs what should do

**FORMAT:**
- Use ### for section headers
- Use code blocks for logs
- Write in English

**EXAMPLE - Bug (with code analysis):**

Bug: --file flag doesn't support stdin input

### Description
The --file flag does not support reading from stdin. File reading logic is in `cmd/root.go` function `readInputFile()`.

### Steps to Reproduce
1. Run command with --file flag
2. Try to pass input via stdin
3. Observe error

### Expected Behavior
Should accept input from stdin when --file flag is used.

### Actual Behavior
Fails with error about missing file.

### Environment
- OS: Linux
- go-magnetar version: v1.0.0
- Go version: go1.27.0

### Logs
error: file not found

### Additional Context
Found in `cmd/root.go:readInputFile()` - needs to check for stdin when file path is "-"

### Example - Task (with code analysis):

Task: support -s|--skill flag for non-interactive mode

### Problem
Currently there's no way to apply a skill when using non-interactive mode. The `main.go` and `agent/repl.go` need to be modified.

### Solution
Add -s|--skill flag that works with -f/--file. Modify `cmd/root.go` to accept skill parameter.

### Alternatives
None considered.

### Additional Context
Files to modify: `cmd/root.go`, `agent/repl.go`

**IMPORTANT RULES:**
- DO NOT skip codebase analysis
- DO NOT create issue without file paths and function names
- DO NOT guess - always search and verify
- **PROHIBITED: Any modifications including code edits, documentation changes, or file modifications**
- **ALLOWED ONLY: Code analysis**
