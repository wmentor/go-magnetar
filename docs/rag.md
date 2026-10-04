# RAG (Retrieval-Augmented Generation)

RAG allows the agent to search a knowledge base for relevant information before generating responses. This document covers installation, configuration, and usage.

## Requirements

- API key for any OpenAI-compatible provider (for LLM, embedding model, and optionally for web content cleaning)

**Note:** Qdrant is **no longer required**. The default vector store is **Chromem** (built-in, in-memory with optional persistence). Qdrant is available as an optional external store for production deployments.

### Vector Stores

go-magnetar supports two vector storage backends:

#### Chromem (default)

- **Built-in**: No external dependencies required
- **In-memory by default**: Store exists only during session (when `rag.chromem.data_dir` is empty)
- **Persistent mode**: Set `rag.chromem.data_dir` to persist vectors to disk
- **Use case**: Local development, testing, edge deployments
- **Configuration**: `rag.store.type: chromem` (default)

#### Qdrant (external)

- **External server**: Requires running Qdrant container
- **Full persistence**: Automatic disk persistence with advanced filtering
- **Use case**: Production, large-scale deployments
- **Configuration**: `rag.store.type: qdrant`

Switch between stores via config:
```yaml
rag:
  store:
    type: chromem  # or qdrant
```

See [docs/architecture.md](./architecture.md) for vector store architecture details.

## Installation

### Using Chromem (default)

No installation required. Chromem is built into go-magnetar.

For persistent storage, set `rag.chromem.data_dir` in the config. If omitted or empty (`""`), Chromem runs in in-memory mode (data lost on restart):
```yaml
rag:
  enable: true
  chromem:
    data_dir: ~/.go-magnetar/store  # omit for in-memory mode
```

### In-Memory Mode

For in-memory storage (data lost on restart), explicitly set `data_dir` to an empty string:

```yaml
rag:
  enable: true
  chromem:
    data_dir: ""  # in-memory mode
```

### Using Qdrant (optional)

Qdrant can be run locally via Docker:

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

Or install Qdrant directly on your system following the [official installation guide](https://qdrant.tech/documentation/guides/installation/).

## Configuration

Configuration files are automatically created in `~/.go-magnetar/` on first run. Plugin configurations are stored separately in `~/.go-magnetar/plugins/` directory.

### Enable RAG Plugin

The RAG plugin configuration file is located at `~/.go-magnetar/plugins/rag.yml`. To enable RAG, edit this file and set `enable: true`:

```yaml
rag:
  enable: true
  store:
    type: chromem  # or qdrant
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
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
    data_dir: ~/.go-magnetar/store  # optional: omit for in-memory only
  qdrant:
    connstr: http://localhost:6333  # required only when using qdrant
    collection: documents
```

### Configuration Parameters

| Key | Type | Description |
|---|---|---|
| `rag.enable` | boolean | Enable/disable RAG functionality (default: `false`) |
| `rag.store.type` | string | Vector store type: `chromem` (built-in) or `qdrant` (external) |
| `rag.llm.base_url` | string | Embedding model endpoint URL |
| `rag.llm.api_key` | string | API key for embedding model |
| `rag.llm.model` | string | Embedding model name (e.g., `text-embedding-3-small`) |
| `rag.llm.vector_size` | integer | Vector dimension (1536 for `text-embedding-3-small`, 1024 for `text-embedding-3-large`) |
| `rag.chunk.size` | integer | Maximum chunk size in runes (default: `2048`) |
| `rag.chunk.overlap` | integer | Overlapping runes between chunks (default: `256`) |
| `rag.search.limit` | integer | Maximum results per query (default: `10`) |
| `rag.search.threshold` | float | Minimum cosine similarity score (default: `0.40`) |
| `rag.search.multi_query` | integer | Number of additional query reformulations (default: `2`) |
| `rag.search.dedup_threshold` | float | Near-duplicate suppression threshold (default: `0.95`) |
| `rag.chromem.data_dir` | string | Chromem data directory for persistent storage (empty = in-memory only) |
| `rag.chromem.collection` | string | Chromem collection name (default: `documents`) |
| `rag.qdrant.connstr` | string | Qdrant connection string (REST port 6333; gRPC 6334 used automatically) |
| `rag.qdrant.collection` | string | Collection name (created automatically if missing) |

### Example: Complete RAG Setup with Chromem (default)

```yaml
version: "1.0"

language: english
verbose: true
profile: default

rag:
  enable: true
  store:
    type: chromem
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
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
    data_dir: ~/.go-magnetar/store

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

### Example: RAG Setup with Qdrant

```yaml
version: "1.0"

language: english
verbose: true
profile: default

rag:
  enable: true
  store:
    type: qdrant
  llm:
    base_url: https://api.openai.com/v1
    api_key: $env:OPENAI_API_KEY
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

## Usage

### First Run

On first run, go-magnetar automatically creates configuration files in `~/.go-magnetar/`. Plugin configurations are stored in `~/.go-magnetar/plugins/` directory. The RAG plugin is disabled by default.

### Indexing Documents

Once RAG is enabled, you can index documents via the `/index` command:

```bash
./bin/go-magnetar
> /index docs/guide.md
Indexed 15 chunks from docs/guide.md
```

Or index from a URL:

```bash
> /index https://example.com/article
Indexed 8 chunks from https://example.com/article
```

### Searching

The agent automatically uses RAG search (`rag_search` tool) when answering questions. If relevant information exists in the knowledge base, it will be used to generate the answer.

## Search Strategy

The agent follows this strategy:

1. **Always first**: Calls `rag_search` to check the knowledge base
2. **Fallback**: If no relevant results, uses `web_fetch` or `web_search` for external information
3. **Explicit failure**: If neither source has an answer, states this clearly

### Multi-Query Search

When `rag.search.multi_query > 0`, the agent generates alternative phrasings of the query via the LLM and searches with all variants. Results are merged by keeping the best score per unique chunk.

### Near-Duplicate Suppression

Chunks with embeddings having cosine similarity above `rag.search.dedup_threshold` are considered near-duplicates. The chunk with the lower score is removed.

## Performance Tips

- **Vector size**: Match `rag.llm.vector_size` to your embedding model (1536 for `text-embedding-3-small`, 1024 for `text-embedding-3-large`)
- **Chunk size**: Larger chunks (`4096`) may improve relevance but increase embedding costs
- **Threshold**: Lower `rag.search.threshold` (e.g., `0.35`) returns more results, higher returns fewer but more relevant
- **Multi-query**: Set `rag.search.multi_query` to `2-3` for better coverage of semantic variations
- **Chromem persistence**: Set `rag.chromem.data_dir` to persist vectors to disk between sessions

## Troubleshooting

### Embedding Errors

Check that `rag.llm.api_key` is valid and `rag.llm.model` matches your provider's embedding models.

### No Results Found

- Verify documents were indexed successfully
- Check `rag.search.threshold` — may be too high
- Try lowering `rag.search.threshold` to `0.30`
- Enable `rag.search.multi_query` to improve coverage

### Qdrant Connection Failed

Ensure Qdrant is running and accessible:

```bash
curl http://localhost:6333
```
