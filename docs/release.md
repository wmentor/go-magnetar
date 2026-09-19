# Release Process

This document describes the release process for go-magnetar using GitHub Flow and GoReleaser.

## Overview

Releases are created by pushing a tagged commit to the repository. GitHub Actions automatically builds and publishes the release using GoReleaser.

## Release Steps

### 1. Create a New Release Tag

Bump the version number in `cmd/go-magnetar/main.go` and create a signed tag:

```bash
# Update version in cmd/go-magnetar/main.go
make lint test

# Create and sign the release tag (e.g., v1.0.0)
git tag -s -a v1.0.0 -m "Release v1.0.0"
```

The `-m` flag specifies the tag message, which becomes the release notes text on GitHub. For best results, include a summary of changes in this message.

The tag format is `vX.Y.Z` where:
- `X` = major version (backward-incompatible changes)
- `Y` = minor version (backward-compatible features)
- `Z` = patch version (backward-compatible bug fixes)

### 2. Push the Tag

Push the tag to the remote repository:

```bash
git push origin v1.0.0
```

### 3. GitHub Release Workflow

When the tag is pushed, GitHub Actions triggers the release workflow which:

1. Validates the tag format (`vX.Y.Z`)
2. Checks out the code
3. Runs tests with `make test`
4. Runs `goreleaser release` to build and publish

### 4. GoReleaser Build Process

GoReleaser performs the following steps:

- **Builds binaries** for all supported platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- **Creates source tarball**
- **Generates checksums** for all binaries
- **Creates GitHub Release** with:
  - Release notes (from git tag `-m` message)
  - Binary attachments
  - Checksum file (`checksums.txt`)
- **Publishes to GitHub**

## Versioning Strategy

go-magnetar follows [Semantic Versioning](https://semver.org/):

- **Major version** (`X`): Breaking changes
- **Minor version** (`Y`): New features (backward-compatible)
- **Patch version** (`Z`): Bug fixes (backward-compatible)

## Examples

### Release v1.0.0 (first release)

```bash
git tag -s -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### Release v1.1.0 (new feature)

```bash
git tag -s -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

### Release v1.0.1 (bug fix)

```bash
git tag -s -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

## Release Artifacts

After a successful release, the following artifacts are published:

| File | Description |
|---|---|
| `go-magnetar_`*_linux_amd64.tar.gz* | Linux AMD64 binary |
| `go-magnetar_`*_linux_arm64.tar.gz* | Linux ARM64 binary |
| `go-magnetar_`*_darwin_amd64.tar.gz* | macOS AMD64 binary |
| `go-magnetar_`*_darwin_arm64.tar.gz* | macOS ARM64 binary |
| `go-magnetar_`*_windows_amd64.zip* | Windows AMD64 binary |
| `source.tar.gz` | Source code tarball |
| `checksums.txt` | SHA256 checksums for all binaries |

## Troubleshooting

### Tag Not Picked Up

If the release workflow doesn't trigger:

1. Verify the tag format: `git tag --list | grep '^v[0-9]\+\.[0-9]\+\.[0-9]\+$'`
2. Check the tag exists remotely: `git ls-remote --tags origin`
3. Ensure the tag is an **annotated** tag: `git tag -v v1.0.0`

### Release Failed

If the GitHub release fails:

1. Check the Actions tab for workflow logs
2. Common issues:
   - Tag format doesn't match `vX.Y.Z`
   - Release already exists
   - GitHub token permissions

3. Fix the issue and retry:
```bash
# Delete the failed release on GitHub
# Then push the tag again
git push origin v1.0.0
```
