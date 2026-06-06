# go-magnetar

Утилита объединяет два AI-агента: **индексатор документов** в RAG и **чат-агент**, который отвечает на вопросы строго на основе проиндексированных данных.

## Требования

- Go 1.26.4+
- Qdrant (gRPC порт `6334`)
- API-ключ для OpenAI-совместимого LLM и embedding-модели

Запустить Qdrant локально:

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## Сборка

```bash
make build        # бинарник -> bin/go-magnetar
make tidy         # синхронизировать go.mod / go.sum
make lint         # go vet ./...
make clean        # удалить bin/
```

## Конфигурация

Скопировать пример и заполнить реальными значениями:

```bash
cp configs/config.yaml my-config.yaml
```

```yaml
llm:
  base_url: https://api.openai.com/v1   # OpenAI-совместимый endpoint
  api_key: YOUR_API_KEY
  model: gpt-4o
  context: 128000                        # лимит токенов передаваемого контекста

rag:
  llm:
    base_url: https://api.openai.com/v1
    api_key: YOUR_API_KEY
    model: text-embedding-3-small        # embedding-модель
    vector_size: 1536                    # размерность векторов модели
  qdrant:
    connstr: http://localhost:6333       # адрес Qdrant (REST-порт; gRPC 6334 используется автоматически)
    collection: documents                # имя коллекции (создаётся автоматически)

log:
  level: info                            # debug | info | warn | error

compact:
  threshold: 0    # порог в токенах для запуска сжатия; 0 = авто (80% от llm.context)
  save_tail: 6    # число хвостовых сообщений, которые сохраняются без изменений
```

> `vector_size` должен совпадать с размерностью выбранной embedding-модели.
> Для `text-embedding-3-small` — 1536, для `text-embedding-ada-002` — 1536, для `text-embedding-3-large` — 3072.

### Параметры `compact`

| Параметр | Описание |
|---|---|
| `compact.threshold` | Число токенов, при достижении которого запускается сжатие истории. `0` — авто: 80 % от `llm.context` |
| `compact.save_tail` | Число последних сообщений, которые сохраняются без изменений. `< 1` — сжимаются все сообщения |

## Агент-индексатор

Читает файлы `.md` и `.txt`, разбивает на логические блоки (до 500 токенов) и сохраняет в Qdrant. Разбивку и сохранение выполняет LLM через tool-use loop.

### Индексировать один файл

```bash
./bin/go-magnetar indexer -c my-config.yaml -f path/to/document.md
```

### Индексировать директорию рекурсивно

```bash
./bin/go-magnetar indexer -c my-config.yaml -d path/to/docs/
```

Все файлы `.md` и `.txt` будут обработаны. При ошибке отдельного файла — в stderr пишется `slog.Error` и обработка продолжается.

### Поведение при дублировании

Каждый блок идентифицируется по `UUID`.

### Инструменты индексатора

| Инструмент | Сигнатура | Описание |
|---|---|---|
| `file_read` | `(filename: string) -> string` | Читает содержимое файла |
| `rag_save` | `(content: string) -> bool` | Сохраняет фрагмент в Qdrant |

## Чат-агент

Интерактивный REPL. Поддерживает multi-turn диалог — история сообщений сохраняется на протяжении всей сессии.

```bash
./bin/go-magnetar agent -c my-config.yaml
# или через Makefile (использует configs/config.yaml):
make run-agent
```

### Работа с агентом

```
> Что такое go-magnetar?
Утилита go-magnetar реализует...

> Какие команды она поддерживает?
Поддерживаются две команды: indexer и agent...

> ^D
```

Завершение — `Ctrl+D` (EOF). Пустые строки игнорируются.

### Стратегия поиска

Агент сам решает, когда обращаться к `rag_search`. Критерий — нужна новая информация или нужно уточнить уже имеющуюся. Агент не выдумывает: если `rag_search` не вернул релевантных результатов, он сообщает об этом явно.

### Инструменты чат-агента

| Инструмент | Сигнатура | Описание |
|---|---|---|
| `rag_search` | `(query: string) -> string` | Возвращает топ-5 релевантных фрагментов из Qdrant |

## Архитектура

```
cmd/go-magnetar/main.go          — entrypoint
internal/
  config/config.go               — загрузка YAML-конфига, инициализация slog
  cmd/
    cmd.go                       — root CLI (kong)
    indexer/cmd.go               — подкоманда indexer
    agent/cmd.go                 — подкоманда agent
  tools/
    generic/generic.go           — инструмент file_read
    rag/rag.go                   — инструменты rag_save, rag_search; подключение к Qdrant
  agent/
    indexer/indexer.go           — агент-индексатор, tool-use loop
    chat/agent.go                — чат-агент, REPL, tool-use loop
    summarizer/summarizer.go     — агент сжатия истории
```

### Поток данных: индексация

```
CLI --> IndexFile(filename)
         --> LLM (system prompt + "index file: X")
               --> tool_call: file_read(filename)
                     --> os.ReadFile
               --> tool_call: rag_save(block1)
                     --> uuid.NewString() -> id
                     --> embed(block1) -> vector
                     --> qdrant.Upsert(id, vector, payload{text: block1})
               --> tool_call: rag_save(block2) ...
               --> finish: "Saved N blocks"
```

### Поток данных: чат

```
CLI --> Run() --> REPL
         --> ask(user_input)
               --> [если достигнут порог токенов]
                     --> summarizer.Compact(history)
                           --> системное сообщение сохраняется
                           --> последние save_tail сообщений сохраняются
                           --> LLM: сжать старые сообщения -> одно summary-сообщение
               --> trimMessages(history) — обрезка до размера контекстного окна
               --> LLM (system prompt + history + user_input)
                     --> tool_call: rag_search(query)
                           --> embed(query) -> vector
                           --> qdrant.Query(vector, limit=5)
                           --> return top-5 texts
                     --> LLM формирует ответ на основе результатов
               --> вывод ответа в stdout
```

## Логирование

Все логи пишутся в `stderr` в формате `slog` text. Уровни:

| Уровень | Когда |
|---|---|
| `DEBUG` | Детали вызовов инструментов (имя, аргументы); число обрезанных/сжатых сообщений |
| `INFO` | Начало/конец индексации файла, создание коллекции, финальный ответ LLM; запуск сжатия истории |
| `WARN` | Перезапись существующего документа в Qdrant |
| `ERROR` | Ошибки чтения файлов, сбои embedding/Qdrant, ошибки LLM; сбой сжатия (не фатальный) |

Для подробного вывода установить `log.level: debug` в конфиге.

## Зависимости

| Пакет | Назначение |
|---|---|
| `github.com/alecthomas/kong` | CLI-парсер |
| `github.com/sashabaranov/go-openai` | LLM и embedding клиент |
| `github.com/qdrant/go-client` | Клиент Qdrant (gRPC) |
| `github.com/knadh/koanf/v2` | Загрузка YAML-конфига |
| `github.com/lmittmann/tint` | Цветной slog handler |
| `github.com/charmbracelet/glamour` | Рендеринг Markdown в терминале |
| `log/slog` | Структурированное логирование (stdlib) |
