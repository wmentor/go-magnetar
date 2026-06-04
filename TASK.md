# ЦЕЛЬ

Проект github.com/wmentor/go-magnetar реализует в одной утилите инструмент для добавления документов
в RAG (рабоиу с индексацией тоже должен делать агент) и чат с AI-агентов, где за информацией он ходит в RAG.

# Имя модуля

github.com/wmentor/go-magnetar

# Технологический стэк

- golang 1.26.4
- CLI github.com/alecthomas/kong
- LLM клиент github.com/sashabaranov/go-openai
- RAG Qdrant github.com/qdrant/go-client
- Конфиг в yaml на базе github.com/knadh/koanf
- Логирование log/slog

# Структура команд

## Индексатор

go-magnetar indexer -c config.json -f|--file filename.(md|txt) -d|--directory directory

если задан файл, то индексируется файл, если задана директория, то все файлы md,txt в директориию

## Агент

go-magnetar agent -c config.json

# Конфигурационный файл

## Пример

```yaml
llm:
  base_url: openai_url
  api_key: key
  model: model
  context: 250000  # лимит токенов передаваемого контекста в запросе к LLM
rag:
  llm:
    base_url: openai_url
    api_key: key
    model: embedding_model
    vector_size: 1000
  qdrant:
    connstr: connect_string  # строка подключения к Qdrant по REST, например http://localhost:6333
    collection: collection_name
log:
  level: info  # уровень логирования по умолчанию, допустимые значения: debug, info, warn, error
```

Возможно нужно добавить какие-то еще поля

# Особенности

## Инструменты

### Работа с файловой системой

- file_read(filename) content, bool    # прочитать файл

### Работы с RAG

- rag_save(content) bool сохраняет фрагмент текста в RAG (внутри использует embedding модель для вычисления векторов, ID - sha256 от текста вычисляется через стандартную библиотеку crypto/sha256; если документ с таким ID уже существует в Qdrant — перезаписывается, логируется slog.Warn)
- rag_search(content) result - по переданному запросу возвращает результаты из RAG (под капотом векторизация)

## Indexer

### Доступные инстурменты

Только file_read и rag_save.

### Обработка директории

Если в параметрах командной строки задана директория, то ее нужно прочитать через path/filepath.WalkDir стандартными средствами языка go и потом
для каждого файла txt и md применить обработку файла. При ошибке обработки конкретного файла — логировать через slog.Error с деталями ошибки и продолжать обработку остальных файлов.

### Системный промпт

```
You are a document indexing agent. Your task is to process text files and store their content in a knowledge base.

When given a filename:
1. Read the file using the file_read tool.
2. Split the content into logical blocks, each no more than 500 tokens. Preserve semantic boundaries — split by paragraphs, sections, or logical units, not in the middle of a sentence or idea.
3. Save each block to the RAG using the rag_save tool.
4. Report how many blocks were saved.

Do not summarize, paraphrase, or alter the content. Store the original text as-is.
```

### Обработка файла

Агент передает LLM сообщение о том, что нужно прочитать файл через инструмент file_read, после чего разбить на логические блоки (каждый не более 500 токенов)
и сохранить их все в RAG при помощи инструмента rag_save.

## Чат-агент

Интерактивный REPL: вопрос читается из stdin, ответ выводится в stdout, цикл повторяется до завершения (EOF или команда выхода).

Агент самостоятельно решает когда вызывать `rag_search` — основной критерий: нужна информация или нужно уточнить имеющуюся. Агент не должен выдумывать — если информации недостаточно, он сообщает об этом пользователю.

### Системный промпт

```
You are a helpful assistant that answers questions strictly based on the knowledge base.

Rules:
- If you need information to answer a question, or need to verify or clarify what you already know, use the rag_search tool before responding.
- Base your answers only on the information retrieved from rag_search. Do not invent, assume, or extrapolate facts.
- If rag_search returns no relevant results, tell the user honestly that you don't have information on this topic.
- You may call rag_search multiple times with different queries if needed.
- Be concise and precise.
```

### Доступные инструменты

- rag_search

# Архитектура проекта

Используем стандартный go layout.

## Структура каталогов

```plain
go-magnetar
  bin/
      go-magnetar - собранный бинарник (в .gitignore)
  cmd
    go-magnetar
      main.go
  configs:
      config.yaml - пример конфигурационного файла
  internal
    cmd
      cmd.go - root command
      agent
        cmd.go
      indexer
        cmd.go
    config
      config.go
    tools
      rag
        rag.go - инструменты для работы с RAG, лучше реализовать в виде объекта с методом на каждый инстурмент в New передать объект Config
      generic
        generic.go - базовые операции (file_read), тоже в виде объекта с передачей конфига на этапе New
    agent
      chat
        agent.go - реализация агента для чата (цикл - вопрос ответ)
      indexer
        indexer.go - реализация агента индексатора
  Makefile - базовые команды сборки и разработки
  .gitignore
```

## Makefile

Проект содержит `Makefile` с базовыми командами:

- `make build` — сборка бинарника в `bin/go-magnetar`
- `make clean` — удаление папки `bin/`
- `make run-agent` — запуск агента с дефолтным конфигом (`configs/config.yaml`)
- `make lint` — запуск `go vet ./...`
- `make tidy` — запуск `go mod tidy`


