# Tasks: phase7-hardening-and-polish

Порядок — по приоритету из `BUGS.md` (H1 → … → H5 последним). TDD: в каждом блоке сначала RED-тесты `[TEST-AGENT]`, затем GREEN-реализация `[CODE-AGENT]`. Критерий готовности каждого блока: `go test -cover ./...` проходит, покрытие `internal/*` ≥ 85%, `gofmt -l .` пуст.

## 1. H1 — HTTP-таймауты и лимиты тела (RED → GREEN)

- [x] 1.1 `[TEST-AGENT]` RED-тесты: конфигурация `http.Server` содержит ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout (через тестируемую фабрику сервера); upload с телом сверх лимита → 413; login/folder-create с JSON > 1MB → 4xx.
- [ ] 1.2 `[CODE-AGENT]` Вынести сборку `http.Server` в отдельную функцию; установить таймауты (10s/30s/5m/60s); обернуть `r.Body` в `http.MaxBytesReader`: upload — квота+1MB (ошибка парсинга multipart → 413), LoginHandler и folder CreateHandler — `1<<20`.

## 2. M2 — Graceful shutdown (RED → GREEN)

- [x] 2.1 `[TEST-AGENT]` RED-тесты: shutdown-логика завершает сервер по сигналу/контексту в пределах таймаута, закрывает dbPool; ошибка старта сервера (не ErrServerClosed) приводит к ненулевому выходу.
- [ ] 2.2 `[CODE-AGENT]` Заменить `http.ListenAndServe` на запуск в горутине + `signal.Notify(os.Interrupt, syscall.SIGTERM)` + `srv.Shutdown(ctx)` (30s) + `dbPool.Close()`.

## 3. H2 — UUID-валидация ID (RED → GREEN)

