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

# Plugin configuration files are automatically loaded from <config_dir>/plugins/*.yml
# No 'include' directive is required

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
  store:
    type: chromem  # or qdrant
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
  chromem:
    data_dir: ~/.go-magnetar/store  # omit for in-memory mode, set to "" for explicit in-memory
  qdrant:
    connstr: http://localhost:6333
    collection: documents

# Conversation history compression
compact:
  mode: soft  # soft or compact
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

# Proxy settings (optional, applies to all HTTP clients)
web:
  proxy:
    http: http://proxy.example.com:8080
    https: https://proxy.example.com:8080
    noproxy: localhost,127.0.0.1,.internal.example.com

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

# SMTP email sending (optional)
smtp:
  enable: false
  host: smtp.gmail.com
  port: 587
  username: your-email@gmail.com
  password: your-app-password
  encryption: starttls  # tls, starttls, or none
  from_name: "Your Name"
  from_email: your-email@gmail.com
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

# Plugin files are automatically loaded from <config_dir>/plugins/*.yml
# No 'include' directive is required

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

# Plugin files are automatically loaded from <config_dir>/plugins/*.yml
# No 'include' directive is required

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
  store:
    type: chromem  # or qdrant
  chunk:
    size: 4096
    overlap: 512
  search:
    limit: 20
    threshold: 0.35
    multi_query: 3
    dedup_threshold: 0.92
  chromem:
    data_dir: ~/.go-magnetar/store

github:
  enable: true
  base_url: https://api.github.com
  access_key: $env:GITHUB_TOKEN

confluence:
  enable: true
  base_url: https://your-domain.atlassian.net
  api_key: $env:CONFLUENCE_TOKEN

smtp:
  enable: true
  host: smtp.gmail.com
  port: 587
  username: $env:SMTP_USERNAME
  password: $env:SMTP_PASSWORD
  encryption: starttls
  from_name: "Your Name"
  from_email: $env:SMTP_USERNAME

# Web proxy settings (optional, applies to all HTTP clients)
web:
  proxy:
    http: http://proxy.example.com:8080    # HTTP proxy URL
    https: https://proxy.example.com:8080   # HTTPS proxy URL
    noproxy: localhost,127.0.0.1,.internal.example.com  # Comma-separated list of hosts to bypass proxy
```

### RAG Configuration Examples

#### Chromem with Persistent Storage

Use Chromem (default) with data persisted to disk:

```yaml
rag:
  enable: true
  store:
    type: chromem
  chromem:
    data_dir: ~/.go-magnetar/store
```

#### Chromem In-Memory Only

For temporary storage (data lost on restart), explicitly set `data_dir` to empty string:

```yaml
rag:
  enable: true
  store:
    type: chromem
  chromem:
    data_dir: ""  # in-memory mode
```

#### Qdrant External Store

For production deployments with Qdrant:

```yaml
rag:
  enable: true
  store:
    type: qdrant
  qdrant:
    connstr: http://localhost:6333
    collection: documents
```
