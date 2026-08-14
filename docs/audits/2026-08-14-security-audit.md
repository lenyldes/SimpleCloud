# SimpleCloud — Отчёт об аудите: баги и уязвимости (Архив от 2026-08-14)

> **Дата аудита:** 2026-08-14  
> **Статус:** ✅ ВСЕ 23 НАХОДКИ ИСПРАВЛЕНЫ И ВЕРИФИЦИРОВАНЫ  
> **Область:** `services/storage-service/` (Go), `services/web-frontend/` (Nginx + Vanilla JS), `docker-compose.yml`, `Dockerfile`, CI/CD.  

---

## Сводная таблица результатов

| ID | Критичность | Заголовок | Где | Статус / Изменение |
|----|-------------|-----------|-----|-------------------|
| ✅ C1 | CRITICAL | Фолбэк на mock-аутентификацию с захардкоженным паролем при недоступности БД | `cmd/main.go:54-57`, `internal/auth/service.go:257-278` | phase7-critical-quick-fixes, 2026-08-14 |
| ✅ C2 | CRITICAL | Метаданные файлов только в памяти: потеря данных при рестарте + IDOR (скачивание чужих файлов) | `internal/handler/file.go:35-49,165-174` | phase7-db-quotas-deletion, 2026-08-14 |
| ✅ C3 | CRITICAL | `/api/v1/auth/me` зарегистрирован без middleware аутентификации — всегда 401 | `cmd/main.go:86` | phase7-critical-quick-fixes, 2026-08-14 |
| ✅ C4 | CRITICAL | Совокупная квота пользователя не контролируется, `used_bytes` не обновляется — исчерпание диска (DoS) | `internal/handler/file.go`, `internal/database/migrations/000001_init_schema.sql` | phase7-db-quotas-deletion, 2026-08-14 |
| ✅ C5 | CRITICAL | Порты `8080` (storage) и `5432` (PostgreSQL) опубликованы наружу — обход rate-limit и прямой доступ к БД | `docker-compose.yml:9-10,25-26` | phase7-critical-quick-fixes, 2026-08-14 |
| ✅ H1 | HIGH | Нет таймаутов HTTP-сервера и лимитов тела запроса (Slowloris, DoS) | `cmd/main.go:104`, `internal/handler/file.go:78` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ H2 | HIGH | ID файла не валидируется как UUID — риск path traversal при ошибках в `GetShardedPath` | `internal/storage/sharding.go:22-29`, `internal/handler/file.go:156-157` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ H3 | HIGH | Cookie без флага `Secure`, нет явной CSRF-защиты для cookie-сессий | `internal/auth/handler.go:52-59` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ H4 | HIGH | Удаление папки не удаляет файлы; эндпоинта удаления файлов нет вообще — бесконечный рост диска и «осиротевшие» файлы | `internal/handler/folder.go:183-231`, `cmd/main.go` | phase7-db-quotas-deletion, 2026-08-14 |
| ✅ H5 | HIGH | Контейнеры работают от root, файлы на хост-томе создаются root-ом | `services/storage-service/Dockerfile`, `services/web-frontend/Dockerfile` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M1 | MEDIUM | Строка подключения к БД собирается конкатенацией — ломается на спецсимволах в пароле; `sslmode=disable` | `cmd/main.go:49` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M2 | MEDIUM | Нет graceful shutdown — незавершённые загрузки обрываются при деплое/рестарте | `cmd/main.go:104` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M3 | MEDIUM | Миграции без журнала версий, выполняются целиком при каждом старте | `internal/database/database.go:45-68` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M4 | MEDIUM | Timing-перечисление пользователей при логине | `internal/auth/service.go:120-134` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M5 | MEDIUM | CSP разрешает `'unsafe-inline'` для скриптов | `services/web-frontend/nginx.conf:17` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M6 | MEDIUM | Срок жизни cookie захардкожен (24ч) и не связан с `sessionDuration` | `internal/auth/handler.go:58` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ M7 | MEDIUM | Ошибка парсинга UUID родителя молча игнорируется — рассинхрон память/БД | `internal/handler/folder.go:106-113` | phase7-db-quotas-deletion, 2026-08-14 |
| ✅ M8 | MEDIUM | Предпроверка размера загрузки сравнивает файл со всей квотой, а не с остатком | `internal/handler/file.go:66-75` | phase7-db-quotas-deletion, 2026-08-14 |
| ✅ L1 | LOW | Фронтенд шлёт несуществующее поле `state.currentPath` («undefined») | `services/web-frontend/src/app.js:200-204` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ L2 | LOW | Повторная загрузка того же файла не работает (не сбрасывается `input.value`) | `services/web-frontend/src/app.js:626-632` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ L3 | LOW | Индикатор квоты считает только файлы корня | `services/web-frontend/src/app.js:259-280` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ L4 | LOW | `Dockerfile` не копирует `go.sum` до `go mod download` | `services/storage-service/Dockerfile:6` | phase7-hardening-and-polish, 2026-08-14 |
| ✅ L5 | LOW | `Content-Disposition` без `filename*` (RFC 5987) для UTF-8 имён | `internal/handler/file.go:206` | phase7-final-cleanup, 2026-08-14 |

