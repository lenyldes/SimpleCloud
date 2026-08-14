## Why

Фаза 7 (устранение аудита `BUGS.md`) почти завершена: из 21 finding остался один — L5 (`Content-Disposition` без `filename*` по RFC 5987), плюс три отложенных пункта в `ROADMAP.md → Technical Debt & Backlog`: покрытие `internal/database` ниже обязательного порога 85% (71.4%), шрифт Inter, блокируемый CSP `style-src 'self'`, и тот же L5. Это последний долг Фазы 7 — закрываем его одним изменением, чтобы реестр багов и бэклог остались чистыми.

## What Changes

- **O2**: рефакторинг `internal/database` — извлечь `RunMigrationsFS(ctx, pool, fsys fs.FS)`; `RunMigrations` остаётся тонкой обёрткой над `embed.FS`. Новые unit-тесты на `fstest.MapFS` (отсутствие каталога, ошибка чтения, пропуск поддиректорий, пустой каталог) + интеграционные тесты веток с живой БД (skip без Postgres). Цель: statement coverage `internal/database` ≥ 85%.
- **CSP-шрифты**: self-host шрифта Inter (woff2) в ассетах `web-frontend`, `@font-face` в CSS, удаление Google Fonts `<link>` из `src/index.html:7-9`. CSP в `nginx.conf` не меняется — `'self'` уже разрешает локальные стили и шрифты.
- **L5**: заголовок `Content-Disposition` при скачивании файла — `attachment; filename="<ascii-fallback>"; filename*=UTF-8''<percent-encoded>` с санацией имени (защита от `\r\n"` — заголовочных инъекций).
- При архивации: пометить `✅ L5` в `BUGS.md`, отметить три пункта Technical Debt в `ROADMAP.md`, перенести O2 в `IDEAS.md` в архив решённых.

## Capabilities

### New Capabilities

(нет)

### Modified Capabilities

- `database-lifecycle`: исполнение миграций принимает injectable `fs.FS` (тестируемость веток ошибок); требование покрытия пакета ≥85% подтверждается тестами.
- `file-storage`: скачивание файла отдаёт `Content-Disposition` с ASCII-фолбэком и RFC 5987 `filename*` для UTF-8 имён, с санацией имени от заголовочных инъекций.
- `web-frontend`: шрифт Inter доставляется self-hosted (без внешних CDN), страница не генерирует CSP-нарушений по стилям/шрифтам.

## Impact

- **Код:** `services/storage-service/internal/database/database.go`, `services/storage-service/internal/handler/file.go`, `services/web-frontend/src/index.html`, `services/web-frontend/src/*.css`, новые файлы шрифтов `services/web-frontend/src/assets/fonts/`.
- **Тесты:** новые/расширенные `*_test.go` в `internal/database` и `internal/handler`; покрытие `internal/database` ≥85% в CI (Postgres-сервис доступен в CI).
- **Документация:** `BUGS.md` (✅ L5), `ROADMAP.md` (Technical Debt −3), `IDEAS.md` (локально, не коммитится).
- **API/зависимости:** публичное поведение API не меняется, кроме формата заголовка `Content-Disposition` (обратно совместимо); внешних сетевых зависимостей у фронтенда становится меньше.
