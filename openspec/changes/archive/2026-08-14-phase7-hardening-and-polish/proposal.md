# Proposal: phase7-hardening-and-polish

## Why

Аудит (`BUGS.md`, 2026-08-14) выявил остаточные дефекты после блоков 1–2 Фазы 7: отсутствие таймаутов и лимитов тела HTTP (Slowloris/DoS), нет graceful shutdown, слабая валидация ID в URL-путях, cookie без `Secure` и без CSRF-защиты, timing-перечисление пользователей, миграции без журнала версий, root-контейнеры и ряд UX/сборочных проблем. Это последний блок аудит-фиксов перед ручным тестированием и релизом v1.0.

## What Changes

1. **H1** — таймауты `http.Server` (ReadHeader/Read/Write/Idle) и `http.MaxBytesReader` на upload и JSON-хэндлерах (login, folder create); превышение → 413.
2. **M2** — graceful shutdown: обработка SIGTERM/SIGINT, `srv.Shutdown` с 30-секундным таймаутом, закрытие `dbPool`.
3. **H2** — валидация file/folder ID как UUID в `DownloadHandler`/`DeleteHandler` (file и folder) + защитная проверка в `GetShardedPath` (path traversal defense-in-depth).
4. **H3** — флаг `Secure` на сессионной cookie (управляется env `COOKIE_SECURE`, по умолчанию true) + CSRF-проверка заголовка `Origin` против `X-Forwarded-Host`/`Host` на всех мутирующих эндпоинтах (403 при несовпадении).
5. **M4** — timing-safe login: сравнение с dummy bcrypt-хэшем при неизвестном email.
6. **M6** — срок жизни cookie привязывается к `sessionDuration` сервиса вместо захардкоженных 24ч.
7. **M1** — DSN к PostgreSQL строится через `url.URL` + `url.Password` (корректное экранирование спецсимволов пароля).
8. **M3** — журнал миграций: таблица `schema_migrations(version, applied_at)`, каждый файл исполняется один раз и внутри транзакции.
9. **M5** — убрать `'unsafe-inline'` из `script-src` CSP в `nginx.conf`; инлайн-обработчики (если есть) вынести в `app.js`.
10. **L1** — удалить `formData.append('path', state.currentPath)` (поле `undefined`) в `app.js`.
11. **L2** — сброс `fileUploadInput.value` после запуска загрузки (повторная загрузка того же файла).
12. **L3** — `/api/v1/auth/me` гарантированно отдаёт `used_bytes`/`quota_bytes`; фронтенд рисует индикатор квоты из них, а не суммированием списка файлов.
13. **L4** — `COPY go.mod go.sum ./` до `go mod download` в `services/storage-service/Dockerfile`.
14. **H5** — non-root контейнеры: `USER app` в storage-service, `nginxinc/nginx-unprivileged` (listen 8080) для web-frontend; обновить compose-маппинг портов. **ПОСЛЕДНИЙ шаг** с деплой-чеклистом: однократный `chown -R` тома `./data/storage` на `pi5server` и проверка перезапуска.
15. **Fail-fast CI/CD** (из `IDEAS.md` 📥 Входящие) — последующие шаги/джобы workflow не выполняются после падения предыдущих (явные `needs`/`if: success()`-семантика в `.github/workflows/ci.yml`).

Вне scope (отложено): логирование, O1 (изоляция тестовой БД), O2 (покрытие `internal/database`), L5 (`filename*`).

## Capabilities

### New Capabilities

- `database-lifecycle`: построение DSN к PostgreSQL и журнал версий SQL-миграций (M1, M3).

### Modified Capabilities

- `security-hardening`: таймауты HTTP-сервера и лимиты тела запроса (H1), ужесточение CSP без `'unsafe-inline'` (M5), непривилегированные контейнеры (H5).
- `auth-multi-tenancy`: `Secure`-cookie и CSRF-проверка Origin (H3), timing-safe login (M4), TTL cookie = sessionDuration (M6), `used_bytes`/`quota_bytes` в `/api/v1/auth/me` (L3, серверная часть).
- `file-storage`: обязательная UUID-валидация ID файла/папки в хэндлерах и в `GetShardedPath` (H2).
- `web-frontend`: удаление мусорного поля `path` (L1), сброс file input (L2), индикатор квоты из данных `/auth/me` (L3, клиентская часть).
- `deployment-ci-cd`: `go.sum` в сборке до `go mod download` (L4), fail-fast порядок шагов/джоб CI (IDEAS fail-fast).
- `project-scaffold`: graceful shutdown жизненного цикла сервиса (M2).

## Impact

- **Код Go:** `services/storage-service/cmd/main.go`, `internal/auth/handler.go`, `internal/auth/service.go`, `internal/auth/middleware.go` (CSRF), `internal/database/database.go` (+ новая миграция `schema_migrations`), `internal/storage/sharding.go`, `internal/handler/file.go`, `internal/handler/folder.go`.
- **Фронтенд:** `services/web-frontend/src/app.js`, `index.html` (инлайн-обработчики при наличии), `nginx.conf`.
- **Сборка/деплой:** `services/storage-service/Dockerfile`, `services/web-frontend/Dockerfile`, `docker-compose.yml` (порт 8080→`nginx-unprivileged`), `.github/workflows/ci.yml`, одноразовая операция `chown` на `pi5server`.
- **Конфигурация:** новая env-переменная `COOKIE_SECURE` (обновить `.env.example`); смена базового образа nginx.
- **BREAKING (операционный):** web-frontend слушает 8080 внутри контейнера вместо 80; после деплоя H5 требуется однократный `chown -R` хост-тома `./data/storage`.
