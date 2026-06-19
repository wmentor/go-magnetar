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
  chunk:
    size: 512                            # максимальный размер чанка в рунах (умолч. 512)
    overlap: 64                          # перекрытие между соседними чанками в рунах (умолч. 64)
  search:
    limit: 10                            # максимальное число результатов на запрос (умолч. 10)
    threshold: 0.40                      # минимальный порог cosine similarity 0–1 (умолч. 0.40)
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

### Параметры `rag.chunk`

| Параметр | Умолч. | Описание |
|---|---|---|
| `rag.chunk.size` | `512` | Максимальный размер чанка в Unicode-рунах |
| `rag.chunk.overlap` | `64` | Перекрытие между соседними чанками в рунах (~12.5 %) |

### Параметры `rag.search`

| Параметр | Умолч. | Описание |
|---|---|---|
| `rag.search.limit` | `10` | Максимальное число результатов, возвращаемых Qdrant на один запрос |
| `rag.search.threshold` | `0.40` | Минимальный cosine similarity; результаты ниже порога отбрасываются |

### Параметры `compact`

| Параметр | Описание |
|---|---|
| `compact.threshold` | Число токенов, при достижении которого запускается сжатие истории. `0` — авто: 80 % от `llm.context` |
| `compact.save_tail` | Число последних сообщений, которые сохраняются без изменений. `< 1` — сжимаются все сообщения |

## Агент-индексатор

Читает файлы `.md` и `.txt` или веб-страницы (по URL), разбивает содержимое на перекрывающиеся чанки с учётом границ абзацев и Markdown-заголовков, вычисляет embedding-векторы и сохраняет в Qdrant. Каждый чанк идентифицируется детерминированным UUID v5, вычисленным из содержимого, — повторная индексация одного файла не создаёт дублей.

### Индексировать один файл

```bash
./bin/go-magnetar indexer -c my-config.yaml -f path/to/document.md
```

### Индексировать директорию рекурсивно

```bash
./bin/go-magnetar indexer -c my-config.yaml -d path/to/docs/
```

Все файлы `.md` и `.txt` будут обработаны. При ошибке отдельного файла — в stderr пишется `slog.Error` и обработка продолжается.

### Индексировать URL

```bash
./bin/go-magnetar indexer -c my-config.yaml -u https://example.com/article
```

HTML-страница очищается от рекламы, навигации и прочего шума, конвертируется в Markdown и разбивается на чанки так же, как локальные файлы.

### Поведение при дублировании

ID каждого чанка — UUID v5, вычисленный из его текста (`uuid.NewSHA1`). Повторный вызов `rag_save` с тем же содержимым выполняет `Upsert` по тому же ID — существующая точка перезаписывается, а не дублируется.

### Инструменты индексатора

| Инструмент | Сигнатура | Описание |
|---|---|---|
| `rag_save` | `(content: string) -> bool` | Сохраняет фрагмент в Qdrant |
| `web_fetch` | `(url: string) -> string` | Загружает и очищает веб-страницу, возвращает Markdown |

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

Завершение — `Ctrl+D` (EOF) или команда `/exit`. Пустые строки игнорируются.

### Встроенные команды чата

| Команда | Псевдонимы | Описание |
|---|---|---|
| `/help` | `help` | Вывести список доступных команд |
| `/exit` | `exit` | Завершить сессию и выйти из программы |
| `/compact` | — | Немедленно сжать историю через summarizer, не дожидаясь автоматического порога |
| `/new` | — | Начать новую сессию и очистить историю |
| `/stat` | — | Вывести статистику контекста: число сообщений, оценку токенов, размер в байтах, название LLM-модели, название RAG-модели и размер вектора |

Команды обрабатываются в методе `handleCommand` (`internal/agent/chat/agent.go`) до отправки ввода в LLM; в историю сообщений не добавляются. Сравнение регистронезависимое.

Текст справки хранится в константе `helpText` того же файла. Данные для `/stat` берутся напрямую из `a.cfg`: `cfg.LLM.Model`, `cfg.RAG.LLM.Model`, `cfg.RAG.LLM.VectorSize`.

### Стратегия поиска

Агент **всегда** сначала обращается к `rag_search`, даже если считает, что уже знает ответ. Если `rag_search` вернул релевантные результаты — ответ формируется исключительно на их основе, `web_fetch` не вызывается. `web_fetch` используется только как fallback: когда `rag_search` не нашёл ничего релевантного и пользователю нужна внешняя или актуальная информация. Если ни один инструмент не дал результата — агент сообщает об этом явно.

