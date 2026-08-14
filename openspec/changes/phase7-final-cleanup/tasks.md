# Tasks: phase7-final-cleanup

Роли: задачи в разделах «Тесты (RED)» выполняет `[TEST-AGENT]` (только `*_test.go`), разделы «Реализация (GREEN)» — `[CODE-AGENT]` (только production-код, тесты не трогать).

## 1. Тесты (RED) — `[TEST-AGENT]`

- [x] 1.1 `internal/database`: RED-тесты `RunMigrationsFS` на `fstest.MapFS` — отсутствие каталога `migrations` → ошибка; пустой каталог → успех без применения; поддиректория в каталоге → пропускается (не исполняется как миграция); кастомный `fs.FS` с падающим `Open` → ошибка чтения файла (сценарии delta-spec `database-lifecycle`).
- [x] 1.2 `internal/database`: RED интеграционные тесты с живой БД (скип без Postgres) — невалидный SQL в миграции → rollback, запись в журнал не появляется, сервис падает с описательной ошибкой; повторный прогон `RunMigrations` → применённые версии пропускаются.
- [x] 1.3 `internal/handler`: RED-тесты `Content-Disposition` в `DownloadHandler` — кириллическое имя (`отчёт.pdf`) → заголовок содержит ASCII-фолбэк `filename` и `filename*=UTF-8''` с корректным percent-encoding; ASCII-имя (`report.pdf`) → `filename="report.pdf"`; имя с `\r\n"`/кавычками → санация, без инъекции заголовков (сценарии delta-spec `file-storage`).
- [x] 1.4 Закоммитить RED-тесты (`ADD:`), убедиться, что они падают по ожидаемым причинам, обновить этот файл задач.

## 2. Реализация O2 — миграции с `fs.FS` (GREEN) — `[CODE-AGENT]`

- [ ] 2.1 Извлечь `RunMigrationsFS(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error` в `internal/database/database.go` (чтение через `fs.ReadDir`/`fs.ReadFile`); `RunMigrations` — тонкая обёртка над `migrationFS` (design D1).
- [ ] 2.2 Добиться прохождения тестов 1.1/1.2; проверить `go test -cover ./internal/database/` локально, итоговый порог ≥85% подтверждается в CI.

## 3. Реализация L5 — RFC 5987 filename* (GREEN) — `[CODE-AGENT]`

- [ ] 3.1 В `internal/handler/file.go` (DownloadHandler, ~:300) собрать заголовок `attachment; filename="<ascii-fallback>"; filename*=UTF-8''<percent-encoded>`: санация имени от `\r`, `\n`, `"` до форматирования; ASCII-фолбэк (удаление не-ASCII, при пустом результате — `file`); percent-encoding через `url.PathEscape` (design D4).
- [ ] 3.2 Добиться прохождения тестов 1.3.

## 4. CSP-шрифты: self-host Inter — `[CODE-AGENT]`

- [ ] 4.1 Скачать Inter variable woff2 (latin subset, официальный источник rsms/inter или Google Fonts) в `services/web-frontend/src/assets/fonts/inter.woff2`; убедиться, что размер разумный (≤350 КБ) и файл валидный.
- [ ] 4.2 Добавить `@font-face` (font-family 'Inter', font-weight 400 700, font-display swap, путь `/assets/fonts/inter.woff2`) в CSS фронтенда; удалить три Google Fonts `<link>` из `services/web-frontend/src/index.html:7-9`.
- [ ] 4.3 Проверить в браузере: страница рендерится с Inter, консоль без CSP-нарушений по стилям/шрифтам, нет запросов к fonts.googleapis.com/fonts.gstatic.com (nginx отдаёт `/assets/fonts/*` как статику с origin — проверить маппинг ассетов в nginx/Dockerfile при необходимости).

## 5. Верификация и документы

- [ ] 5.1 `[CODE-AGENT]`: `gofmt -l .` пуст, `go test ./...` зелёный в `services/storage-service/`, коммит (`UPD:`/`FIX:`) и push.
- [ ] 5.2 `[AUDIT-AGENT]`: deep code review всех изменённых файлов, `go test -v -cover ./...` с порогом ≥85% для `internal/*` (CI-покрытие `internal/database`), проверка CSP-шрифтов, статус CI/CD (`gh run list --limit 3`).
- [ ] 5.3 `[ORCHESTRATOR-AGENT]` при архивации: `✅ L5` в `BUGS.md` (сводная таблица + заголовок секции), отметить три пункта Technical Debt в `ROADMAP.md`, перенести O2 из обсуждения в архив `IDEAS.md`.
