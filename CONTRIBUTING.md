# Contributing to go-magnetar

Thank you for your interest in contributing to go-magnetar! We welcome contributions from the community and appreciate your time and effort in helping improve this project.

## Getting Started

If you're here because you found a bug or want to add a new feature, you're in the right place! If you're unsure where to begin, check out our [open issues](https://github.com/wmentor/go-magnetar/issues) in the issue tracker — there's something for every skill level.

### How to Contribute

1. **Choose an issue** — Browse the issue tracker and pick something that interests you. If you're new to the project, look for issues labeled `good first issue` or `help wanted`.

2. **Comment on the issue** — Before starting work, please comment on the issue to let us know you'd like to work on it. This helps avoid duplicate efforts and allows us to provide guidance.

3. **Discuss your approach** — For feature requests or behavioral changes, describe your plan in the issue. This ensures alignment with project goals before you invest time in implementation.

4. **Fork the repository** — Create your own fork and work on your changes in a feature branch.

5. **Implement your changes** — Follow the guidelines below to ensure your contribution meets our standards.

## Development Setup

Before you begin, make sure you have the development tools installed:

```bash
make setup/tools
```

This installs all required development tools as documented in [docs/dev_tools.md](./docs/dev_tools.md).

## Coding Standards

### Code Quality

Before submitting your contribution, ensure it passes all quality checks:

```bash
make run/fix      # Auto-fix formatting issues
make run/tidy     # Synchronize go.mod and go.sum
make fmt/sources  # Format Go source files
make fmt/imports  # Sort and group imports
make test         # Run all tests
make build        # Build the project
```

All tests must pass before your pull request can be merged.

### Test Requirements

- **Every behavioral change must have corresponding tests** — If you're adding new functionality or modifying existing behavior, please include appropriate test cases.
- Use `t.Parallel()` in all tests to enable concurrent execution.
- Use `t.Helper()` in test helper functions for better error context.

### Documentation

- All code comments and technical documentation must be written in **English**.
- Update relevant documentation in the `docs/` directory if your changes affect functionality.

## Pull Request Guidelines

1. **Create a descriptive title** — Clearly summarize your changes (e.g., "Add support for PDF embedding models" instead of "Fix stuff").

2. **Reference related issues** — Use "Fixes #123" or "Closes #456" in your PR description to automatically close issues.

3. **Provide context** — Explain what you changed, why you changed it, and how you tested it.

4. **Be responsive to feedback** — We may request changes or clarifications — please engage constructively in the review process.

## Code Style

- Use `gopls` for code navigation and references instead of grep.
- Use `errors.Wrap` or `fmt.Errorf` with context when returning errors.
- Define custom errors as global variables with `Err` prefix in `internal/<package>/errors.go`.
- When building strings with `strings.Builder`, use `fmt.Fprintf(&sb, ...)` instead of `sb.WriteString(fmt.Sprintf(...))`.

## Questions?

If you have questions about the contribution process or need help with an issue:

- Comment on the relevant issue
- Check existing discussions in the issue tracker
- Review the documentation in `docs/`

We're happy to help you make your first contribution to go-magnetar!
