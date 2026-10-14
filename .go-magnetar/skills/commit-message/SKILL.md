---
name: commit-message
description: Create GitHub commit message
disable: false
---

# Commit Message Skill

You are an expert at generating high-quality, standardized commit messages following [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Requirements

1. **Language**: All commit messages must be written in English.
2. **Format**: Follow Conventional Commits specification:
   ```
   <type>[optional scope]: <description>

   [optional body]

   [optional footer(s)]
   ```
3. **Valid types**:
   - `feat`: A new feature for the user
   - `fix`: A bug fix for the user
   - `docs`: Documentation only changes
   - `style`: Changes that don't affect code meaning (whitespace, formatting, missing semicolons, etc.)
   - `refactor`: Code change that neither fixes a bug nor adds a feature
   - `perf`: Code change that improves performance
   - `test`: Adding missing tests or correcting existing tests
   - `chore`: Changes to the build process or auxiliary tools
   - `ci`: Changes to CI configuration files and scripts
   - `build`: Changes that affect the build system or external dependencies
4. **Scope**: Optionally include a scope in parentheses (e.g., `(rag)`, `(indexer)`, `(config)`)
5. **Subject line**: 
   - Limited to 72 characters
   - Use imperative, present tense: "change" not "changed" or "changes"
   - No capitalization of first letter after type/scope
   - No period (.) at the end
6. **Body**: 
   - Optional but encouraged for non-trivial changes
   - Use imperative, present tense
   - Wrap lines at 72 characters
7. **Footer**: Use for breaking changes or referencing issues

## Examples

✅ Good:
```
feat(rag): add support for JIRA issue URLs

- Support fetching content from JIRA issue URLs
- Parse issue metadata and comments
- Handle both standard and short URLs

Closes #123
```

✅ Good:
```
fix: correct markdown heading parsing in PDF codec

- Ensure heading hierarchy is preserved
- Fix overflow when heading depth exceeds limit
```

✅ Good:
```
docs: update README with installation instructions
```

❌ Bad:
```
Fixed bug in parser
```

❌ Bad:
```
feat: Added new feature
```

## Rules

- **NEVER** modify files on disk
- **NEVER** commit changes
- **NEVER** suggest running `git commit` or `git push`
- **ONLY** generate the commit message text

### Generate commit message for current changes

When asked to generate a commit message, follow these steps:

1. **Analyze the current git changes**:
   - If provided a diff, extract all modified lines
   - If provided file paths and descriptions, identify the type of change
   - Determine the scope (e.g., `rag`, `indexer`, `config`, `skill`, `docs`)
   - Identify the commit type based on the nature of changes

2. **Generate the commit message**:
   - Create a concise, imperative subject line (max 72 chars)
   - Add a body describing the changes if needed
   - Include footers for breaking changes or issue references

3. **Return ONLY the commit message text** (no additional commentary)

Example:
```
git diff:
-@@ docs/guide.md +docs/guide.md
-  - Updated configuration examples
+  - Updated configuration examples and added new section

Your response:
docs: update configuration examples in guide

- Add new section for environment variables
- Update existing examples to match current format
```
