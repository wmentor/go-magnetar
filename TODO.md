# TODO — План реализации go-magnetar

---

## 1. Инициализация проекта

- [ ] Выполнить `go mod init github.com/wmentor/go-magnetar` с директивой `go 1.26.4`
- [ ] Создать структуру каталогов:
  ```
  bin/
  cmd/go-magnetar/
  configs/
  internal/cmd/
  internal/cmd/agent/
  internal/cmd/indexer/
  internal/config/
  internal/tools/generic/
  internal/tools/rag/
  internal/agent/chat/
  internal/agent/indexer/
  ```
- [ ] Создать `.gitignore`:
  - папка `bin/`
  - `*.env`
  - не добавлять `configs/config.yaml` с реальными ключами (добавить в gitignore только реальный конфиг, пример оставить)
- [ ] Добавить зависимости:
  - `go get github.com/alecthomas/kong`
  - `go get github.com/sashabaranov/go-openai`
  - `go get github.com/qdrant/go-client`
  - `go get github.com/knadh/koanf/v2`
  - `go get github.com/knadh/koanf/parsers/yaml`
  - `go get github.com/knadh/koanf/providers/file`
- [ ] Выполнить `go mod tidy`

---

## 2. Конфигурация (`internal/config/config.go`)

### 2.1 Структуры данных

- [ ] Определить `LLMConfig`:
  ```go
  type LLMConfig struct {
      BaseURL string `koanf:"base_url"`
      APIKey  string `koanf:"api_key"`
      Model   string `koanf:"model"`
      Context int    `koanf:"context"`
  }
  ```
- [ ] Определить `EmbeddingConfig`:
  ```go
  type EmbeddingConfig struct {
      BaseURL    string `koanf:"base_url"`
      APIKey     string `koanf:"api_key"`
      Model      string `koanf:"model"`
      VectorSize int    `koanf:"vector_size"`
  }
  ```
- [ ] Определить `QdrantConfig`:
  ```go
  type QdrantConfig struct {
      ConnStr    string `koanf:"connstr"`
      Collection string `koanf:"collection"`
  }
  ```
- [ ] Определить `RAGConfig`:
  ```go
  type RAGConfig struct {
      LLM    EmbeddingConfig `koanf:"llm"`
      Qdrant QdrantConfig    `koanf:"qdrant"`
  }
  ```
- [ ] Определить `LogConfig`:
  ```go
  type LogConfig struct {
      Level string `koanf:"level"` // debug | info | warn | error, default: info
  }
  ```
- [ ] Определить корневую `Config`:
  ```go
  type Config struct {
      LLM RAGConfig `koanf:"llm"`
      RAG RAGConfig `koanf:"rag"`
      Log LogConfig `koanf:"log"`
  }
  ```

### 2.2 Загрузка конфига

- [ ] Реализовать `Load(path string) (*Config, error)`:
  - Создать экземпляр `koanf.New(".")`
  - Загрузить yaml-файл через `file.Provider` + `yaml.Parser`
  - Распарсить в структуру `Config` через `k.Unmarshal`
  - Вернуть ошибку с контекстом если файл не найден или невалиден

### 2.3 Инициализация логера

- [ ] Реализовать `SetupLogger(cfg *Config)`:
  - Смаппить строку уровня (`debug`/`info`/`warn`/`error`) в `slog.Level`
  - Если уровень не распознан — использовать `slog.LevelInfo`
  - Создать `slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})`
  - Установить через `slog.SetDefault`

### 2.4 Пример конфигурации

- [ ] Создать `configs/config.yaml`:
  ```yaml
  llm:
    base_url: https://api.openai.com/v1
    api_key: YOUR_API_KEY
    model: gpt-4o
    context: 128000
  rag:
    llm:
      base_url: https://api.openai.com/v1
      api_key: YOUR_API_KEY
      model: text-embedding-3-small
      vector_size: 1536
    qdrant:
      connstr: http://localhost:6333
      collection: documents
  log:
    level: info
  ```