---

## Детализация находок

### ✅ C1 (phase7-critical-quick-fixes, 2026-08-14). Фолбэк на mock-аутентификацию с захардкоженными учётными данными
**Где:** `services/storage-service/cmd/main.go:54-57` и `:72-74`, `internal/auth/service.go:257-278`.  
**Решение:** Удалена mock-аутентификация, БД сделана обязательной, тесты переведены на Stub-double.

### ✅ C2 (phase7-db-quotas-deletion, 2026-08-14). Метаданные файлов хранятся только в оперативной памяти
**Где:** `internal/handler/file.go`.  
**Решение:** Перевод метаданных на таблицы PostgreSQL (`files`), полная валидация прав доступа и владельцев.

### ✅ C3 (phase7-critical-quick-fixes, 2026-08-14). `/api/v1/auth/me` всегда возвращает 401
**Где:** `cmd/main.go:86`.  
**Решение:** Обернут эндпоинт `/api/v1/auth/me` в `requireAuth` middleware.

### ✅ C4 (phase7-db-quotas-deletion, 2026-08-14). Нет контроля совокупной квоты — переполнение диска (DoS)
**Где:** `internal/handler/file.go`.  
**Решение:** Реализован подсчёт остатка квоты с блокировкой строки в транзакции (`FOR UPDATE`) и обновление `used_bytes`.

### ✅ C5 (phase7-critical-quick-fixes, 2026-08-14). Лишние опубликованные порты: `8080` и `5432`
**Где:** `docker-compose.yml`.  
**Решение:** Удалены публичные порты `8080` и `5432`, доступ только через internal docker network и Caddy/Nginx.

### ✅ H1 (phase7-hardening-and-polish, 2026-08-14). Нет таймаутов HTTP-сервера и лимита тела запроса
**Где:** `cmd/main.go`, `internal/handler/file.go`, `internal/auth/handler.go`.  
**Решение:** Настроены `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` и `http.MaxBytesReader`.

### ✅ H2 (phase7-hardening-and-polish, 2026-08-14). ID файла/папки из URL не валидируется как UUID
**Где:** `internal/handler/file.go`, `internal/storage/sharding.go`.  
**Решение:** Строгая проверка `uuid.Parse()` и проверка path traversal в `GetShardedPath`.

### ✅ H3 (phase7-hardening-and-polish, 2026-08-14). Cookie без `Secure`; CSRF-защита только через SameSite=Lax
**Где:** `internal/auth/handler.go`.  
**Решение:** Добавлены `Secure` cookie флагов и Origin/Host verification middleware.

