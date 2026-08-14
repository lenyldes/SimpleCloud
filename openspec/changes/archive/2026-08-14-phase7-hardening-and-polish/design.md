# Design: phase7-hardening-and-polish

## Context

Остаток аудит-фиксов `BUGS.md` (H1–H3, H5, M1–M6, L1–L4) плюс fail-fast CI из `IDEAS.md`. Мотивация — в proposal.md. Текущее состояние: `cmd/main.go` использует `http.ListenAndServe` без таймаутов и без обработки сигналов; DSN собирается конкатенацией; миграции исполняются целиком при каждом старте; cookie без `Secure` с захардкоженным TTL 24ч; оба контейнера работают от root; CI-джобы `lint` и `test` независимы, `deploy` имеет `needs: [lint, test]`.

## Goals / Non-Goals

**Goals:**
- Закрыть все перечисленные findings без изменения публичного API (кроме гарантированных `used_bytes`/`quota_bytes` в `/auth/me`).
- Сохранить TDD-поток: сначала RED-тесты `[TEST-AGENT]`, затем GREEN `[CODE-AGENT]`.
- Покрыть тестами поведение, проверяемое без поднятия продакшн-инфраструктуры (хэндлеры, middleware, sharding, миграционный журнал через live Postgres в CI).

**Non-Goals:**
- Логирование/структурные логи, O1 (изоляция тестовой БД), O2 (покрытие `internal/database`), L5 (`filename*`), изменение rate-limit-конфигурации nginx.

## Decisions

### D1. Таймауты и лимиты тела (H1)
`http.Server{ReadHeaderTimeout: 10s, ReadTimeout: 30s, WriteTimeout: 5m, IdleTimeout: 60s}`. WriteTimeout 5m — под загрузки до 100M. `MaxBytesReader`: upload — `quota + 1MB` на multipart-overhead (ошибка → 413), login/folder-create — `1<<20` (ошибка → 400). Альтернатива: middleware-обёртка на все роуты — отвергнута, лимиты различаются по эндпоинтам.

### D2. Graceful shutdown (M2)
Стандартная схема: `srv.ListenAndServe()` в горутине, `signal.Notify` на `SIGINT/SIGTERM`, `srv.Shutdown(ctx)` с таймаутом 30с, затем `dbPool.Close()`. Ошибка `ListenAndServe` (кроме `http.ErrServerClosed`) → fatal. Тестируется извлекаемым `run(ctx)/shutdown`-каркасом или интеграционно; минимум — unit-проверка конфигурации сервера и обработчика сигналов в отдельном тестируемом виде (например, вынести сборку `http.Server` и shutdown-логику в функции, вызываемые из теста).

### D3. UUID-валидация (H2)
`uuid.Parse` сразу после извлечения ID в `DownloadHandler`, `FileHandler.DeleteHandler`, `folder.DeleteHandler` → 400. В `GetShardedPath` — defense-in-depth: `strings.ContainsAny(fileID, "/\\") || filepath.Base(fileID) != fileID` → `ErrInvalidFileID`. Существующий length-check остаётся.

### D4. Secure cookie + CSRF (H3)
- `Secure: true` по умолчанию; env `COOKIE_SECURE=false` отключает (локальная разработка). Обновить `.env.example`.
- CSRF: middleware `RequireSameOrigin` на мутирующих эндпоинтах (login, logout, upload, DELETE file, POST/DELETE folders): при наличии `Origin` — сравнить `url.Parse(Origin).Host` с `X-Forwarded-Host` (фолбэк `r.Host`); несовпадение → 403. Без `Origin` — пропускаем (curl/API-клиенты); `SameSite=Lax` остаётся вторым рубежом. Nginx уже прокидывает `Host $host`; добавить `proxy_set_header X-Forwarded-Host $host;` в оба proxy-блока nginx.conf.

### D5. Timing-safe login (M4)
Предвычисленный `dummyHash` (bcrypt константа в пакете `auth`); при `user not found` выполнить `CheckPasswordHash(password, dummyHash)` перед возвратом `ErrInvalidCredentials`. Альтернатива (hmac timing-equalize) избыточна.

### D6. Cookie TTL = sessionDuration (M6)
Пробросить `sessionDuration` в `AuthHandler` через конструктор (`NewAuthHandler(svc, sessionTTL)` либо метод `SessionTTL()` на сервисе). `Expires = time.Now().Add(ttl)`. В `main.go` TTL = та же константа 24h, что у `NewDBAuthService`.

### D7. DSN через url.URL (M1)
`url.URL{Scheme:"postgres", User:url.UserPassword(dbUser, dbPass), Host:dbHost+":"+dbPort, Path:dbName}` + `q.Set("sslmode","disable")`. `sslmode` не меняем (изолированная docker-сеть; отмечено в BUGS.md как допустимое).

