# TODO

## Интеграция GitHub API с web_fetch и инструментом `github_repo`

- [ ] Создать пакет `internal/tools/github`
  - По аналогии с `internal/tools/gitlab`
  - Структура: `github.go` (основная логика), `github_test.go` (тесты), `types.go` (для структур ответов)
- [ ] Реализовать обработку GitHub URL (`https://github.com/{owner}/{repo}`)
  - Парсинг owner и repo из URL
  - Приведение к `https://api.github.com/repos/{owner}/{repo}`
- [ ] Реализовать HTTP-запрос с нужными заголовками:
  - `Authorization: Bearer {github.access_key}`
  - `Accept: application/vnd.github+json`
  - `X-GitHub-Api-Version: 2026-03-10`
- [ ] Добавить второй запрос к `https://api.github.com/repos/{owner}/{repo}/readme`
  - Использовать те же заголовки и параметры
- [ ] Реализовать объединение двух ответов (repo + readme) в единый Markdown со следующими полями:
  - `full_name`
  - `default_branch`
  - `license`
  - `language`
  - `forks_count`
  - `archived`
  - `stargazers_count`
  - `open_issues_count`
  - `topics`
  - `description`
  - `readme`
- [ ] Добавить конфигурационные параметры (`configs/config.yaml` и `internal/config`):
  - `github.base_url` (default: `https://api.github.com`)
  - `github.access_key` (required)
  - `github.disable` (default: `false`)
- [ ] Обработать ошибки API (401, 403, 404 и т.д.)
- [ ] Декодировать и структурировать ответы GitHub API (в том числе license.name, license.spdx_id и т.п.)
- [ ] Поддержка расширенных URL: pull requests, issues, commits, файлов и т.д. (если потребуется расширение URL-паттернов)
- [ ] Настроить тесты для GitHub API endpoint
- [ ] Обновить документацию (конфигурация, использование)
- [ ] Интеграция с существующим `web_fetch`: распознавание `github.com/{owner}/{repo}` и вызов `github.fetch()`

## Инструмент `github_repo`

- [ ] Реализовать LLM-инструмент `github_repo`, принимающий:
  - Вход: `repo` — строка в формате `{owner}/{repo}` или `github.com/{owner}/{repo}`
  - Выход: единый Markdown (тот же, что в `web_fetch`)
- [ ] Парсинг `repo`-строки для извлечения owner и repo
  - Обработка как `owner/repo`, так и `github.com/owner/repo`
  - Валидация формата и обработка ошибок
- [ ] Использовать уже реализованную функцию `github.fetch(owner, repo)` для получения данных
- [ ] Зарегистрировать инструмент через `hub.RegisterTool(...)`
- [ ] Настроить тесты инструмента
- [ ] Обновить документацию (описание инструмента)

## Интеграция GitHub API с web_fetch для файлов и инструмент `github_file`

- [ ] Расширить обработку `web_fetch` для URL вида `https://github.com/{owner}/{repo}/blob/{branch}/{file}`
  - Извлечь `owner`, `repo`, `branch` и `file` из URL
  - Сформировать GitHub API endpoint: `https://api.github.com/repos/{owner}/{repo}/contents/{file}?ref={branch_name}`
  - Выполнить запрос с теми же заголовками (`Authorization`, `Accept`, `X-GitHub-Api-Version`)
  - Декодировать ответ — получить base64-encoded содержимое файла
  - Раскодировать base64 и вернуть как Markdown через `web_fetch`
- [ ] Реализовать LLM-инструмент `github_file`, принимающий:
  - `repo` — строка в формате `{owner}/{repo}` или `github.com/{owner}/{repo}`
  - `branch` — имя ветки (по умолчанию: `main` или `master`)
  - `file` — путь к файлу в репозитории
  - Выход: содержимое файла (decoded Markdown / plain text)
- [ ] Парсинг `repo`-строки (как для `github_repo`)
- [ ] Зарегистрировать инструмент через `hub.RegisterTool(...)`
- [ ] Настроить тесты для инструмента
- [ ] Обновить документацию

## Инструмент `github_tree`

- [ ] Реализовать LLM-инструмент `github_tree`, принимающий:
  - `repo` — строка в формате `{owner}/{repo}` или `github.com/{owner}/{repo}`
  - `branch` — имя ветки (по умолчанию: `main` или `master`)
  - Выход: список файлов и папок в корне репозитория в удобном формате (tree-like или Markdown table)
- [ ] Использовать GitHub API endpoint: `https://api.github.com/repos/{owner}/{repo}/contents?ref={branch}`
- [ ] Парсинг `repo`-строки (как для `github_repo` и `github_file`)
- [ ] Зарегистрировать инструмент через `hub.RegisterTool(...)`
- [ ] Настроить тесты для инструмента
- [ ] Обновить документацию

## Конфигурация

- [ ] Обновить `configs/config.yaml` — добавить блок `github` с параметрами:
  - `github.base_url` (default: `https://api.github.com`)
  - `github.access_key` (required)
  - `github.disable` (default: `false`)

## Документация

- [ ] Обновить `AGENTS.md` — добавить раздел про GitHub API и инструменты (`github_repo`, `github_file`, `github_tree`)
- [ ] Обновить `README.md` — добавить примеры использования GitHub инструментов
- [ ] Обновить `docs/user_manual.md` — добавить описание работы с GitHub репозиториями

