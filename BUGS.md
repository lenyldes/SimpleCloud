# SimpleCloud — Отчёт об аудите: баги и уязвимости

> **Дата аудита:** 2026-08-14
> **Область:** `services/storage-service/` (Go), `services/web-frontend/` (Nginx + Vanilla JS), `docker-compose.yml`, `Dockerfile`, CI/CD.
> **Цель:** стабильность, отказоустойчивость, безопасность облачного хранилища.
>
> Документ рассчитан на исполнителя-LLM: каждое finding содержит **что сломано, где именно (файл:строка), чем опасно и пошаговую инструкцию по исправлению с примерами кода**. Исправляйте строго в порядке приоритета. После каждого исправления прогоняйте `go test ./...` в `services/storage-service/`.

---

## Сводная таблица

| ID | Критичность | Заголовок | Где |
|----|-------------|-----------|-----|
| ✅ C1 (phase7-critical-quick-fixes, 2026-08-14) | CRITICAL | Фолбэк на mock-аутентификацию с захардкоженным паролем при недоступности БД | `cmd/main.go:54-57`, `internal/auth/service.go:257-278` |
| ✅ C2 (phase7-db-quotas-deletion, 2026-08-14) | CRITICAL | Метаданные файлов только в памяти: потеря данных при рестарте + IDOR (скачивание чужих файлов) | `internal/handler/file.go:35-49,165-174` |
| ✅ C3 (phase7-critical-quick-fixes, 2026-08-14) | CRITICAL | `/api/v1/auth/me` зарегистрирован без middleware аутентификации — всегда 401 | `cmd/main.go:86` |
| ✅ C4 (phase7-db-quotas-deletion, 2026-08-14) | CRITICAL | Совокупная квота пользователя не контролируется, `used_bytes` не обновляется — исчерпание диска (DoS) | `internal/handler/file.go`, `internal/database/migrations/000001_init_schema.sql` |
| ✅ C5 (phase7-critical-quick-fixes, 2026-08-14) | CRITICAL | Порты `8080` (storage) и `5432` (PostgreSQL) опубликованы наружу — обход rate-limit и прямой доступ к БД | `docker-compose.yml:9-10,25-26` |
| ✅ H1 (phase7-hardening-and-polish, 2026-08-14) | HIGH | Нет таймаутов HTTP-сервера и лимитов тела запроса (Slowloris, DoS) | `cmd/main.go:104`, `internal/handler/file.go:78` |
| ✅ H2 (phase7-hardening-and-polish, 2026-08-14) | HIGH | ID файла не валидируется как UUID — риск path traversal при ошибках в `GetShardedPath` | `internal/storage/sharding.go:22-29`, `internal/handler/file.go:156-157` |
| ✅ H3 (phase7-hardening-and-polish, 2026-08-14) | HIGH | Cookie без флага `Secure`, нет явной CSRF-защиты для cookie-сессий | `internal/auth/handler.go:52-59` |
| ✅ H4 (phase7-db-quotas-deletion, 2026-08-14) | HIGH | Удаление папки не удаляет файлы; эндпоинта удаления файлов нет вообще — бесконечный рост диска и «осиротевшие» файлы | `internal/handler/folder.go:183-231`, `cmd/main.go` |
| ✅ H5 (phase7-hardening-and-polish, 2026-08-14) | HIGH | Контейнеры работают от root, файлы на хост-томе создаются root-ом | `services/storage-service/Dockerfile`, `services/web-frontend/Dockerfile` |
| ✅ M1 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | Строка подключения к БД собирается конкатенацией — ломается на спецсимволах в пароле; `sslmode=disable` | `cmd/main.go:49` |
| ✅ M2 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | Нет graceful shutdown — незавершённые загрузки обрываются при деплое/рестарте | `cmd/main.go:104` |
| ✅ M3 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | Миграции без журнала версий, выполняются целиком при каждом старте | `internal/database/database.go:45-68` |
| ✅ M4 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | Timing-перечисление пользователей при логине | `internal/auth/service.go:120-134` |
| ✅ M5 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | CSP разрешает `'unsafe-inline'` для скриптов | `services/web-frontend/nginx.conf:17` |
| ✅ M6 (phase7-hardening-and-polish, 2026-08-14) | MEDIUM | Срок жизни cookie захардкожен (24ч) и не связан с `sessionDuration` | `internal/auth/handler.go:58` |
| ✅ M7 (phase7-db-quotas-deletion, 2026-08-14) | MEDIUM | Ошибка парсинга UUID родителя молча игнорируется — рассинхрон память/БД | `internal/handler/folder.go:106-113` |
| ✅ M8 (phase7-db-quotas-deletion, 2026-08-14) | MEDIUM | Предпроверка размера загрузки сравнивает файл со всей квотой, а не с остатком | `internal/handler/file.go:66-75` |
| ✅ L1 (phase7-hardening-and-polish, 2026-08-14) | LOW | Фронтенд шлёт несуществующее поле `state.currentPath` («undefined») | `services/web-frontend/src/app.js:200-204` |
| ✅ L2 (phase7-hardening-and-polish, 2026-08-14) | LOW | Повторная загрузка того же файла не работает (не сбрасывается `input.value`) | `services/web-frontend/src/app.js:626-632` |
| ✅ L3 (phase7-hardening-and-polish, 2026-08-14) | LOW | Индикатор квоты считает только файлы корня | `services/web-frontend/src/app.js:259-280` |
| ✅ L4 (phase7-hardening-and-polish, 2026-08-14) | LOW | `Dockerfile` не копирует `go.sum` до `go mod download` | `services/storage-service/Dockerfile:6` |
| L5 | LOW | `Content-Disposition` без `filename*` (RFC 5987) для UTF-8 имён | `internal/handler/file.go:206` |

