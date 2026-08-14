# Design: phase7-db-metadata-quotas-deletion

## Context

Мотивация — в proposal.md (тикеты C2, C4, M8, H4, M7 из `BUGS.md`). Текущее состояние, определяющее дизайн:

- `FileHandler` (`internal/handler/file.go`) хранит метаданные в `map[string]FileMetadata` под `sync.RWMutex`; таблица `files` не используется вообще. `NewFileHandler(engine, defaultQuota)` не принимает пул БД.
- `FolderHandler` (`internal/handler/folder.go`) — dual-write: пишет в БД (если пул передан), но читает/валидирует из in-memory `map`; конструктор variadic `NewFolderHandler(args ...interface{})`.
- Схема БД готова: `users.quota_bytes/used_bytes`, `files` (+`folder_id` из миграции 000003, `ON DELETE CASCADE`), `folders` (`parent_id ON DELETE CASCADE`). Новая миграция не нужна.
- `used_bytes` нигде не обновляется; предпроверка при загрузке сравнивает размер файла с полным `defaultQuota`.
- Роуты в `cmd/main.go`: `/api/v1/files/upload`, `/api/v1/files/download/`, `/api/v1/files` (mux `net/http`, Go 1.22+ pattern routing доступен, но сейчас используется классический prefix-матчинг).

## Goals / Non-Goals

**Goals:**
- PostgreSQL — единственный источник метаданных файлов и папок; процесс не хранит никакого in-memory состояния метаданных.
- Атомарность: «файл на диске ⇔ запись в `files` ⇔ учёт в `used_bytes`».
- Квота считается от остатка, гонки при конкурентных загрузках исключены.
- Удаление файлов и рекурсивное удаление папок с физической очисткой диска.

**Non-Goals:**
- H1/H2/H3-хартенинг (таймауты сервера, MaxBytesReader, UUID-валидация ID в URL загрузки/скачивания, CSRF/Secure-cookie) — отдельные блоки Фазы 7. В этом изменении UUID валидируется только для `parent_id` (M7) и минимально для новых DELETE-роутов, где это тривиально.
- Фронтенд (L1–L5), включая индикатор квоты из `/api/v1/auth/me`.
- Пересборка/перезаполнение `used_bytes` по историческим файлам на диске (исторических записей нет — метаданные раньше не персистились, «наследовать» нечего; осиротевшие бинарные файлы на томе вне скоупа).
- Миграционный журнал (M3) — существующие миграции идемпотентны, новая миграция не добавляется.

## Decisions

### D1. Обязательный `*pgxpool.Pool` в обоих хэндлерах, типизированные конструкторы
`NewFileHandler(engine *storage.DiskEngine, pool *pgxpool.Pool, defaultQuota int64)` — пул обязателен (panic/nil-check на старте в `cmd/main.go` невозможен без БД — пул уже гарантированно создаётся до хэндлеров, см. фикс C1). `NewFolderHandler(pool *pgxpool.Pool, engine *storage.DiskEngine)` — типизированные параметры вместо `args ...interface{}`. Альтернатива «опциональный пул с фолбэком в память» отклонена: именно опциональность породила dual-write и рассинхрон (M7, H4).

### D2. Транзакция загрузки: FOR UPDATE на строке пользователя
Последовательность в `UploadHandler`:
1. `BEGIN`; `SELECT used_bytes, quota_bytes FROM users WHERE id = $1 FOR UPDATE`.
2. Предпроверка `Content-Length` (если есть) против `quota - used` → 413 до чтения тела.
3. `engine.Save(...)` с лимитом `quota - used` (а не `defaultQuota`) → при переливе стрим обрывается с 413, временной файл чистит engine.
4. `INSERT INTO files (...)` и `UPDATE users SET used_bytes = used_bytes + $size` в той же транзакции; `COMMIT`.
5. Если любой шаг после записи на диск падает — `os.Remove` записанного файла + rollback + 500.
Альтернатива «сначала сохранить, потом проверить/писать отдельными запросами» отклонена: окно гонки позволяет двумя конкурентными загрузками превысить квоту. `FOR UPDATE` выбран вместо advisory lock — проще и достаточно при нагрузке self-hosted.

### D3. Чтение/скачивание/список — только SQL
- Download: `SELECT user_id, filename, storage_path FROM files WHERE id = $1`; нет строки ИЛИ `user_id != userID` → единый 404 (анти-IDOR, без различия «нет» vs «чужой»).
- List файлов: `WHERE user_id = $1 AND ($2::uuid IS NULL AND folder_id IS NULL OR folder_id = $2)` — с сохранением текущей семантики `?folder_id=` (пустое значение/отсутствие параметра → корень).
- List/Create/Delete папок: аналогично из таблицы `folders`; проверка родителя при создании — `SELECT 1 FROM folders WHERE id = $1 AND user_id = $2`.