---

## 3. Инструменты (`internal/tools`)

### 3.1 Generic tools (`internal/tools/generic/generic.go`)

- [ ] Определить структуру:
  ```go
  type GenericTools struct {
      cfg *config.Config
  }
  func New(cfg *config.Config) *GenericTools
  ```
- [ ] Реализовать `FileRead(filename string) (string, bool)`:
  - Читать файл через `os.ReadFile`
  - При ошибке логировать `slog.Error` и вернуть `"", false`
  - При успехе вернуть содержимое как строку и `true`
- [ ] Реализовать `Definition() openai.Tool` — JSON-схема инструмента `file_read`:
  - name: `"file_read"`
  - description: `"Read the contents of a file by its path"`
  - parameters: объект с полем `filename` (тип `string`, required)
- [ ] Реализовать `Dispatch(name string, args string) string` — диспетчер вызовов инструментов:
  - Распарсить `args` из JSON
  - Вызвать нужный метод по имени инструмента
  - Вернуть результат как строку (для передачи обратно в LLM как tool result)

### 3.2 RAG tools (`internal/tools/rag/rag.go`)

- [ ] Определить структуру:
  ```go
  type RAGTools struct {
      cfg        *config.Config
      embedClient *openai.Client
      qdrantConn  *grpc.ClientConn  // или REST клиент
  }
  func New(cfg *config.Config) (*RAGTools, error)
  ```
- [ ] В конструкторе `New`:
  - Создать `openai.Client` для embedding с кастомным `BaseURL` и `APIKey` из `cfg.RAG.LLM`
  - Подключиться к Qdrant по `cfg.RAG.Qdrant.ConnStr`
  - Убедиться что коллекция существует; если нет — создать с нужным `vector_size` и метрикой `Cosine`
- [ ] Реализовать вспомогательный метод `embed(text string) ([]float32, error)`:
  - Вызвать `openai.CreateEmbeddings` с моделью из `cfg.RAG.LLM.Model`
  - Вернуть вектор первого результата
- [ ] Реализовать `RagSave(content string) bool`:
  - Вычислить `id = fmt.Sprintf("%x", sha256.Sum256([]byte(content)))`
  - Вызвать `embed(content)` для получения вектора
  - Проверить существование точки в Qdrant через `Points.Get` по ID
  - Если точка найдена — `slog.Warn("document already exists, overwriting", "id", id)`
  - Выполнить `Points.Upsert` с вектором и payload `{"text": content}`
  - При ошибке — `slog.Error` и вернуть `false`
  - При успехе вернуть `true`
- [ ] Реализовать `RagSearch(query string) string`:
  - Вызвать `embed(query)` для получения вектора запроса
  - Выполнить `Points.Search` в Qdrant: топ-5 ближайших, с payload
  - Собрать тексты из payload всех результатов в одну строку (разделитель `\n\n---\n\n`)
  - При ошибке — `slog.Error` и вернуть пустую строку
- [ ] Реализовать `DefinitionSave() openai.Tool` — JSON-схема `rag_save`:
  - name: `"rag_save"`
  - description: `"Save a text fragment to the knowledge base"`
  - parameters: поле `content` (тип `string`, required)
- [ ] Реализовать `DefinitionSearch() openai.Tool` — JSON-схема `rag_search`:
  - name: `"rag_search"`
  - description: `"Search the knowledge base for relevant information"`
  - parameters: поле `query` (тип `string`, required)
- [ ] Реализовать `Dispatch(name string, args string) string` — диспетчер:
  - Распарсить JSON args
  - Вызвать `RagSave` или `RagSearch` по имени
  - Вернуть результат как строку

---

## 4. Агент-индексатор (`internal/agent/indexer/indexer.go`)

- [ ] Определить структуру:
  ```go
  type Indexer struct {
      cfg     *config.Config
      llm     *openai.Client
      generic *generic.GenericTools
      rag     *rag.RAGTools
  }
  func New(cfg *config.Config) (*Indexer, error)
  ```
