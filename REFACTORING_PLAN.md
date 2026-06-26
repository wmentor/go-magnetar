# План переделки internal/config/config.go

## Цель
Упростить структуру конфига, используя `*koanf.Koanf` как поле и добавив методы доступа с поддержкой вложенных ключей.

## Изменения

### 1. Удалить все вложенные структуры конфига
- Удалить: `LLMConfig`, `EmbeddingConfig`, `QdrantConfig`, `ChunkConfig`, `WebFetchConfig`, `SearchConfig`, `RAGConfig`, `LogConfig`, `ConfluenceConfig`, `JIRAConfig`, `CompactConfig`

### 2. Изменить структуру `Config`
```go
type Config struct {
    cfg *koanf.Koanf
}
```

### 3. Добавить методы доступа к параметрам
```go
func (c *Config) String(key string) string
func (c *Config) Bool(key string) bool
func (c *Config) Int(key string) int
func (c *Config) Float64(key string) float64
```

### 4. Реализация методов доступа
- Использовать встроенные методы `koanf.Koanf`: `String()`, `Bool()`, `Int()`, `Float64()`
- Поддерживать вложенные ключи через точку: `"rag.search.limit"`
- Возвращать дефолтное значение для типа при ошибке:
  - `String()` → `""`
  - `Bool()` → `false`
  - `Int()` → `0`
  - `Float64()` → `0.0`
- Метод `Float32()` **не нужен** (используется `Float64()`)

### 5. Обновить функцию `Load()`
```go
func Load(path string) (*Config, error) {
    k := koanf.New(".")
    if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
        return nil, fmt.Errorf("config: failed to load defaults: %w", err)
    }
    if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
        return nil, fmt.Errorf("config: failed to load %q: %w", path, err)
    }
    return &Config{cfg: k}, nil
}
```

### 6. Обновить `SetupLogger()`
- Доступ к `cfg.Log.Level` заменить на `cfg.String("log.level")`

### 7. Обновить `defaults`
- Оставить как есть, ключи уже в правильном формате

### 8. Обновить все файлы, использующие старую структуру

#### internal/config/config.go (61 usage)
- `cfg.Log.Level` → `cfg.String("log.level")`

#### internal/agent/indexer/indexer.go (2 usages)
- `cfg.RAG.Chunk.Size` → `cfg.Int("rag.chunk.size")`
- `cfg.RAG.Chunk.Overlap` → `cfg.Int("rag.chunk.overlap")`

#### internal/tools/rag/rag.go (19 usages)
- `cfg.RAG.LLM.APIKey` → `cfg.String("rag.llm.api_key")`
- `cfg.RAG.LLM.BaseURL` → `cfg.String("rag.llm.base_url")`
- `cfg.RAG.LLM.VectorSize` → `cfg.Int("rag.llm.vector_size")`
- `cfg.RAG.LLM.Model` → `cfg.String("rag.llm.model")`
- `cfg.RAG.Qdrant.ConnStr` → `cfg.String("rag.qdrant.connstr")`
- `cfg.RAG.Qdrant.Collection` → `cfg.String("rag.qdrant.collection")`
- `cfg.RAG.Search.Limit` → `cfg.Int("rag.search.limit")`
- `cfg.RAG.Search.Threshold` → `cfg.Float64("rag.search.threshold")`
- `cfg.RAG.Search.DedupThreshold` → `cfg.Float64("rag.search.dedup_threshold")`
- `cfg.RAG.Search.MultiQuery` → `cfg.Int("rag.search.multi_query")`
- `cfg.LLM.APIKey` → `cfg.String("llm.api_key")`
- `cfg.LLM.BaseURL` → `cfg.String("llm.base_url")`
- `cfg.LLM.Model` → `cfg.String("llm.model")`
- `cfg.LLM.Context` → `cfg.Int("llm.context")`

#### internal/agent/summarizer/summarizer.go (7 usages)
- `cfg.LLM.APIKey` → `cfg.String("llm.api_key")`
- `cfg.LLM.BaseURL` → `cfg.String("llm.base_url")`
- `cfg.LLM.Model` → `cfg.String("llm.model")`
- `cfg.LLM.Context` → `cfg.Int("llm.context")`
- `cfg.Compact.Threshold` → `cfg.Int("compact.threshold")`
- `cfg.Compact.SaveTail` → `cfg.Int("compact.save_tail")`

#### internal/tools/web/fetch.go (9 usages)
- `cfg.WebFetch.BaseURL` → `cfg.String("webfetch.base_url")`
- `cfg.WebFetch.APIKey` → `cfg.String("webfetch.api_key")`
- `cfg.WebFetch.Model` → `cfg.String("webfetch.model")`
- `cfg.WebFetch.Context` → `cfg.Int("webfetch.context")`
- `cfg.Confluence.BaseURL` → `cfg.String("confluence.base_url")`
- `cfg.Confluence.APIKey` → `cfg.String("confluence.api_key")`
- `cfg.JIRA.BaseURL` → `cfg.String("jira.base_url")`
- `cfg.JIRA.APIKey` → `cfg.String("jira.api_key")`

#### internal/plugins/chatcmd/stat/plugin.go (3 usages)
- `cfg.LLM.Model` → `cfg.String("llm.model")`
- `cfg.RAG.LLM.Model` → `cfg.String("rag.llm.model")`
- `cfg.RAG.LLM.VectorSize` → `cfg.Int("rag.llm.vector_size")`

#### internal/agent/markdown/preprocessor.go (4 usages)
- `cfg.WebFetch.BaseURL` → `cfg.String("webfetch.base_url")`
- `cfg.WebFetch.APIKey` → `cfg.String("webfetch.api_key")`
- `cfg.WebFetch.Model` → `cfg.String("webfetch.model")`
- `cfg.WebFetch.Context` → `cfg.Int("webfetch.context")`

#### internal/agent/chat/agent.go (8 usages)
- `cfg.LLM.APIKey` → `cfg.String("llm.api_key")`
- `cfg.LLM.BaseURL` → `cfg.String("llm.base_url")`
- `cfg.LLM.Model` → `cfg.String("llm.model")`
- `cfg.LLM.Context` → `cfg.Int("llm.context")`
