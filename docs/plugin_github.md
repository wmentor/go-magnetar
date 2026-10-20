# GitHub Plugin

The GitHub plugin enables integration with GitHub for repository information, file content, issue management, pull requests, milestones, releases, and security advisories.

## Enabling the Plugin

Set `github.enable: true` in your configuration file (see [configuration.md](./configuration.md) for complete configuration options).

## Configuration

```yaml
github:
  enable: false  # enable/disable GitHub integration (default: false)
  base_url: https://api.github.com  # GitHub API base URL
  api_key: ""  # GitHub Personal Access Token (PAT)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `github.enable` | boolean | Enable/disable GitHub plugin |
| `github.base_url` | string | GitHub API base URL (e.g., `https://api.github.com` or GitHub Enterprise) |
| `github.api_key` | string | GitHub Personal Access Token (PAT) for authenticated requests |

## Tools

When the GitHub plugin is enabled, the following tools become available:

| Tool | Signature | Description |
|------|-----------|-------------|
| `github_repo` | `(repo: string) -> string` | Fetches GitHub repository information and returns its details in Markdown format |
| `github_file` | `(repo: string, branch: string, file: string) -> string` | Fetches a file from GitHub repository and returns its content |
| `github_tree` | `(repo: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |
| `github_issue` | `(repo: string, issue: string) -> string` | Fetches a GitHub issue and its comments, returns issue details in Markdown format |
| `github_milestone` | `(repo: string, milestone: string) -> string` | Fetches a GitHub milestone and returns its details in Markdown format |
| `github_pull_request` | `(repo: string, pr: string) -> string` | Fetches a GitHub pull request with all details including files changed, comments, and reviews |
| `github_release_info` | `(repo: string, tag: string) -> string` | Fetches a GitHub release by tag name and returns its details in Markdown format, including changelog and assets |
| `github_advisory` | `(advisory: string) -> string` | Fetches a GitHub security advisory by GHSA ID |
| `github_repo_advisories` | `(repo: string) -> string` | Fetches all security advisories for a repository |

### Tool Details

#### `github_repo`

Fetches repository information including:

- Repository name and full path
- Default branch
- License information
- Primary programming language
- Statistics (stars, forks, open issues)
- Topics
- Description
- README content
- URLs (Web, SSH, HTTP)

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |

#### `github_file`

Fetches a specific file from a GitHub repository with support for:

- File content retrieval
- Base64 decoding (automatically handled)
- Specific branch selection

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `branch` | string | Branch name (default: `master`) |
| `file` | string | Path to the file in the repository (e.g., `src/main.go`) |

#### `github_tree`

Lists repository contents at the root or specified path:

- Directory structure navigation
- File and directory type identification
- Branch-specific view
- File sizes

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `branch` | string | Branch name (default: `master`) |
| `path` | string | Path to directory (default: empty string for root) |

#### `github_issue`

Fetches a GitHub issue with full details including:

- Issue metadata (number, title, state, locked status)
- Author and assignees
- Labels
- Description
- Comments (if any)
- Creation and update timestamps
- URLs (Web, API)

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `issue` | string | Issue number (e.g., `7`) or GHSA ID for security advisories |

#### `github_milestone`

Fetches a GitHub milestone with associated issues:

- Milestone metadata (number, title, state)
- Creator and creation/update timestamps
- Description
- Due date (if set)
- Issue counts (open/closed)
- List of issues with details

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `milestone` | string | Milestone number (e.g., `278`) |

#### `github_pull_request`

Fetches a GitHub pull request with comprehensive details including:

- PR metadata (number, title, state, locked status)
- Author and merge status
- Base and head branches
- Change statistics (files, additions, deletions)
- Labels
- Mergeability status
- Description
- Files changed with diffs
- Review comments
- Reviews (approved, changes requested, etc.)
- URLs (Web, API)

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `pr` | string | Pull request number (e.g., `7`) |

#### `github_release_info`

Fetches a GitHub release with:

- Release metadata (name, tag, published date, draft status, prerelease flag)
- Author (typically a bot for automated releases)
- Changelog/description
- Assets (download links, sizes, download counts, upload dates)
- Tarball and zipball URLs

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |
| `tag` | string | Release tag name (e.g., `v1.6.0`) |

#### `github_advisory`

Fetches a GitHub security advisory by GHSA ID:

- Advisory metadata (GHSA ID, CVE ID, URL)
- Summary and detailed description
- Severity and CVSS scores
- Affected packages and version ranges
- Patches and workarounds
- References and credits

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `advisory` | string | GitHub Security Advisory ID (e.g., `GHSA-xxxx-xxxx-xxxx`) |

#### `github_repo_advisories`

Fetches all security advisories for a repository:

- List of advisories with GHSA IDs
- Titles and severity levels
- Published dates
- Summary information

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `repo` | string | GitHub repository URL (e.g., `https://github.com/owner/repo` or `owner/repo`) |

## Authentication

The plugin uses GitHub Personal Access Tokens (PAT) for authentication. To generate a PAT:

1. Navigate to your GitHub account settings
2. Go to **Developer settings** > **Personal access tokens**
3. Click **Generate new token** > **Generate new token (classic)**
4. Set token expiration and select appropriate scopes:
   - `repo` scope for full repository access (read and write)
   - `public_repo` scope for public repositories (read-only)
   - `repo:status` scope for access to commit status
   - `read:org` scope for organization read access
5. Copy the generated token

Store the token securely in your configuration:

```yaml
github:
  enable: true
  base_url: https://api.github.com
  api_key: $env:GITHUB_API_KEY  # or $file:secrets/github_token.txt
```

### Using Environment Variables

For security, it's recommended to use environment variables:

```bash
export GITHUB_API_KEY="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

```yaml
github:
  enable: true
  base_url: https://api.github.com
  api_key: $env:GITHUB_API_KEY
```

### Using File-based Secrets

You can also load the token from a file:

```yaml
github:
  enable: true
  base_url: https://api.github.com
  api_key: $file:secrets/github_token.txt
```

## Usage Examples

### Fetch Repository Information

```bash
> What information do you have about wmentor/go-magnetar?
[Agent calls github_repo with repo="wmentor/go-magnetar"]
```

### Fetch File Content

```bash
> Show me the contents of AGENTS.md in wmentor/go-magnetar on the main branch
[Agent calls github_file with repo="wmentor/go-magnetar", branch="main", file="AGENTS.md"]
```

### List Repository Contents

```bash
> What files are in the internal/tools directory of wmentor/go-magnetar on the main branch?
[Agent calls github_tree with repo="wmentor/go-magnetar", branch="main", path="internal/tools"]
```

### Fetch Issue

```bash
> Show me issue #7 in wmentor/go-magnetar
[Agent calls github_issue with repo="wmentor/go-magnetar", issue="7"]
```

### Fetch Milestone

```bash
> Show me milestone #278 in golang/go
[Agent calls github_milestone with repo="golang/go", milestone="278"]
```

### Fetch Pull Request

```bash
> Show me PR #123 in wmentor/go-magnetar
[Agent calls github_pull_request with repo="wmentor/go-magnetar", pr="123"]
```

### Fetch Release

```bash
> Show me release v1.6.0 in wmentor/go-magnetar
[Agent calls github_release_info with repo="wmentor/go-magnetar", tag="v1.6.0"]
```

### Fetch Security Advisory

```bash
> Show me advisory GHSA-xxxx-xxxx-xxxx
[Agent calls github_advisory with advisory="GHSA-xxxx-xxxx-xxxx"]
```

### Fetch Repository Advisories

```bash
> What security advisories does wmentor/go-magnetar have?
[Agent calls github_repo_advisories with repo="wmentor/go-magnetar"]
```

## Security

- API tokens are stored in configuration and can be loaded from environment variables or files
- The plugin respects the global `readonly` mode if implemented
- All API requests use HTTPS (default for GitHub API)
- Token is included in the `Authorization` header as a Bearer token
- API version is specified via `X-GitHub-Api-Version` header

## Custom GitHub Instances (GitHub Enterprise)

The plugin supports GitHub Enterprise and custom GitHub instances. Simply configure the `base_url` to point to your instance:

```yaml
github:
  enable: true
  base_url: https://github.company.com/api/v3
  api_key: $env:GITHUB_API_KEY
```

## API Rate Limits

GitHub API has rate limits:

- **Unauthenticated requests**: 60 requests per hour
- **Authenticated requests**: 5,000 requests per hour

The plugin implements a default timeout of 1 minute per request. If you encounter rate limiting issues, consider:

- Reducing the frequency of API calls
- Using caching where appropriate
- Upgrading to GitHub Enterprise for higher rate limits
- Using application tokens instead of user tokens for higher limits

## Troubleshooting

### Common Errors

**"401 Unauthorized"**
- Ensure the API token is correct and has the necessary scopes
- Check that the token has not expired
- Verify the token has read access to the repository

**"404 Not Found"**
- Verify the repository URL is correct
- Check that the branch exists
- Ensure the file, issue, or release exists

**"403 Forbidden"**
- Verify the token has the necessary permissions (e.g., `repo` scope)
- Check that the repository is accessible with the provided credentials
- For GitHub Enterprise, ensure the correct `base_url` is configured

**"403 Rate limit exceeded"**
- Reduce the frequency of API calls
- Wait for rate limit to reset (typically 1 hour)
- Use environment variables or file-based secrets for authentication

### Debugging

Enable verbose logging in your configuration:

```yaml
log:
  level: debug
```

### GitHub Enterprise TLS Issues

If you encounter TLS certificate errors with GitHub Enterprise:

```yaml
github:
  enable: true
  base_url: https://github.company.com/api/v3
  api_key: $env:GITHUB_API_KEY
```

Note: The plugin uses `InsecureSkipVerify: true` for TLS connections (same as other plugins) to handle self-signed certificates. For production, configure proper CA certificates.