---

## CRITICAL — исправлять в первую очередь

### ✅ C1 (phase7-critical-quick-fixes, 2026-08-14). Фолбэк на mock-аутентификацию с захардкоженными учётными данными

**Где:** `services/storage-service/cmd/main.go:54-57` и `:72-74`, `internal/auth/service.go:257-278`.

**Что происходит:** Если инициализация БД не удалась (БД не поднялась, неверный пароль, таймаут), сервис **не падает**, а продолжает работу с `MockAuthService`. У mock-сервиса захардкожена учётная запись:

```go
if email == "admin@simplecloud.local" && password == "adminpassword123" {
```

**Чем опасно:** Любой, кто знает (а теперь — и прочитал в исходниках) эти креды, входит как admin при любом сбое БД. Для облачного хранилища это полный компрометирующий сценарий: «отказ БД» превращается в «открытый доступ».

**Как исправить (пошагово):**
1. В `cmd/main.go` замените ветку «DB init failed → mock» на фатальную ошибку:
   ```go
   dbPool, err := database.InitDB(ctx, connStr)
   if err != nil {
       log.Fatalf("Database initialization failed: %v", err)
   }
   ```
2. Там же: если переменная окружения `POSTGRES_HOST` пустая, сервис сейчас тоже поднимает mock. Для production-сборки это недопустимо. Сделайте БД обязательной:
   ```go
   if dbHost == "" {
       log.Fatalln("POSTGRES_HOST is not set; refusing to start without database")
   }
   ```
3. Полностью удалите `MockAuthService` из `internal/auth/service.go` (методы `Login/Logout/ValidateSession/GetUserByID/CreateValidSessionToken` и `NewMockAuthService`). Тесты, которые его используют (`auth_test.go`, `file_test.go` и т.д.), переведите на лёгкий тестовый double, реализующий интерфейс `Service` **без захардкоженных реальных кредов** (например, stub, принимающий токен как параметр теста).
4. Проверьте: `grep -rn "adminpassword123" services/` должен остаться **только** в `.github/workflows/ci.yml` (тестовое окружение CI) и ни в коем случае не в production-коде.