- [x] 3.1 `[TEST-AGENT]` RED-тесты: download/delete(file)/delete(folder) с не-UUID сегментом → 400 без обращения к ФС; `GetShardedPath` с `../`, `/`, `\` в ID → `ErrInvalidFileID`.
- [ ] 3.2 `[CODE-AGENT]` `uuid.Parse` в `DownloadHandler`, `FileHandler.DeleteHandler`, `folder.DeleteHandler`; защитная проверка разделителей/`filepath.Base` в `GetShardedPath`.

## 4. H3 — Secure cookie и CSRF (RED → GREEN)

- [x] 4.1 `[TEST-AGENT]` RED-тесты: cookie логина имеет `Secure` при `COOKIE_SECURE` unset/true и без него при false; мутирующий запрос с чужим `Origin` → 403; с совпадающим `Origin` (с учётом `X-Forwarded-Host`) → проходит; без `Origin` → проходит.
- [ ] 4.2 `[CODE-AGENT]` Флаг `Secure` по env `COOKIE_SECURE` (дефолт true) в обоих `SetCookie`; CSRF-middleware проверки `Origin` против `X-Forwarded-Host`/`Host` на login/logout/upload/delete file/folders; обновить `.env.example`; добавить `proxy_set_header X-Forwarded-Host $host;` в nginx.conf.

## 5. M4 — Timing-safe login (RED → GREEN)

- [x] 5.1 `[TEST-AGENT]` RED-тест: логин с несуществующим email выполняет bcrypt-сравнение (поведенчески: вызов `CheckPasswordHash` с dummy-хэшем, единая ошибка invalid credentials).
- [ ] 5.2 `[CODE-AGENT]` Предвычисленный dummy bcrypt-хэш в `internal/auth`; при user-not-found выполнить сравнение перед возвратом ошибки.

## 6. M6 — Cookie TTL = sessionDuration (RED → GREEN)

- [x] 6.1 `[TEST-AGENT]` RED-тест: `Expires` cookie соответствует TTL, переданному в handler (например 1h ≠ 24h).
- [ ] 6.2 `[CODE-AGENT]` Проброс sessionDuration в `AuthHandler` (конструктор или метод сервиса), `Expires = now + ttl`; в `main.go` единая константа.

## 7. M1 — DSN через url.URL (RED → GREEN)

- [x] 7.1 `[TEST-AGENT]` RED-тесты функции построения DSN: пароль с `@:/ %` корректно экранируется; host/port/db/sslmode сохраняются.
- [ ] 7.2 `[CODE-AGENT]` Вынести построение DSN в функцию (`url.URL` + `url.UserPassword`), заменить конкатенацию в `main.go`.

## 8. M3 — Журнал миграций (RED → GREEN)

- [x] 8.1 `[TEST-AGENT]` RED-тесты (live Postgres, как в существующих db-тестах): создаётся таблица `schema_migrations`; повторный `RunMigrations` не исполняет тела (проверка по журналу/счётчику); сбой миграции → откат транзакции, версия не записана.
- [ ] 8.2 `[CODE-AGENT]` Bootstrap `schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ)`; каждый файл — в транзакции с записью версии, исполнение только при отсутствии версии.

## 9. M5 + L1 + L2 + L3 — фронтенд и CSP

- [ ] 9.1 `[CODE-AGENT]` Убрать `'unsafe-inline'` из `script-src` в nginx.conf; вынести инлайн-обработчики/скрипты из `index.html` в `app.js` (если есть).
- [ ] 9.2 `[CODE-AGENT]` Удалить `formData.append('path', state.currentPath)` (L1); добавить `fileUploadInput.value = ''` после запуска загрузки (L2).
- [x] 9.3 `[TEST-AGENT]` RED-тест бэкенда: `/api/v1/auth/me` возвращает `used_bytes` и `quota_bytes` всегда, включая `used_bytes = 0`.
- [ ] 9.4 `[CODE-AGENT]` Убрать `omitempty` с `QuotaBytes`/`UsedBytes` в `auth.User` (L3, сервер).
- [ ] 9.5 `[CODE-AGENT]` Индикатор квоты в `app.js` рисуется из `used_bytes`/`quota_bytes` ответа `/auth/me` (с обновлением после загрузок/удалений), без суммирования списка файлов (L3, клиент).

## 10. L4 — Dockerfile go.sum

- [ ] 10.1 `[CODE-AGENT]` `COPY go.mod go.sum ./` до `RUN go mod download` в `services/storage-service/Dockerfile`; проверить сборку образа.

## 11. H5 — Non-root контейнеры (ПОСЛЕДНИЙ шаг)

- [ ] 11.1 `[CODE-AGENT]` storage-service Dockerfile: создать пользователя `app`, `chown` директории хранения, `USER app`; проверить, что сервис пишет в том под app.
- [ ] 11.2 `[CODE-AGENT]` web-frontend: базовый образ `nginxinc/nginx-unprivileged:alpine`, `listen 8080;` в nginx.conf, маппинг `32214:8080` в compose.
- [ ] 11.3 `[AUDIT-AGENT]/деплой` Выполнить чеклист из design.md D12 на pi5server: однократный `chown -R` тома `./data/storage` на UID/GID пользователя контейнера, `docker compose up -d --build`, проверка login→upload→download→delete и перезапуска; зафиксировать в отчёте аудита.

## 12. Fail-fast CI (IDEAS)

- [ ] 12.1 `[CODE-AGENT]` В `.github/workflows/ci.yml` добавить `needs: [lint]` джобе `test`; убедиться, что `deploy` (уже `needs: [lint, test]`) не стартует при падении любой зависимости.
- [ ] 12.2 `[AUDIT-AGENT]` Проверить по `gh run list`/`gh run view`, что пайплайн ведёт себя fail-fast (упавший lint/test → deploy skipped).

## 13. Завершение

- [ ] 13.1 `[CODE-AGENT]`/`[TEST-AGENT]` Итоговый прогон `gofmt -l .` (пусто), `go test -v -cover ./...` (все пакеты, ≥85% в `internal/*`), `docker compose up` смоук-тест.
- [ ] 13.2 `[AUDIT-AGENT]` Полный аудит-прогон по протоколу (все изменённые `*.go`/`*_test.go`, покрытие, CI/CD статус, проверка приёмки из BUGS.md).