### D8. Журнал миграций (M3)
`RunMigrations`: bootstrap `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now())` вне журнала. Для каждого файла: в одной транзакции `INSERT ... ON CONFLICT DO NOTHING`-проверка (или SELECT-then-INSERT), при новом version — исполнение тела миграции и commit. Версия = имя файла. Существующие миграции (`000001_init_schema.sql`, последующие) идемпотентны и останутся валидными; на уже развёрнутой БД первый старт с журналом запишет все текущие версии без переисполнения тел — реализовать как «если таблица создана впервые и схемы объектов уже нет» не требуется: проще исполнить тела один раз при переходе, они идемпотентны (`IF NOT EXISTS`). Решение: при первом запуске журнала исполняем все файлы в транзакциях и пишем версии — идемпотентность SQL гарантирует безопасность.

### D9. CSP без unsafe-inline (M5)
Убрать `'unsafe-inline'` из `script-src` (style-src оставляем — стили могут быть инлайн). Проверить `index.html` на inline-обработчики/скрипты и вынести в `app.js`. Проверка — вручную в браузере (консоль без CSP-ошибок) на этапе аудита.

### D10. Фронтенд-фиксы (L1, L2, L3)
- L1: удалить `else`-ветку с `formData.append('path', ...)`.
- L2: `fileUploadInput.value = ''` сразу после запуска `handleFileUpload`.
- L3: убрать `omitempty` с `QuotaBytes/UsedBytes` в `auth.User` (иначе `used_bytes:0` отсутствует в JSON — нарушает спеку), индикатор в `app.js` брать из ответа `/auth/me` (`state.quota` обновлять после login/me и после каждой загрузки/удаления).

### D11. Dockerfile go.sum (L4)
`COPY go.mod go.sum ./` до `RUN go mod download`.

### D12. Non-root контейнеры (H5) — ПОСЛЕДНИЙ шаг
- storage-service: `RUN addgroup -S app && adduser -S app -G app`, `mkdir -p /storage && chown app:app /storage`, `USER app`. `STORAGE_DIR` в compose уже маппит `./data/storage` — проверить значение env внутри контейнера на `/storage` или оставить текущий путь с chown.
- web-frontend: базовый образ `nginxinc/nginx-unprivileged:alpine` (listen 8080 от пользователя nginx) → `nginx.conf`: `listen 8080;`, compose: маппинг `32214:8080`.
- **Деплой-чеклист (выполнить на pi5server однократно, зафиксировать в отчёте аудита):**
  1. `cd /opt/simplecloud` (или `~/SimpleCloud`).
  2. `docker compose down`.
  3. `UID=$(id -u <deploy-user>)` — определить UID пользователя контейнера (для alpine app обычно 100/1000; проверить `docker run --rm image id app`).
  4. `sudo chown -R <uid>:<gid> ./data/storage`.
  5. `docker compose up -d --build`.
  6. Проверка: `docker compose ps` (все up), login → upload → download → delete через UI, `ls -ln ./data/storage` — файлы создаются новым владельцем, рестарт контейнера не ломает права.

### D13. Fail-fast CI (IDEAS)
В `ci.yml`: добавить `needs: [lint]` джобе `test` (сейчас независимы) — тогда падение lint скипает test; `deploy` уже имеет `needs: [lint, test]` (GitHub Actions автоматически не запускает джобы при падении зависимостей). Дополнительно шаг деплоя остаётся только для `main`. Деплой при упавших тестах становится невозможным.

## Risks / Trade-offs

- [H5: существующие файлы тома принадлежат root → контейнер от app не может их читать/удалять] → обязательный `chown -R` по чеклисту D12 до/во время деплоя; проверка аудита.
- [`Secure`-cookie ломает локальную разработку по HTTP] → `COOKIE_SECURE=false` в `.env.example`, дефолт true.
- [CSRF-проверка по `Origin` может резать запросы за прокси без `X-Forwarded-Host`] → добавить прокидку заголовка в nginx.conf; при отсутствии `Origin` запрос пропускается.
- [Удаление `omitempty` с `used_bytes` меняет JSON-ответы] → изменение обратно-совместимое (поле добавляется); фронтенд обновляется одновременно.
- [nginx-unprivileged меняет внутренний порт] → обновить compose-маппинг и healthcheck-ожидания; Caddy смотрит на host-порт 32214, не затрагивается.
- [Журнал миграций на уже живых БД] → идемпотентные `IF NOT EXISTS`-миграции безопасны при первом «прогоне» журнала; транзакционность гарантирует откат при сбое.

## Migration Plan

1. Блоки 1–13 (H1…L4) деплоятся обычным push → CI → автодеплой; не требуют ручных операций на сервере.
2. Блок H5 — отдельный финальный деплой с ручным `chown` на `pi5server` (чеклист D12) и проверкой перезапуска.
3. Rollback: `git revert` + redeploy; для H5 — вернуть root-образы и маппинг `32214:80` (права тома при этом остаются совместимыми с root).

## Open Questions

- Точный UID/GID пользователя `app` в alpine-образе (определить при реализации и зафиксировать в чеклисте деплоя).