**Приёмка:** сервис не стартует без доступной БД (проверить: `docker compose stop postgres && docker compose up storage-service` → контейнер падает с ошибкой, а не отвечает на логин).

---

### ✅ C2 (phase7-db-quotas-deletion, 2026-08-14). Метаданные файлов хранятся только в оперативной памяти

**Где:** `internal/handler/file.go:35-49` (`files map[string]FileMetadata`), `:132-134` (запись), `:165-174` (проверка владельца при скачивании), `:232-254` (список).

**Что происходит:**
1. Все метаданные файлов (имя, владелец, папка, размер) живут в `map` в памяти. Таблица `files` в PostgreSQL (миграция `000001_init_schema.sql`) **не используется вообще** — ни один `INSERT INTO files` в коде не выполняется.
2. При любом рестарте контейнера (деплой через CI делает это на каждый push!) список файлов **полностью теряется**, хотя бинарные файлы остаются на диске.
3. Хуже того — проверка владельца в `DownloadHandler`:
   ```go
   if exists && meta.UserID != "" && meta.UserID != userID.String() { ... 404 ... }
   ```
   После рестарта `exists == false` для **всех** файлов → проверка пропускается → **любой аутентифицированный пользователь может скачать любой файл на диске, подобрав/перебрав UUID** (IDOR + перечисление).

**Чем опасно:** Потеря пользовательских данных (главное свойство облака) и утечка чужих файлов.

**Как исправить (пошагово):**
1. Передайте `*pgxpool.Pool` в `FileHandler` (аналогично тому, как это сделано в `FolderHandler`), но сделайте его **обязательным**, а не опциональным:
   ```go
   func NewFileHandler(engine *storage.DiskEngine, pool *pgxpool.Pool, defaultQuota int64) *FileHandler
   ```
