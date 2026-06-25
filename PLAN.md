# План миграции: объединение CLI и чат-режима

## Цель
Заменить два отдельных CLI-команды (`go-magnetar agent`, `go-magnetar indexer`) на одну (`go-magnetar`) с чат-режимом по умолчанию и командой `/index` для индексации.

## Архитектурные изменения

### 1. Упрощение CLI
**Файл**: `internal/cmd/cmd.go`

**Текущее состояние**:
- `cli` структура использует `kong.Plugins` для динамического добавления подкоманд
- Две независимые подкоманды: `agent` и `indexer`

**Планируемое состояние**:
- Удалить `kong.Plugins` из `cli` структуры
- Удалить вызов `plugin.KongPlugins()` и `RegisterCLI`
- CLI больше не будет поддерживать динамические подкоманды
- Вместо этого комманды будут регистрироваться как чат-команды через `hub.RegisterChatCommand()`

**Изменения в cmd.go** (строгое):
```go
type cli struct {
    Globals
}

func Execute() error {
    root := &cli{}
    ctx := kong.Parse(root,
        kong.Name("go-magnetar"),
        kong.Description("RAG indexer and AI chat agent"),
        kong.UsageOnError(),
        kong.Bind(&root.Globals),
    )

    cfg, err := config.Load(root.Globals.Config)
    if err != nil {
        return err
    }

    config.SetupLogger(cfg)

    // Запуск чат-режима напрямую (без subcommand dispatch)
    workDir, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("get work dir error: %w", err)
    }

    rootFS, err := os.OpenRoot(workDir)
    if err != nil {
        return fmt.Errorf("open work dir error: %w", err)
    }
    defer rootFS.Close()

    agent, err := chat.New(cfg, rootFS)
    if err != nil {
        return err
    }

    return agent.Run()
}
```

### 2. Удаление старых CLI-плагинов
**Файлы**: `internal/plugins/cli/agent/plugin.go`, `internal/plugins/cli/indexer/plugin.go`

**Изменения**:
- Удалить `internal/plugins/cli/indexer/plugin.go` полностью
- Удалить `internal/plugins/cli/agent/plugin.go` полностью
- Удалить соответствующие импорты из `cmd/go-magnetar/main.go`

### 3. Перенос логики индексатора
**Файл**: `internal/cmd/indexer/cmd.go`

**Изменения**:
- Удалить `Run()` метод отсюда (или оставить для backward compatibility на время миграции)
- Перенести логику `IndexFile()` и `IndexURL()` в новый чат-плагин

### 4. Создание indexcmd plugin
**Файл**: `internal/plugins/indexcmd/plugin.go` (новый файл)

**Обязательные задачи**:
1. Создать чат-команду `/index` с алиасами `/i`
2. Автоопределение типа:
   - URL с `http://` или `https://` → `IndexURL()`
   - Любой другой путь → `IndexFile()`
3. Поддержка флага `-m <message>`
4. Валидация аргументов
5. Обработка ошибок
6. Возврат результата в чат

**Требуемая функциональность**:
- Доступ к `Indexer` из `internal/agent/indexer`
- Автоматический выбор метода на основании префикса URL
- Показ пользователю статуса операции (успех/ошибка)

**Пример кода**:
```go
package indexcmd

import (
    "context"
    "fmt"
    "strings"

    "github.com/wmentor/go-magnetar/internal/agent/indexer"
    "github.com/wmentor/go-magnetar/internal/plugin"
    "github.com/wmentor/go-magnetar/internal/tools/rag"
    "github.com/wmentor/go-magnetar/internal/tools/web"
)

func init() {
    plugin.Register("indexcmd", &Plugin{})
}

type Plugin struct {
    idx *indexer.Indexer
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
    ragTools, err := rag.New(s.Config)
    if err != nil {
        return fmt.Errorf("indexcmd: failed to init RAG: %w", err)
    }

    webTools, err := web.New(s.Config, s.Root)
    if err != nil {
        return fmt.Errorf("indexcmd: failed to init web: %w", err)
    }

    p.idx, err = indexer.New(s.Config, s.Root)
    if err != nil {
        return fmt.Errorf("indexcmd: failed to create indexer: %w", err)
    }

    hub.RegisterChatCommand(plugin.ChatCommand{
        Name:    "index",
        Aliases: []string{"i"},
        Help:    "Index file or URL into RAG knowledge base (auto-detects URL vs file)",
        Execute: p.execute,
    })

    return nil
}

func (p *Plugin) execute(ctx context.Context, agent plugin.AgentHandle, args string) error {
    if args == "" {
        return fmt.Errorf("usage: /index <path|url> [-m <message>]")
    }

    target := strings.TrimSpace(args)
    message := ""

    // Parse optional -m flag
    parts := strings.Fields(target)
    target = parts[0]
    for i := 1; i < len(parts); i++ {
        if parts[i] == "-m" && i+1 < len(parts) {
            message = parts[i+1]
            i++
        }
    }

    // Auto-detect URL vs file based on scheme
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        return p.idx.IndexURL(target, message)
    }
    
    // Default to file for anything else (relative/absolute paths)
    return p.idx.IndexFile(target, message)
}
```