### Инструменты чат-агента

| Инструмент | Сигнатура | Описание |
|---|---|---|
| `rag_search` | `(query: string) -> string` | Возвращает топ-N релевантных фрагментов из Qdrant (N задаётся `rag.search.limit`) |
| `web_fetch` | `(url: string) -> string` | Загружает и очищает веб-страницу (fallback, если RAG не дал результата) |

## Архитектура

```
cmd/go-magnetar/main.go          — entrypoint
internal/
  config/config.go               — загрузка YAML-конфига, инициализация slog
  chunk/chunk.go                 — разбивка текста на чанки (UTF-8, границы абзацев/заголовков)
  cmd/
    cmd.go                       — root CLI (kong)
    indexer/cmd.go               — подкоманда indexer
    agent/cmd.go                 — подкоманда agent
  tools/
    rag/rag.go                   — инструменты rag_save, rag_search; подключение к Qdrant
    web/fetch.go                 — инструмент web_fetch; загрузка и очистка HTML
  agent/
    indexer/indexer.go           — агент-индексатор
    chat/agent.go                — чат-агент, REPL, tool-use loop
    summarizer/summarizer.go     — агент сжатия истории
```

### Поток данных: индексация файла

```
CLI --> IndexFile(filename)
         --> os.ReadFile(filename)
         --> chunk.Split(content, cfg)
               --> splitParagraphs   — границы абзацев и Markdown-заголовков
               --> greedy pack       — жадная упаковка до MaxSize рун
               --> forceSplit        — для абзацев длиннее MaxSize
         --> for each chunk:
               --> rag.RagSave(chunk)
                     --> contentUUID(chunk) -> UUID v5 (детерминированный)
                     --> embed(chunk)        -> []float32
                     --> qdrant.Upsert(id, vector, payload{text: chunk})
```

### Поток данных: индексация URL

```
CLI --> IndexURL(url)
         --> web.WebFetch(url)       — загрузка + очистка HTML -> Markdown
         --> chunk.Split(content, cfg)
         --> (далее как для файла)
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
                           --> embed(query) -> []float32
                           --> qdrant.Query(vector, limit=N, score_threshold=T)
                           --> return top-N texts
                     --> [если rag_search вернул результаты]
                           --> LLM формирует ответ, web_fetch не вызывается
                     --> [если rag_search пустой]
                           --> tool_call: web_fetch(url) -> Markdown
                           --> LLM формирует ответ
               --> вывод ответа в stdout
```

## Разбивка на чанки (`internal/chunk`)

Пакет `internal/chunk` реализует разбивку, оптимизированную для RAG:

- **Границы абзацев** — разбивка по пустым строкам (`\n\n`).
- **Markdown-заголовки** — каждый ATX-заголовок (`# … ######`) начинает новый чанк, чтобы заголовок раздела оставался вместе со своим содержимым.
- **Выравнивание по границам слов** — точки разбивки и перекрытия выравниваются по границам слов; слова никогда не обрезаются посередине.
- **UTF-8 safe** — весь учёт размеров ведётся в Unicode-рунах, а не байтах. Кириллица, CJK, эмодзи работают корректно.
- **Нормализация переносов строк** — `\r\n` и `\r` приводятся к `\n` перед обработкой.

## Логирование

Все логи пишутся в `stderr` в формате `slog` text. Уровни:

| Уровень | Когда |
|---|---|
| `DEBUG` | Детали вызовов инструментов (имя, аргументы); score и превью результатов поиска; число обрезанных/сжатых сообщений |
| `INFO` | Начало/конец индексации файла, создание коллекции, запуск сжатия истории |
| `WARN` | Перезапись существующего чанка в Qdrant |
| `ERROR` | Ошибки чтения файлов, сбои embedding/Qdrant, ошибки LLM; сбой сжатия (не фатальный) |

Для подробного вывода установить `log.level: debug` в конфиге.

## Зависимости

| Пакет | Назначение |
|---|---|
| `github.com/alecthomas/kong` | CLI-парсер |
| `github.com/sashabaranov/go-openai` | LLM и embedding клиент |
| `github.com/qdrant/go-client` | Клиент Qdrant (gRPC) |
| `github.com/google/uuid` | UUID v5 для детерминированных ID чанков |
| `github.com/knadh/koanf/v2` | Загрузка YAML-конфига |
| `github.com/lmittmann/tint` | Цветной slog handler |
| `github.com/charmbracelet/glamour` | Рендеринг Markdown в терминале |
| `log/slog` | Структурированное логирование (stdlib) |
