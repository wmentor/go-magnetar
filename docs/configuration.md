# Configuration

Copy the example and fill in actual values:

```bash
cp configs/config.yaml my-config.yaml
```

```yaml
llm:
  base_url: https://api.openai.com/v1   # OpenAI-compatible endpoint
  api_key: YOUR_API_KEY
  model: gpt-4o
  context: 128000                        # token limit of the context window
  temperature: 0.5                       # LLM temperature for response generation (default: 0.5)
  top_p: 0.95                            # LLM top_p for response generation (default: 0.95)
  reasoning_effort: high                 # LLM reasoning effort: low, medium, or high (default: high)

language: english                      # language for agent responses (default: english)

rag:
  disable: false                         # disable RAG (default: false)
  llm:
    base_url: https://api.openai.com/v1
    api_key: YOUR_API_KEY
    model: text-embedding-3-small        # embedding model
    vector_size: 1536                    # vector dimensionality of the model
    disable: false                       # disable embedding model (default: false)
  chunk:
    size: 512                            # maximum chunk size in runes (default: 512)
    overlap: 64                          # overlap between adjacent chunks in runes (default: 64)
  search:
    limit: 10                            # maximum number of results per query (default: 10)
    threshold: 0.40                      # minimum cosine similarity threshold, 0–1 (default: 0.40)
    multi_query: 2                       # extra LLM-generated query variants (default: 2, 0 = off)
    dedup_threshold: 0.95                # near-duplicate suppression threshold (default: 0.95, 0 = off)
  qdrant:
    connstr: http://localhost:6333       # Qdrant address (REST port; gRPC 6334 is used automatically)
    collection: documents                # collection name (created automatically if missing)
  
verbose: true                          # enables verbose tool call output (default: true)

compact:
  threshold: 0    # token threshold for history compression; 0 = auto (80% of llm.context)
  save_tail: 6    # number of trailing messages kept unchanged

webfetch:
  base_url: https://api.openai.com/v1
  api_key: YOUR_API_KEY
  model: gpt-4o
  context: 128000
  disable: false                       # disable web page preprocessing (default: false)

confluence:
  base_url: https://your-domain.atlassian.net
  api_key: YOUR_API_KEY
  disable: false                       # disable Confluence fetching (default: false)

jira:
  base_url: https://jira.example.com
  api_key: YOUR_API_KEY
  disable: false                       # disable JIRA fetching (default: false)

gitlab:
  base_url: https://gitlab.example.com
  api_key: YOUR_API_KEY
  disable: false                       # disable GitLab fetching (default: false)

github:
  base_url: https://api.github.com
  api_key: YOUR_GITHUB_TOKEN
  disable: false                       # disable GitHub fetching (default: false)

guard:
  disable: false    # disable guard agent for exec commands (default: false)
  ask: false        # ask user for confirmation when guard blocks a command (default: false)
```

> If the `webfetch` block is specified, the listed model parameters are used to clean HTML content obtained from web pages.

> The `confluence` block enables fetching Confluence pages directly by URL (both standard and short links).

> The `jira` block enables fetching JIRA issues directly by URL.

> `vector_size` must match the dimensionality of the chosen embedding model.
> For `text-embedding-3-small` — 1536, for `text-embedding-ada-002` — 1536, for `text-embedding-3-large` — 3072.

### Parameter `language`

| Parameter | Default | Description |
|---|---|---|
| `language` | `english` | Language used for agent responses in chat conversations (does not affect code comments, documentation, or other technical writing) |
| `guard.ask` | `false` | If true and guard blocks a command, ask user for confirmation before executing |

| Parameter | Description |
|---|---|
| `guard.disable` | Disable guard agent for exec commands. When true, security checks via guard are skipped |
| `guard.ask` | If true and guard blocks a command, ask user for confirmation before executing |

### Parameter `llm.reasoning_effort`

| Parameter | Default | Description |
|---|---|---|
| `llm.reasoning_effort` | `high` | LLM reasoning effort. Controls the tradeoff between reasoning quality and latency. Valid values: `low`, `medium`, `high` |

### Parameters `rag.chunk`

| Parameter | Default | Description |
|---|---|---|
| `rag.chunk.size` | `512` | Maximum chunk size in Unicode runes |
| `rag.chunk.overlap` | `64` | Overlap between adjacent chunks in runes (~12.5 %) |

### Parameters `rag.search`

| Parameter | Default | Description |
|---|---|---|
| `rag.search.limit` | `10` | Maximum number of results returned by Qdrant per query |
| `rag.search.threshold` | `0.40` | Minimum cosine similarity; results below the threshold are discarded |
| `rag.search.multi_query` | `2` | Number of extra query reformulations generated by the LLM to improve recall. `0` disables multi-query. Each variant adds one embedding + one Qdrant call, all run in parallel |
| `rag.search.dedup_threshold` | `0.95` | Cosine similarity above which two result chunks are considered near-duplicates; the lower-scoring one is dropped. `0` disables deduplication |

### Parameters `compact`

| Parameter | Description |
|---|---|
| `compact.threshold` | Token count threshold that triggers history compression. `0` = auto: 80 % of `llm.context` |
| `compact.save_tail` | Number of trailing messages kept unchanged. `< 1` — compress all messages |

### Guard configuration

The guard can be configured via `guard.disable` and `guard.ask` parameters:

- **`guard.disable`** (default: `false`): When set to `true`, skips all security checks via the guard agent and executes commands directly.
- **`guard.ask`** (default: `false`): When set to `true` and guard blocks a command, the user is prompted for confirmation before execution. If user confirms with 'y', the command executes; otherwise, it is blocked.

Example configuration:

```yaml
guard:
  disable: false    # Set to true to skip guard checks
  ask: true         # Set to true to prompt user when guard blocks commands
```