### ✅ H4 (phase7-db-quotas-deletion, 2026-08-14). Нет удаления файлов; удаление папки оставляет файлы-сироты
**Где:** `internal/handler/folder.go`, `internal/handler/file.go`.  
**Решение:** Реализован `DELETE /api/v1/files/:id` и рекурсивное каскадное удаление файлов/папок с освобождением квоты и физическим стиранием с диска.

### ✅ H5 (phase7-hardening-and-polish, 2026-08-14). Контейнеры работают от root
**Где:** Dockerfiles.  
**Решение:** Перевод контейнеров на непривилегированных пользователей (`app` и `nginx-unprivileged`).

### ✅ M1 (phase7-hardening-and-polish, 2026-08-14). Строка подключения к БД собирается конкатенацией
**Где:** `cmd/main.go`.  
**Решение:** Использование `net/url` для безопасного построения PostgreSQL DSN.

### ✅ M2 (phase7-hardening-and-polish, 2026-08-14). Нет graceful shutdown
**Где:** `cmd/main.go`.  
**Решение:** Перехват `SIGTERM`/`SIGINT` и `srv.Shutdown(ctx)`.

### ✅ M3 (phase7-hardening-and-polish, 2026-08-14). Миграции без журнала версий
**Где:** `internal/database/database.go`.  
**Решение:** Таблица `schema_migrations` и транзакционное отслеживание версий миграций.

### ✅ M4 (phase7-hardening-and-polish, 2026-08-14). Timing-перечисление пользователей
**Где:** `internal/auth/service.go`.  
**Решение:** Фиктивный `bcrypt` вызов при отсутствии пользователя.

### ✅ M5 (phase7-hardening-and-polish, 2026-08-14). CSP с `'unsafe-inline'` для скриптов
**Где:** `services/web-frontend/nginx.conf`.  
**Решение:** Замена на `'self'` без `unsafe-inline`.

### ✅ M6 (phase7-hardening-and-polish, 2026-08-14). Срок cookie не связан с длительностью сессии
**Где:** `internal/auth/handler.go`.  
**Решение:** Синхронизация `Expires` cookie с конфигурацией `sessionDuration`.

### ✅ M7 (phase7-db-quotas-deletion, 2026-08-14). Молчаливое игнорирование ошибки парсинга UUID родителя
**Где:** `internal/handler/folder.go`.  
**Решение:** Возврат `400 Bad Request` при некорректном UUID.

### ✅ M8 (phase7-db-quotas-deletion, 2026-08-14). Предпроверка размера файла сравнивает со всей квотой
**Где:** `internal/handler/file.go`.  
**Решение:** Сравнение с остатком квоты (`quota - used`).

### ✅ L1 (phase7-hardening-and-polish, 2026-08-14). Фронтенд отправляет несуществующее поле
**Где:** `services/web-frontend/src/app.js`.  
**Решение:** Удалено ненужное поле `path`.

### ✅ L2 (phase7-hardening-and-polish, 2026-08-14). Невозможно загрузить тот же файл дважды подряд
**Где:** `services/web-frontend/src/app.js`.  
**Решение:** Сброс `input.value = ''` после вызова загрузки.

### ✅ L3 (phase7-hardening-and-polish, 2026-08-14). Индикатор квоты занижен
**Где:** `services/web-frontend/src/app.js`.  
**Решение:** Отображение реальных `used_bytes`/`quota_bytes` из эндпоинта `/api/v1/auth/me`.

### ✅ L4 (phase7-hardening-and-polish, 2026-08-14). Dockerfile не копирует go.sum
**Где:** `services/storage-service/Dockerfile`.  
**Решение:** Добавлен `COPY go.mod go.sum ./`.

### ✅ L5 (phase7-final-cleanup, 2026-08-14). Content-Disposition без filename* для UTF-8 имён
**Где:** `internal/handler/file.go`.  
**Решение:** Форматирование заголовка по RFC 5987 (`filename*=UTF-8''...`).