- [ ] В конструкторе `New`:
  - Создать `openai.Client` с `BaseURL` и `APIKey` из `cfg.LLM`
  - Создать `GenericTools` и `RAGTools`
- [ ] Реализовать вспомогательный метод `runAgentLoop(messages []openai.ChatCompletionMessage) error`:
  - Собрать список инструментов: `generic.Definition()`, `rag.DefinitionSave()`
  - Отправить запрос `CreateChatCompletion` с `MaxTokens = cfg.LLM.Context`
  - Если в ответе `FinishReason == "tool_calls"`:
    - Для каждого `ToolCall` определить имя инструмента
    - Вызвать `generic.Dispatch` или `rag.Dispatch` в зависимости от имени
    - Добавить в `messages` сообщение роли `tool` с результатом
    - Добавить ответ ассистента в `messages`
    - Рекурсивно вызвать `runAgentLoop` с обновлёнными сообщениями
  - Если `FinishReason == "stop"` — вывести финальный ответ через `slog.Info` и завершить
- [ ] Реализовать `IndexFile(filename string) error`:
  - Сформировать начальные сообщения:
    - system: системный промпт индексатора из TASK.md
    - user: `"Please index the file: <filename>"`
  - Вызвать `runAgentLoop`
- [ ] Реализовать `IndexDirectory(dir string) error`:
  - Вызвать `filepath.WalkDir(dir, ...)`
  - В колбэке пропускать директории и файлы с расширением не `.md`/`.txt`
  - Для каждого подходящего файла вызвать `IndexFile(path)`
  - При ошибке `IndexFile` — `slog.Error("failed to index file", "file", path, "err", err)` и продолжить

---

## 5. Чат-агент (`internal/agent/chat/agent.go`)

- [ ] Определить структуру:
  ```go
  type ChatAgent struct {
      cfg      *config.Config
      llm      *openai.Client
      rag      *rag.RAGTools
      messages []openai.ChatCompletionMessage
  }
  func New(cfg *config.Config) (*ChatAgent, error)
  ```
- [ ] В конструкторе `New`:
  - Создать `openai.Client` с параметрами из `cfg.LLM`
  - Создать `RAGTools`
  - Инициализировать `messages` с системным промптом чат-агента из TASK.md
- [ ] Реализовать вспомогательный метод `ask(userInput string) (string, error)`:
  - Добавить в `messages` сообщение роли `user` с текстом вопроса
  - Запустить tool-use loop:
    - Отправить `CreateChatCompletion` с инструментом `rag.DefinitionSearch()` и `MaxTokens = cfg.LLM.Context`
    - Если `FinishReason == "tool_calls"`:
      - Для каждого `ToolCall` вызвать `rag.Dispatch`
      - Добавить ответ ассистента и результат инструмента в `messages`
      - Повторить запрос
    - Если `FinishReason == "stop"` — добавить ответ ассистента в `messages`, вернуть текст ответа
- [ ] Реализовать `Run() error` — REPL-цикл:
  - Создать `bufio.Scanner` над `os.Stdin`
  - Вывести приглашение `"> "` в stdout перед каждым вводом
  - Читать строки через `scanner.Scan()`
  - Завершать цикл при EOF (`!scanner.Scan()`)
  - Игнорировать пустые строки
  - Для каждой строки вызвать `ask(line)`
  - Вывести ответ в stdout с завершающим переводом строки
  - При ошибке `ask` — `slog.Error` и продолжить (не падать)

---

## 6. CLI (`internal/cmd`)

### 6.1 Root command (`internal/cmd/cmd.go`)

- [ ] Определить корневую структуру kong:
  ```go
  type CLI struct {
      Indexer IndexerCmd `cmd:"" help:"Index files into RAG"`
      Agent   AgentCmd   `cmd:"" help:"Run interactive chat agent"`
  }
  ```
