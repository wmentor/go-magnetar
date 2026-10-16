# GitLab Plugin

The GitLab plugin enables integration with GitLab for repository information, file content, and merge request fetching.

## Enabling the Plugin

Set `gitlab.enable: true` in your configuration file (see [configuration.md](./configuration.md) for complete configuration options).

## Configuration

```yaml
gitlab:
  enable: false  # enable/disable GitLab integration (default: false)
  base_url: https://gitlab.com  # GitLab base URL
  api_key: ""  # GitLab API access token
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `gitlab.enable` | boolean | Enable/disable GitLab plugin |
| `gitlab.base_url` | string | GitLab base URL (e.g., `https://gitlab.com` or custom GitLab instance) |
| `gitlab.api_key` | string | GitLab API access token (required for authenticated requests) |

## Tools

When the GitLab plugin is enabled, the following tools become available:

| Tool | Signature | Description |
|------|-----------|-------------|
| `gitlab_fetch_mr` | `(project_path: string, mr_id: string) -> string` | Fetches a GitLab merge request and returns its details in Markdown format |
| `gitlab_repo` | `(project_path: string) -> string` | Fetches GitLab repository information and returns its details in Markdown format |
| `gitlab_file` | `(project_path: string, branch: string, file: string) -> string` | Fetches a file from GitLab repository and returns its content |
| `gitlab_tree` | `(project_path: string, branch: string, path: string) -> string` | Lists repository contents at root or specified path |

### Tool Details

#### `gitlab_fetch_mr`

Fetches detailed information about a GitLab merge request including:

- MR metadata (ID, IID, title, state, author)
- Source and target branches
- Labels and milestone
- Changes/diffs for all modified files
- Creation and update timestamps
- Web URL for the MR

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `project_path` | string | GitLab project path (e.g., `namespace/project` or `group/subgroup/project`) |
| `mr_id` | string | Merge request ID (numeric ID or IID) |

#### `gitlab_repo`

Fetches repository information including:

- Repository name and full path
- Default branch
- Visibility level
- Statistics (forks, stars, open issues)
- Description
- URLs (Web, SSH, HTTP)
- Last activity timestamp

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `project_path` | string | GitLab project path (e.g., `namespace/project` or `group/subgroup/project`) |

#### `gitlab_file`

Fetches a specific file from a GitLab repository with support for:

- File content retrieval
- Base64 decoding (automatically handled)
- Specific branch selection

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `project_path` | string | GitLab project path (e.g., `namespace/project` or `group/subgroup/project`) |
| `branch` | string | Branch name (default: `main`) |
| `file` | string | Path to the file in the repository (e.g., `src/main.go`) |

#### `gitlab_tree`

Lists repository contents at the root or specified path:

- Directory structure navigation
- File and directory type identification
- Branch-specific view

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `project_path` | string | GitLab project path (e.g., `namespace/project` or `group/subgroup/project`) |
| `branch` | string | Branch name (default: `main`) |
| `path` | string | Path to directory (default: empty string for root) |

## Authentication

The plugin uses GitLab API tokens for authentication. To generate an API token:

1. Navigate to your GitLab profile settings
2. Go to **Access Tokens** or **Personal Access Tokens**
3. Create a new token with appropriate scope:
   - `api` scope for full API access
   - `read_api` scope for read-only access
   - `read_repository` scope for repository content access

Store the token securely in your configuration:

```yaml
gitlab:
  enable: true
  base_url: https://gitlab.com
  api_key: $env:GITLAB_API_KEY  # or $file:secrets/gitlab_token.txt
```

### Using Environment Variables

For security, it's recommended to use environment variables:

```bash
export GITLAB_API_KEY="glpat-xxxxx"
```

```yaml
gitlab:
  enable: true
  base_url: https://gitlab.com
  api_key: $env:GITLAB_API_KEY
```

### Using File-based Secrets

You can also load the token from a file:

```yaml
gitlab:
  enable: true
  base_url: https://gitlab.com
  api_key: $file:secrets/gitlab_token.txt
```

## Usage Examples

### Fetch Repository Information

```bash
> What information do you have about namespace/project?
[Agent calls gitlab_repo with project_path="namespace/project"]
```

### List Repository Contents

```bash
> What files are in the src directory of namespace/project on the main branch?
[Agent calls gitlab_tree with project_path="namespace/project", branch="main", path="src"]
```

### Fetch File Content

```bash
> Show me the contents of src/main.go in namespace/project on the main branch
[Agent calls gitlab_file with project_path="namespace/project", branch="main", file="src/main.go"]
```

### Fetch Merge Request

```bash
> Show me MR !123 in namespace/project
[Agent calls gitlab_fetch_mr with project_path="namespace/project", mr_id="123"]
```

## Security

- API tokens are stored in configuration and can be loaded from environment variables or files
- The plugin respects the global `readonly` mode if implemented
- All API requests use HTTPS (default for GitLab API)
- Token is included in the `Authorization` header as a Bearer token

## Custom GitLab Instances

The plugin supports self-hosted GitLab instances. Simply configure the `base_url` to point to your instance:

```yaml
gitlab:
  enable: true
  base_url: https://gitlab.company.com
  api_key: $env:GITLAB_API_KEY
```

## API Rate Limits

GitLab API has rate limits. The plugin implements a default timeout of 1 minute per request. If you encounter rate limiting issues, consider:

- Reducing the frequency of API calls
- Using caching where appropriate
- Upgrading to GitLab Premium for higher rate limits

## Troubleshooting

### Common Errors

**"401 Unauthorized"**
- Ensure the API token is correct and has the necessary scopes
- Check that the token has not expired

**"404 Not Found"**
- Verify the project path is correct
- Check that the branch exists
- Ensure the file exists in the repository

**"403 Forbidden"**
- Verify the token has the necessary permissions (e.g., `read_repository`)
- Check that the repository is accessible with the provided credentials

### Debugging

Enable verbose logging in your configuration:

```yaml
log:
  level: debug
```