### 5. Миграция инициализации
**Файлы**: `cmd/go-magnetar/main.go`

**Изменения**:
```go
// Удалить:
_ "github.com/wmentor/go-magnetar/internal/plugins/cli/indexer"
_ "github.com/wmentor/go-magnetar/internal/plugins/cli/agent"

// Добавить:
_ "github.com/wmentor/go-magnetar/internal/plugins/indexcmd"
```

### 6. Обновление документации
**Файлы**: `AGENTS.md`, `README.md`, `configs/config.yaml` (комментарии)

**Требуемые обновления**:
- Удалить раздел про `go-magnetar agent` и `go-magnetar indexer`
- Обновить раздел про команды CLI до одной команды `go-magnetar`
- Добавить документацию по `/index` команде
- Обновить примеры в документации

## Шаги реализации

### Этап 1: Подготовка
1. Создать резервную копию репозитория
2. Убедиться, что все изменения закоммичены и ветка чистая
3. Прочитать текущую структуру `internal/cmd/cmd.go` и `cmd/go-magnetar/main.go`

### Этап 2: Обновление cmd.go
1. Удалить `kong.Plugins` из `cli` структуры
2. Изменить `Execute()` чтобы вызывать `chat.New()` напрямую (см. раздел 1)
3. Убрать вызов `plugin.KongPlugins()`
4. Удалить импорт устаревших плагинов из `main.go`
5. Протестировать, что `go-magnetar` запускает чат (без subcommands)
6. Убедиться, что можно остановить чат через `/exit` или `Ctrl+D`

### Этап 3: Создание нового плагина
1. Создать `internal/plugins/indexcmd/plugin.go`
2. Реализовать чат-команду `/index` (см. раздел 4)
3. Проверить компиляцию: `make tidy && make lint`
4. Убедиться, что команда `/index` появляется в `/help`

### Этап 4: Удаление старой CLI-логики
1. Удалить `internal/plugins/cli/indexer/plugin.go`
2. Удалить `internal/cmd/indexer/cmd.go`
3. Удалить `internal/plugins/cli/agent/plugin.go`
4. Обновить `cmd/go-magnetar/main.go` (удалить импорты старых CLI plugin'ов)
5. Проверить компиляцию again
6. Убедиться, что старые команды `go-magnetar agent` и `go-magnetar indexer` больше не работают

### Этап 5: Интеграция и тестирование
1. Обновить `cmd/go-magnetar/main.go` (добавить импорт new indexcmd)
2. Протестировать обычный чат-режим (без подкоманд)
3. Протестировать `/index <path>` (локальный файл)
4. Протестировать `/index <url>` с http:// или https:// (auto-detect)
5. Протестировать `/index` с флагом `-m`
6. Протестировать обработку ошибок:
   - `/index` без аргументов
   - `/index nonexistent_file.txt`
   - `/index http://nonexistent.example.com`
7. Протестировать `/help` (показывает команду `/index`)

### Этап 6: Документирование
1. Обновить `PLAN.md` (если изменения в процессе)
2. Обновить `README.md`
3. Обновить `AGENTS.md`
4. Добавить примеры использования новой команды

## Риски и меры по их снижению

### Риск 1: Потеря функциональности
**Мера**: Пошаговая миграция с тестированием на каждом этапе

### Риск 2: Нарушение совместимости
**Мера**: Миграция производится один раз; backward compatibility не сохраняется для старых subcommand'ов

### Риск 3: Ошибки в логике индексации
**Мера**: Повторное использование существующего кода из `internal/agent/indexer`

## Проверка завершения

Когда все эти шаги выполнены, проверьте:

- [ ] `go-magnetar` запускается без ошибок и запускает чат-режим
- [ ] Команды `go-magnetar agent` и `go-magnetar indexer` больше не работают
- [ ] `/help` показывает `/index` команду
- [ ] `/index <path>` (локальный файл) работает корректно
- [ ] `/index <url>` с http:// или https:// работает корректно (auto-detect)
- [ ] `/index` с флагом `-m` работает корректно
- [ ] Ошибки обрабатываются правильно (правильные сообщения об ошибках)
- [ ] `make lint` и `make tidy` проходят без ошибок
- [ ] Документация обновлена
- [ ] No broken imports or references to old CLI subcommands
