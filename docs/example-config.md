# Example Configuration

This is a complete example configuration file demonstrating all available options:

```yaml
version: "1.0"

# Language for agent responses (default: english)
language: english

# Enable verbose tool call output (default: true)
verbose: true

# Default profile name
profile: default

# Include plugin configurations
include:
  - plugins/jira.yml
  - plugins/github.yml
  - plugins/confluence.yml
  - plugins/gitlab.yml
  - plugins/rag.yml
  - plugins/guard.yml
  - plugins/ssh.yml

profiles:
  default:
    # Primary LLM for chat responses
    llm:
      base_url: https://api.openai.com/v1
      api_key: YOUR_API_KEY
      model: gpt-4o
      context: 128000
      temperature: 0.9
      top_p: 0.95
      reasoning_effort: high

# RAG configuration
rag:
  enable: false
  llm:
    base_url: https://api.openai.com/v1
    api_key: YOUR_EMBEDDING_API_KEY
    model: text-embedding-3-small
    vector_size: 1536
  chunk:
    size: 2048
    overlap: 256
  search:
    limit: 10
    threshold: 0.40
    multi_query: 2
    dedup_threshold: 0.95
  qdrant:
    connstr: http://localhost:6333
    collection: documents

# Conversation history compression
compact:
  threshold: 0
  save_tail: 6

# Confluence integration (optional)
confluence:
  enable: false
  base_url: https://your-domain.atlassian.net
  api_key: YOUR_API_KEY

# JIRA integration (optional)
jira:
  enable: false
  base_url: https://jira.example.com
  api_key: YOUR_API_KEY

# GitLab integration (optional)
gitlab:
  enable: false
  base_url: https://gitlab.example.com
  api_key: YOUR_API_KEY

# GitHub integration (optional)
github:
  enable: false
  base_url: https://api.github.com
  access_key: YOUR_API_KEY

# Guard agent for exec commands (optional)
guard:
  enable: false
  ask: false

# SSH execution (optional)
ssh:
  enable: false
  user: $env:USER
  key: ~/.ssh/id_rsa
  password: ""
  use_ssh_agent: false
  remote_dir: ""
  timeout: 120
```

### Minimal Configuration

A minimal working configuration with only essential settings:

```yaml
version: "1.0"

language: english
verbose: true
profile: default

profiles:
  default:
    llm:
      base_url: https://api.openai.com/v1
      api_key: $env:OPENAI_API_KEY
      model: gpt-4o
      context: 128000
      temperature: 0.9
      top_p: 0.95
      reasoning_effort: high
```

### Profile Switching

To use different API keys for development and production:

**configs/dev-config.yml:**
```yaml
version: "1.0"
profile: development
include:
  - plugins/rag.yml

profiles:
  development:
    llm:
      base_url: https://api.openai.com/v1
      api_key: $env:DEV_API_KEY
      model: gpt-4o
      context: 128000
```

**configs/prod-config.yml:**
```yaml
version: "1.0"
profile: production
include:
  - plugins/rag.yml

profiles:
  production:
    llm:
      base_url: https://api.openai.com/v1
      api_key: $env:PROD_API_KEY
      model: gpt-4o
      context: 128000
```

Run with:
```bash
# Development profile
go-magnetar -p development

# Production profile
go-magnetar -p production
```

### Using Environment Variables and Files

```yaml
profiles:
  default:
    llm:
      # Use environment variable for API key
      api_key: $env:OPENAI_API_KEY

      # Or load from file
      api_key: $file:secrets/api_key.txt

      # Or use file path from environment variable
      api_key: $file:$env:API_KEY_FILE
```

### Enabling Plugins

```yaml
rag:
  enable: true
  chunk:
    size: 4096
    overlap: 512
  search:
    limit: 20
    threshold: 0.35
    multi_query: 3
    dedup_threshold: 0.92

github:
  enable: true
  base_url: https://api.github.com
  access_key: $env:GITHUB_TOKEN

confluence:
  enable: true
  base_url: https://your-domain.atlassian.net
  api_key: $env:CONFLUENCE_TOKEN
```