2. В `UploadHandler` после успешного `engine.Save(...)` выполняйте:
   ```go
   _, err := fh.pool.Exec(r.Context(),
       `INSERT INTO files (id, user_id, folder_id, filename, size_bytes, sha256_hash, storage_path)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
       fileID, userID, folderIDPtr, header.Filename, size, sha256Hex, filePath)
   ```
   Если `INSERT` не удался — **удалите уже записанный файл с диска** (`os.Remove`) и верните 500. Иначе появятся файлы без владельца.
3. В `DownloadHandler` замените обращение к `fh.files` на запрос:
   ```go
   var ownerID uuid.UUID
   var filename string
   err := fh.pool.QueryRow(r.Context(),
       `SELECT user_id, filename FROM files WHERE id = $1`, fileID).Scan(&ownerID, &filename)
   if err != nil || ownerID != userID {
       // 404 file not found (единый ответ и для «нет файла», и для «чужой файл»)
   }
   ```
   Ключевое правило: **нет записи в БД → 404**. Никогда не отдавайте файл, которого нет в БД.
4. `ListHandler` переписать на SQL: `SELECT ... FROM files WHERE user_id = $1 AND (folder_id = $2 OR ($2::uuid IS NULL AND folder_id IS NULL))`.
5. Полностью удалите in-memory `map files` и мьютекс из `FileHandler`.
6. Загрузку метаданных папок тоже переведите на БД (сейчас `FolderHandler` пишет в БД, но читает из памяти — тот же класс бага C2: после рестарта папки «пропадают»).
7. Обновите тесты: они должны проверять, что после перезапуска хэндлера (новый инстанс с тем же пулом) файлы по-прежнему видны владельцу и невидимы чужим.

**Приёмка:** загрузить файл → перезапустить контейнер storage-service → файл виден в списке и скачивается только владельцем; чужой/неизвестный UUID → 404.

---

### ✅ C3 (phase7-critical-quick-fixes, 2026-08-14). `/api/v1/auth/me` всегда возвращает 401

**Где:** `cmd/main.go:86`.

**Что происходит:** Роут зарегистрирован как `http.HandleFunc("/api/v1/auth/me", authHandler.MeHandler)` — **без обёртки `requireAuth`**. `MeHandler` читает `userID` из контекста запроса, но кладёт его туда только middleware `RequireAuth`. Контекст всегда пустой → всегда 401.

**Чем опасно/вредно:** Фронтенд при старте вызывает `/api/v1/auth/me` (`app.js:93`), получает 401 и показывает модалку логина даже залогиненному пользователю. Фактически сессия «не переживает» перезагрузку страницы. Это и баг стабильности UX, и признак того, что middleware-слой проверен не полностью.

**Как исправить:**
```go
http.Handle("/api/v1/auth/me", requireAuth(http.HandlerFunc(authHandler.MeHandler)))
```
(строку 86 заменить, обычный `HandleFunc` → `Handle` с обёрткой, как у `/api/v1/files`).

**Приёмка:** залогиниться → перезагрузить страницу → модалка логина НЕ появляется; `curl -b cookies /api/v1/auth/me` возвращает 200 с профилем. Добавить интеграционный тест: `GET /api/v1/auth/me` с валидной сессией → 200, без сессии → 401.

---

### ✅ C4 (phase7-db-quotas-deletion, 2026-08-14). Нет контроля совокупной квоты — переполнение диска (DoS)

**Где:** `internal/handler/file.go:66-75` и `:109`; колонки `quota_bytes/used_bytes` в `users` существуют, но не обновляются нигде.

**Что происходит:**
- Проверка при загрузке сравнивает размер **одного** файла со всей квотой (5 ГБ), а не с **остатком** квоты пользователя.
- `used_bytes` в таблице `users` никогда не инкрементируется.
- Пользователь может загрузить неограниченное число файлов по 4.9 ГБ и заполнить весь диск сервера. Для self-hosted облака на Raspberry Pi/малом VPS это гарантированная смерть сервиса (падает и БД, и Caddy — всё на одном диске).

**Как исправить (пошагово):**
1. Перед сохранением читайте текущее использование в одной транзакции с блокировкой строки пользователя:
   ```go
   tx, err := fh.pool.Begin(ctx)
   // ...
   var used, quota int64
   tx.QueryRow(ctx, `SELECT used_bytes, quota_bytes FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&used, &quota)
   ```
2. Сравните `Content-Length` (если есть) с `quota - used`; при превышении — 413.
3. После успешного `engine.Save` и `INSERT INTO files` — `UPDATE users SET used_bytes = used_bytes + $2 WHERE id = $1` и `tx.Commit()`.
4. Лимитируйте тело запроса жёстко через `http.MaxBytesReader` (см. H1) — квота не должна быть единственным ограничителем.
5. Предпроверку `Content-Length` в `file.go:66-75` переделать на сравнение с остатком (`quota - used`), а не с `fh.defaultQuota` (это устраняет и M8).

**Приёмка:** при квоте 5 ГБ вторая загрузка, суммарно превышающая 5 ГБ, отклоняется с 413; `SELECT used_bytes FROM users` растёт и уменьшается после удалений (см. H4).

---

### ✅ C5 (phase7-critical-quick-fixes, 2026-08-14). Лишние опубликованные порты: `8080` и `5432`

**Где:** `docker-compose.yml:9-10` (postgres) и `:25-26` (storage-service).

**Что происходит:**
- `storage-service` опубликован на `8080` хоста. Весь трафик должен идти через nginx-фронтенд (`32214`), где настроены rate-limit и `client_max_body_size 100M`. Прямое обращение к `:8080` **обходит все эти защиты** (брутфорс логина без лимита 5r/s, загрузки без ограничения размера).
- `postgres` опубликован на `5432` хоста. Если файрвол сервера не режет порт, база с учётками и хэшами паролей доступна всей сети/интернету.

**Как исправить:**
1. Уберите публикацию портов storage-service полностью — он доступен фронтенду внутри docker-сети по имени `storage-service:8080`:
   ```yaml
   storage-service:
     # блок ports удалить целиком
     expose:
       - "8080"
   ```
2. Для postgres тоже уберите `ports:` — доступ снаружи контейнеру не нужен:
   ```yaml
   postgres:
     # блок ports удалить целиком
   ```
   (Для локальной отладки БД используйте `docker compose exec postgres psql ...`.)
3. На сервере `pi5server` дополнительно проверьте файрвол: снаружи должны быть открыты только 80/443 (Caddy) и SSH.

**Приёмка:** с другой машины `nc -vz <host> 8080` и `nc -vz <host> 5432` → connection refused; приложение продолжает работать через `32214`/домен.

---

## HIGH

### ✅ H1 (phase7-hardening-and-polish, 2026-08-14). Нет таймаутов HTTP-сервера и лимита тела запроса

**Где:** `cmd/main.go:104` (`http.ListenAndServe`), `internal/handler/file.go:78`, `internal/auth/handler.go:34` (`json.NewDecoder(r.Body)` без лимита).

**Что происходит:** Используется дефолтный сервер без `ReadTimeout/WriteTimeout/IdleTimeout` → классический Slowloris: тысячи медленных соединений вешают сервис. Тела JSON-запросов (логин, создание папки) читаются без ограничения размера (лимит 1M есть только в nginx, но см. C5 — прямой доступ его обходит).

**Как исправить:**
1. В `cmd/main.go` замените `http.ListenAndServe` на настроенный сервер:
   ```go
   srv := &http.Server{
       Addr:              ":" + port,
       ReadHeaderTimeout: 10 * time.Second,
       ReadTimeout:       30 * time.Second,
       WriteTimeout:      5 * time.Minute, // большие загрузки
       IdleTimeout:       60 * time.Second,
   }
   if err := srv.ListenAndServe(); err != nil { ... }
   ```
2. В `UploadHandler` сразу после проверки метода оберните тело:
   ```go
   r.Body = http.MaxBytesReader(w, r.Body, fh.defaultQuota+1<<20) // квота + небольшой запас на multipart-заголовки
   ```
   При превышении `ParseMultipartForm` вернёт ошибку — верните 413.
3. В `LoginHandler` и `CreateHandler` (папки) лимитируйте JSON:
   ```go
   r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB достаточно
   ```

---

### ✅ H2 (phase7-hardening-and-polish, 2026-08-14). ID файла/папки из URL не валидируется как UUID

**Где:** `internal/handler/file.go:156-163`, `internal/storage/sharding.go:22-29`.

**Что происходит:** `GetShardedPath` принимает любую строку длиной ≥4 и строит из неё путь `filepath.Join(baseDir, id[0:2], id[2:4], id)`. Сейчас спасает только то, что `net/http` нормализует `..` в URL-пути. Это «защита по счастливой случайности» — любое изменение роутинга (например, переход на сырой `r.URL.EscapedPath()` или другой мультиплексор) мгновенно даёт path traversal и чтение произвольных файлов.

**Как исправить:** В `DownloadHandler` и `DeleteHandler` (папки) сразу после извлечения ID:
```go
fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/files/download/")
if _, err := uuid.Parse(fileID); err != nil {
    // 400 bad request / 404
    return
}
```
Дополнительно — защита в глубину в `GetShardedPath`:
```go
if strings.ContainsAny(fileID, "/\\") || filepath.Base(fileID) != fileID {
    return "", ErrInvalidFileID
}
```

---

### ✅ H3 (phase7-hardening-and-polish, 2026-08-14). Cookie без `Secure`; CSRF-защита только через SameSite=Lax

**Где:** `internal/auth/handler.go:52-59` и `:86-93`.

**Что происходит:** Сессионный cookie устанавливается без флага `Secure`. Приложение работает за Caddy с HTTPS, но между Caddy и контейнерами трафик идёт по HTTP; при любом прямом HTTP-доступе cookie улетает в открытом виде. Также все state-changing запросы (upload, создание/удаление папок, logout) идентифицируются исключительно cookie — классическая поверхность CSRF.

**Как исправить:**
1. Добавьте `Secure: true` в оба `http.SetCookie` (при локальной разработке без TLS это можно делать зависимым от env-переменной `COOKIE_SECURE=false`).
2. Добавьте проверку заголовка происхождения на мутирующих эндпоинтах (простая и надёжная схема):
   ```go
   // в middleware или в начале Upload/Create/Delete/Login-хэндлеров
   if origin := r.Header.Get("Origin"); origin != "" {
       host := r.Header.Get("X-Forwarded-Host")
       if host == "" { host = r.Host }
       u, err := url.Parse(origin)
       if err != nil || u.Host != host {
           // 403 forbidden
       }
   }
   ```
   SameSite=Lax оставьте как второй рубеж.

---

### ✅ H4 (phase7-db-quotas-deletion, 2026-08-14). Нет удаления файлов; удаление папки оставляет файлы-сироты

**Где:** `internal/handler/folder.go:183-231`; эндпоинта `DELETE /api/v1/files/:id` нет вообще (см. `cmd/main.go:88-101`).

**Что происходит:**
- Файлы невозможно удалить через API. Диск только растёт, квоту освободить нельзя.
- Удаление папки стирает папку (и подпапки) из map/БД, но файлы внутри остаются на диске и в списке файлов (в памяти), ссылаясь на несуществующую папку. В БД при этом `files.folder_id` каскадно занулится/удалится, а в памяти — нет: очередной рассинхрон.

**Как исправить:**
1. Добавьте `DELETE /api/v1/files/:id` в `FileHandler`:
   - найти запись `SELECT ... FROM files WHERE id = $1 AND user_id = $2` (чужие → 404);
   - удалить файл с диска (`os.Remove` по `storage_path`);
   - `DELETE FROM files WHERE id = $1`;
   - `UPDATE users SET used_bytes = used_bytes - $2 WHERE id = $1` (не дать уйти в минус: `GREATEST(used_bytes - $2, 0)`).
2. В `DeleteHandler` папок: собрать ID всех файлов в удаляемых папках, удалить их с диска и из БД, скорректировать `used_bytes` — и только потом удалять сами папки. Оберните всё в одну SQL-транзакцию.
3. Зарегистрируйте роут в `cmd/main.go`:
   ```go
   http.Handle("/api/v1/files/", requireAuth(http.HandlerFunc(fileHandler.DeleteHandler)))
   ```
   и убедитесь, что он не конфликтует с `/api/v1/files/download/` (более специфичный паттерн `/api/v1/files/download/` mux выбирает первым — проверьте тестом).

---

### ✅ H5 (phase7-hardening-and-polish, 2026-08-14). Контейнеры работают от root

**Где:** `services/storage-service/Dockerfile` (нет `USER`), `services/web-frontend/Dockerfile` (nginx по умолчанию мастер от root).

**Что происходит:** При любом RCE в сервисе атакующий получает root в контейнере и создаёт файлы на хост-томе `./data/storage` с владельцем root.

**Как исправить:**
1. В `storage-service/Dockerfile` (stage 2):
   ```dockerfile
   RUN addgroup -S app && adduser -S app -G app
   RUN mkdir -p /storage && chown app:app /storage
   USER app
   ```
2. Для nginx используйте образ `nginxinc/nginx-unprivileged` (слушает 8080 от пользователя nginx) и поправьте `listen` в `nginx.conf` и маппинг портов в compose (`32214:8080`).
3. После смены пользователя проверьте права на существующий том `./data/storage` на сервере (может потребоваться `chown -R` один раз).

---

## MEDIUM

### ✅ M1 (phase7-hardening-and-polish, 2026-08-14). Строка подключения к БД собирается конкатенацией

**Где:** `cmd/main.go:49`.

**Что происходит:** Пароль, содержащий `@`, `:`, `/` или `%`, ломает URL подключения (сервис не стартует — баг стабильности при «неудобном» пароле). Также используется `sslmode=disable`.

**Как исправить:**
```go
import "net/url"

connURL := url.URL{
    Scheme: "postgres",
    User:   url.Password(dbUser, dbPass), // корректное экранирование
    Host:   dbHost + ":" + dbPort,
    Path:   dbName,
}
q := connURL.Query()
q.Set("sslmode", "disable") // внутри изолированной docker-сети допустимо
connURL.RawQuery = q.Encode()
connStr := connURL.String()
```

### ✅ M2 (phase7-hardening-and-polish, 2026-08-14). Нет graceful shutdown

**Где:** `cmd/main.go:104`.

**Что происходит:** `docker stop` шлёт SIGTERM, но `http.ListenAndServe` его не обрабатывает: активные загрузки обрываются, `defer dbPool.Close()` не выполняется, временные файлы `.upload-*` могут оставаться на диске.

**Как исправить:** стандартная схема:
```go
srv := &http.Server{...}
go srv.ListenAndServe()
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
<-stop
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx)
```

### ✅ M3 (phase7-hardening-and-polish, 2026-08-14). Миграции без журнала версий

**Где:** `internal/database/database.go:45-68`.

**Что происходит:** Все SQL-файлы исполняются при каждом старте; идемпотентность держится исключительно на `IF NOT EXISTS`. Первая же «необратимая» миграция (переименование колонки, изменение типа) сломает старт.

**Как исправить:** минимально — заведите таблицу `schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ)` и в транзакции проверяйте/вставляйте версию перед исполнением каждого файла. Файл миграции исполнять внутри `BEGIN/COMMIT`.

### ✅ M4 (phase7-hardening-and-polish, 2026-08-14). Timing-перечисление пользователей

**Где:** `internal/auth/service.go:120-134`.

**Что происходит:** Если email не найден, ответ возвращается мгновенно; если найден — выполняется медленный bcrypt. По времени ответа можно перечислить существующие email.

**Как исправить:** при отсутствии пользователя всё равно выполнить сравнение с фиктивным хэшем:
```go
var dummyHash = "$2a$10$..." // предвычисленный bcrypt-хэш любой строки
if err != nil { // user not found
    CheckPasswordHash(password, dummyHash) // сжечь то же время
    return "", nil, ErrInvalidCredentials
}
```

### ✅ M5 (phase7-hardening-and-polish, 2026-08-14). CSP с `'unsafe-inline'` для скриптов

**Где:** `services/web-frontend/nginx.conf:17`.

**Что происходит:** `script-src 'self' 'unsafe-inline'` сводит на нет защиту CSP от XSS. Весь JS у вас во внешнем `app.js` — инлайн-скриптов быть не должно.

**Как исправить:** уберите `'unsafe-inline'` из `script-src`. Если в `index.html` есть инлайн-обработчики (`onclick=...`) — вынесите их в `app.js`. Проверьте консоль браузера на сообщения о заблокированных скриптах после изменения.

### ✅ M6 (phase7-hardening-and-polish, 2026-08-14). Срок cookie не связан с длительностью сессии

**Где:** `internal/auth/handler.go:58`.

**Что происходит:** Cookie всегда живёт 24 часа, хотя `sessionDuration` передаётся в сервис. Если сессию укоротят (например, до 1 часа), cookie продолжит отправляться и валидироваться в БД будет уже после истечения — лишний шум и путаница.

**Как исправить:** пробросьте длительность сессии в `AuthHandler` (через конструктор или метод сервиса `SessionTTL()`) и используйте её в `Expires`.

### ✅ M7 (phase7-db-quotas-deletion, 2026-08-14). Молчаливое игнорирование ошибки парсинга UUID родителя

**Где:** `internal/handler/folder.go:106-113`.

**Что происходит:** Если `parent_id` не парсится как UUID, в БД пишется `NULL`-родитель, а в памяти остаётся ссылка на родителя — рассинхронизация данных.

**Как исправить:** если `parent_id` задан, но не парсится — возвращайте 400 `invalid parent_id`. И вообще после перевода метаданных на БД (C2) этот dual-write исчезнет.

### ✅ M8 (phase7-db-quotas-deletion, 2026-08-14). Предпроверка размера файла сравнивает со всей квотой

**Где:** `internal/handler/file.go:66-75`. См. C4 — исправляется одновременно (сравнение с остатком квоты).

---

## LOW / UX

### ✅ L1 (phase7-hardening-and-polish, 2026-08-14). Фронтенд отправляет несуществующее поле

**Где:** `services/web-frontend/src/app.js:200-204`.

`formData.append('path', state.currentPath)` — `state.currentPath` не определён никогда; в запрос уходит строка `"undefined"`. Сервер её игнорирует, но это мусор и потенциальная путаница. **Исправление:** удалить ветку `else` целиком (папка передаётся через `folder_id`).

### ✅ L2 (phase7-hardening-and-polish, 2026-08-14). Невозможно загрузить тот же файл дважды подряд

**Где:** `services/web-frontend/src/app.js:626-632`.

Событие `change` у `<input type=file>` не срабатывает, если значение не изменилось. **Исправление:** после запуска `handleFileUpload(...)` сделать `fileUploadInput.value = ''`.

### ✅ L3 (phase7-hardening-and-polish, 2026-08-14). Индикатор квоты занижен

**Где:** `services/web-frontend/src/app.js:259-280`.

`loadFiles()` запрашивает только файлы корня (без `folder_id`), поэтому `state.quota.used` не учитывает файлы в папках. **Исправление:** после реализации серверной квоты (C4) отдавайте `used_bytes/quota_bytes` в ответе `/api/v1/auth/me` и рисуйте индикатор из него, а не суммируйте список на клиенте.

### ✅ L4 (phase7-hardening-and-polish, 2026-08-14). Dockerfile не копирует go.sum

**Где:** `services/storage-service/Dockerfile:6`.

`RUN go mod download` выполняется без `go.sum` → зависимости скачиваются без верификации контрольных сумм, кэш слоя менее детерминирован. **Исправление:**
```dockerfile
COPY go.mod go.sum ./
RUN go mod download
```

### L5. Content-Disposition без filename* для UTF-8 имён

**Где:** `internal/handler/file.go:206`.

Кириллические имена файлов у части браузеров приедут «кракозябрами». **Исправление:**
```go
w.Header().Set("Content-Disposition",
    fmt.Sprintf(`attachment; filename=%q; filename*=UTF-8''%s`,
        fallbackASCIIName, url.PathEscape(filename)))
```

---

## Рекомендуемый порядок работ для следующего этапа

1. **C5 → C1 → C3** (быстрые, изолированные правки, максимальное снижение риска).
2. **C2 + C4 + H4 + M8** — единый блок: перевод метаданных файлов/папок в PostgreSQL, квоты, удаление файлов. Это самая большая часть; разбить на подзадачи: сначала `files` в БД (чтение+запись), затем квоты, затем удаление.
3. **H1 + M2** — таймауты, лимиты тела, graceful shutdown.
4. **H2, H3, M4, M6, M7, M1, M3** — hardening.
5. **H5** — непривилегированные контейнеры (отдельно, потребует прав доступа на томе при деплое).
6. **M5, L1–L5** — фронтенд и полировка.
7. После каждого блока: `gofmt -l .` пустой, `go test -cover ./...` ≥ 85%, прогнать `docker compose up` и вручную проверить сценарии из раздела «Приёмка».

## Обязательные регрессионные тесты (добавить при исправлении)

- Сервис не стартует без БД (C1).
- После «рестарта» (новый хэндлер с тем же пулом) файлы видны владельцу, чужой UUID → 404 (C2).
- `GET /api/v1/auth/me` с сессией → 200 (C3).
- Вторая загрузка сверх квоты → 413, `used_bytes` корректен (C4).
- Удаление файла освобождает место и квоту; чужой файл удалить нельзя (H4).
- Запрос с телом сверх лимита → 413, медленное соединение не вешает сервер (H1).
