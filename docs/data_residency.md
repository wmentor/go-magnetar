# Data Residency

## 100% Local Processing with Chromem and Ollama

When using **Chromem** (built-in vector store) and **local LLMs** (e.g., Ollama, lm-studio, any OpenAI-compatible server running on your machine), **your data never leaves your computer**. All processing — including document embedding, vector storage, and LLM inference — happens entirely on your local machine.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        LOCAL MACHINE                             │
│                                                                 │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│  │   Document   │      │   Chromem    │      │   Local LLM  │  │
│  │     Files    │ ───> │ Vector Store │ ───> │   (e.g.,    │  │
│  │              │      │              │      │   Ollama)    │  │
│  └──────────────┘      └──────────────┘      └──────────────┘  │
│                                                                 │
│          └── Embedding Compute ──> Local GPU/CPU              │
│          └── LLM Inference ─────> Local GPU/CPU               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### What Happens Locally

| Operation | Location | External Network |
|-----------|----------|------------------|
| Document ingestion | Local | ❌ Not required |
| Text chunking | Local | ❌ Not required |
| Embedding computation | Local | ❌ Not required |
| Vector storage (Chromem) | Local | ❌ Not required |
| RAG retrieval | Local | ❌ Not required |
| LLM inference | Local | ❌ Not required |
| Answer generation | Local | ❌ Not required |

### Requirements for 100% Local Operation

To ensure your data stays on your machine, follow these guidelines:

1. **Use Chromem as vector store** (default):
   ```yaml
   rag:
     store:
       type: chromem  # Built-in, no external dependencies
   ```

2. **Point to local LLM server** (e.g., Ollama):
   ```yaml
   profiles:
     default:
       llm:
         base_url: http://localhost:11434/v1  # Ollama default
         api_key: not-needed  # Local LLMs typically don't require API keys
         model: llama3.1:8b
   ```

3. **Disable any cloud plugins** (Confluence, JIRA, GitHub, etc.) if you don't want external network access:
   ```yaml
   plugins:
     confluence:
       enable: false
     jira:
       enable: false
     github:
       enable: false
   ```

### Configuration Examples

#### Fully Local Setup (No Internet Required)

```yaml
version: "1.0"
profile: local

profiles:
  local:
    llm:
      base_url: http://localhost:11434/v1
      api_key: not-needed
      model: llama3.1:8b
      context: 8192
      temperature: 0.7
    
    rag:
      enable: true
      store:
        type: chromem
      llm:
        base_url: http://localhost:11434/v1
        api_key: not-needed
        model: nomic-embed-text:1.0
        vector_size: 768
      chromem:
        data_dir: ~/.go-magnetar/store
        collection: documents
```

#### Local LLM + Cloud Embedding (Partial Cloud)

If you use a local LLM but cloud-based embedding models:

```yaml
profiles:
  hybrid:
    llm:
      base_url: http://localhost:11434/v1
      api_key: not-needed
      model: llama3.1:8b
      context: 8192
    
    rag:
      enable: true
      store:
        type: chromem
      llm:
        base_url: https://api.openai.com/v1
        api_key: $env:OPENAI_API_KEY
        model: text-embedding-3-small
        vector_size: 1536
      chromem:
        data_dir: ~/.go-magnetar/store
```

**Note:** In this hybrid setup, document embeddings are sent to the cloud, but:
- LLM inference remains local
- Your documents stay on your machine
- Only embedding vectors (not full document content) are sent to the cloud

### Verification

You can verify local-only operation using network monitoring tools:

```bash
# macOS
sudo lsof -i -P | grep go-magnetar

# Linux
sudo lsof -i -P | grep go-magnetar

# Windows (PowerShell)
Get-NetTCPConnection -OwningProcess $(Get-Process go-magnetar).Id
```

When using Chromem + Ollama locally, you should see **no network connections** (except possibly localhost loopback for Ollama).

### Security Implications

With Chromem + local LLMs:

- ✅ **Zero data exfiltration risk** — nothing is sent to external servers
- ✅ **Compliance-friendly** — suitable for regulated industries (HIPAA, GDPR, PCI-DSS)
- ✅ **Air-gapped compatible** — works on disconnected machines
- ✅ **No API key exposure** — no need to store cloud credentials
- ✅ **Offline operation** — works without internet connection

### When External Services Are Used

If you enable cloud-based plugins or use cloud embedding models, the following data flows occur:

| Component | Data Sent | Purpose | Can Be Disabled |
|-----------|-----------|---------|-----------------|
| Cloud embedding API | Document chunks | Generate vector embeddings | Yes (use local embedding model) |
| Confluence plugin | Page content | Read/write pages | Yes (disable plugin) |
| JIRA plugin | Issue data | Read/create issues | Yes (disable plugin) |
| GitHub plugin | Repository data | Read repos/issues | Yes (disable plugin) |

All external integrations are **disabled by default** and can be enabled/disabled per-profile.

### Summary

| Scenario | Data Stays Local? |
|----------|-------------------|
| Chromem + Ollama | ✅ Yes |
| Chromem + Local LLM | ✅ Yes |
| Chromem + Cloud Embeddings | ⚠️ Embeddings only (not full content) |
| Qdrant Cloud | ❌ No |
| Cloud plugins enabled | ❌ No |

For maximum privacy and compliance, use the fully local configuration above.