- [ ] Функция `Execute()` — точка входа: `kong.Parse(&cli)` + `cmd.Run()`

### 6.2 Indexer command (`internal/cmd/indexer/cmd.go`)

- [ ] Определить структуру:
  ```go
  type IndexerCmd struct {
      Config    string `short:"c" help:"Path to config file" required:""`
      File      string `short:"f" help:"File to index"`
      Directory string `short:"d" help:"Directory to index"`
  }
  ```
- [ ] Реализовать `Run() error`:
  - Вызвать `config.Load(cfg.Config)`
  - Вызвать `config.SetupLogger(cfg)`
  - Если оба `File` и `Directory` пустые — вернуть `fmt.Errorf("--file or --directory required")`
  - Создать `indexer.New(cfg)`
  - Если задан `--file` — вызвать `idx.IndexFile(cmd.File)`
  - Если задана `--directory` — вызвать `idx.IndexDirectory(cmd.Directory)`

### 6.3 Agent command (`internal/cmd/agent/cmd.go`)

- [ ] Определить структуру:
  ```go
  type AgentCmd struct {
      Config string `short:"c" help:"Path to config file" required:""`
  }
  ```
- [ ] Реализовать `Run() error`:
  - Вызвать `config.Load(cmd.Config)`
  - Вызвать `config.SetupLogger(cfg)`
  - Создать `chat.New(cfg)`
  - Вызвать `agent.Run()`

---

## 7. Entrypoint (`cmd/go-magnetar/main.go`)

- [ ] Импортировать `internal/cmd`
- [ ] В `main()` вызвать `cmd.Execute()`
- [ ] При ошибке — вывести в stderr и завершить с кодом 1

---

## 8. Makefile

- [ ] Создать `Makefile` в корне проекта с командами:
  - `build` — компилирует бинарник в `bin/go-magnetar`:
    ```makefile
    build:
        go build -o bin/go-magnetar ./cmd/go-magnetar
    ```
  - `clean` — удаляет папку `bin/`:
    ```makefile
    clean:
        rm -rf bin/
    ```
  - `run-agent` — запускает агента с дефолтным конфигом:
    ```makefile
    run-agent: build
        ./bin/go-magnetar agent -c configs/config.yaml
    ```
  - `lint` — статический анализ:
    ```makefile
    lint:
        go vet ./...
    ```
  - `tidy` — приводит зависимости в порядок:
    ```makefile
    tidy:
        go mod tidy
    ```
- [ ] Убедиться что `bin/` добавлен в `.gitignore`
- [ ] Убедиться что `make build` создаёт `bin/go-magnetar` и он запускается

---

## 9. Финальная проверка

- [ ] `go build ./...` — убедиться что компилируется без ошибок
- [ ] `go vet ./...` — убедиться что нет предупреждений
- [ ] Поднять Qdrant локально (`docker run -p 6333:6333 qdrant/qdrant`)
- [ ] Запустить индексацию одного файла:
  ```
  ./go-magnetar indexer -c configs/config.yaml -f README.md
  ```
  Ожидаемо: в логах видны вызовы `rag_save`, финальное сообщение о кол-ве блоков
- [ ] Запустить индексацию директории:
  ```
  ./go-magnetar indexer -c configs/config.yaml -d ./docs
  ```
  Ожидаемо: обработаны все `.md`/`.txt` файлы, ошибки отдельных файлов не прерывают процесс
- [ ] Проверить дубликат: проиндексировать один файл дважды — в логах должен быть `slog.Warn` с ID
- [ ] Запустить чат-агент:
  ```
  ./go-magnetar agent -c configs/config.yaml
  ```
  Задать вопрос по проиндексированным данным — убедиться что агент отвечает на основе RAG
- [ ] Задать вопрос вне контекста RAG — агент должен сообщить что информации нет, не выдумывать
- [ ] Проверить завершение по EOF (Ctrl+D) — процесс должен завершаться чисто