### D4. `DELETE /api/v1/files/:id` и роутинг
Новый `FileHandler.DeleteHandler`: валидация UUID из пути (не UUID → 404), `SELECT id, size_bytes, storage_path FROM files WHERE id = $1 AND user_id = $2` (нет → 404), одна транзакция: `DELETE FROM files` + `UPDATE users SET used_bytes = GREATEST(used_bytes - $size, 0)`, commit, затем `os.Remove(storage_path)`; если диск не удалился — лог warning (запись уже удалена, файл станет невидим и будет осиротевшим, но консистентность квоты важнее; обратный порядок «сначала диск» рисковал бы потерей записи при сбое БД). Роут: `http.Handle("/api/v1/files/{id}", requireAuth(...))` не использовать — паттерны Go 1.22 конфликтуют с существующим стилем; вместо этого `DELETE` различается по методу в обёртке на `/api/v1/files/`: регистрируется `http.Handle("/api/v1/files/", ...)`-диспетчер, где `GET` + prefix `download/` → Download, `DELETE` + один сегмент → Delete. Более специфичный зарегистрированный паттерн `/api/v1/files/download/` продолжает матчиться mux'ом первым — проверить тестом на конфликт роутов.

### D5. Рекурсивное удаление папки — один recursive CTE в транзакции
`DELETE /api/v1/folders/:id`: `WITH RECURSIVE tree AS (SELECT id FROM folders WHERE id=$1 AND user_id=$2 UNION ALL SELECT f.id FROM folders f JOIN tree t ON f.parent_id = t.id)`; из `tree` — `SELECT id, size_bytes, storage_path FROM files WHERE folder_id = ANY(...)`; затем `DELETE FROM files`, `DELETE FROM folders` (каскад подстрахует, но явное удаление даёт контроль), `UPDATE users SET used_bytes = GREATEST(used_bytes - $total, 0)`; commit; затем физическое удаление собранных файлов с диска (best-effort, лог ошибок). Альтернатива «рекурсия в Go с повторными запросами» (текущий `collectSubfoldersLocked`) отклонена: N запросов и гонки.

### D6. Валидация `parent_id` (M7)
`parent_id` задан и не пуст → `uuid.Parse`; ошибка → 400 `invalid parent_id` до любого обращения к БД. Родитель проверяется запросом в БД (не в памяти). Пустой/отсутствующий `parent_id` → корень (`NULL`).

### D7. Тестовая стратегия
- Юнит-тесты хэндлеров без БД — через существующий паттерн проекта: интеграционные тесты против живого PostgreSQL (адрес из env, по умолчанию `127.0.0.1:5432`) со skip при недоступности (как `auth_db_test.go`); в CI Postgres-сервис поднимается workflow'ом, поэтому в CI тесты исполняются.
- Обязательные регрессионные сценарии (из BUGS.md): «рестарт» = новый экземпляр хэндлера с тем же пулом → файлы видны владельцу, чужой UUID → 404 (C2); вторая загрузка сверх остатка квоты → 413 и корректный `used_bytes` (C4/M8); удаление файла/папки освобождает место и квоту, чужое удалить нельзя (H4); не-UUID `parent_id` → 400 (M7); конфликт роутов `/api/v1/files/download/` vs DELETE `/api/v1/files/:id`.
- Изоляция тестовых данных: уникальные email/UUID на тест, очистка созданных записей в `t.Cleanup` (идеи O1/O2 из IDEAS.md — отдельная работа, здесь придерживаемся текущего паттерна).
- Покрытие `internal/handler` ≥ 85% (обязательный порог).

## Risks / Trade-offs

- [Поведенческий разрыв для фронта: список файлов больше не «пропадает» после рестарта, но старые in-memory файлы без записей в БД станут невидимы] → Ожидаемо и желательно (C2): бинарные осиротевшие файлы на томе не удаляются автоматически, они не учитываются в квоте; документировать в отчёте аудита.
- [`FOR UPDATE` сериализует загрузки одного пользователя] → Нагрузка self-hosted (единицы пользователей) делает это приемлемым; блокировка держится только на время транзакции загрузки.
- [Диск-удаление после commit может оставить осиротевшие файлы при сбое] → Логирование; консистентность БД и квоты приоритетнее; возможна отдельная future-задача «gc осиротевших шардов».
- [Смена сигнатур конструкторов ломает все вызовы и тесты] → Намеренный BREAKING в пределах одного изменения; `cmd/main.go` и тесты обновляются синхронно (`[TEST-AGENT]` переписывает тесты первым в RED).
- [Существующие тесты, завязанные на in-memory поведение, упадут в RED] → Это ожидаемая фаза TDD; переписывание тестов входит в задачи изменения.

## Migration Plan

Новых миграций нет. Деплой штатный через CI/CD (`git push origin main` → GitHub Actions → `pi5server`). Откат — revert коммитов; данные в `files`/`folders` обратно в память не переносятся (в этом нет смысла: до изменения метаданные не персистились). Ручная приёмка по сценариям «Приёмка» из BUGS.md для C2/C4/H4 после деплоя.
