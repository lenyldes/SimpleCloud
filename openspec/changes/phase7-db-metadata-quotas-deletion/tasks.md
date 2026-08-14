## 1. RED: тесты персистентных метаданных файлов (C2)

- [x] 1.1 Обновить/переписать тесты `FileHandler`: конструктор принимает `*pgxpool.Pool`, in-memory map удалён — интеграционные тесты против живого PostgreSQL со skip при недоступности (паттерн `auth_db_test.go`)
- [x] 1.2 Тест: загрузка файла → запись в `files` (user_id, filename, size_bytes, sha256_hash, storage_path, folder_id) и файл виден в `GET /api/v1/files` владельца
- [x] 1.3 Тест «рестарта»: новый экземпляр `FileHandler` с тем же пулом → файл виден владельцу в списке и скачивается
- [x] 1.4 Тест IDOR: скачивание чужого файла и несуществующего UUID → одинаковый 404; файл без записи в БД не отдаётся
- [x] 1.5 Тест: `GET /api/v1/files` возвращает только файлы владельца (фильтры корня и `?folder_id=`)

## 2. RED: тесты транзакционной квоты (C4, M8)

- [x] 2.1 Тест: успешная загрузка инкрементирует `users.used_bytes` на размер файла
- [x] 2.2 Тест: вторая загрузка сверх остатка квоты (не полной квоты) → 413, `used_bytes` не меняется
- [x] 2.3 Тест: предпроверка `Content-Length` сравнивается с остатком `quota_bytes - used_bytes` → 413 до чтения тела
- [x] 2.4 Тест: сбой записи метаданных после сохранения на диск → файл удалён с диска, `used_bytes` не меняется, ответ 500

## 3. RED: тесты удаления файлов и папок (H4)

- [x] 3.1 Тест: `DELETE /api/v1/files/:id` владельцем → 200, бинарный файл удалён с диска, записи в `files` нет, `used_bytes` декрементирован
- [x] 3.2 Тест: удаление чужого файла и несуществующего/не-UUID id → 404, диск и БД не тронуты
- [x] 3.3 Тест: `used_bytes` не уходит в минус (clamp через GREATEST)
- [x] 3.4 Тест: `DELETE /api/v1/folders/:id` рекурсивно удаляет вложенные папки и файлы — записи в БД удалены, бинарные шарды удалены с диска, `used_bytes` скорректирован
- [x] 3.5 Тест: удаление чужой папки → 404; файлы удалённой папки больше не скачиваются (404)
- [x] 3.6 Тест роутинга: `DELETE /api/v1/files/:id` не конфликтует с `/api/v1/files/download/` и `/api/v1/files`

## 4. RED: тесты валидации parent_id (M7)

- [x] 4.1 Тест: `POST /api/v1/folders` с не-UUID `parent_id` → 400 `invalid parent_id`, запись не создаётся
- [x] 4.2 Тест: родитель существует и принадлежит пользователю (проверка по БД) → 201; чужой/несуществующий родитель → 404
- [x] 4.3 Тест «рестарта» для папок: новый экземпляр `FolderHandler` с тем же пулом → папки видны владельцу

## 5. GREEN: FileHandler на PostgreSQL (C2, C4, M8)

- [x] 5.1 `NewFileHandler(engine, pool, defaultQuota)` с обязательным `*pgxpool.Pool`; удалить `mu`/`files` map из `FileHandler`
- [x] 5.2 `UploadHandler`: транзакция — `SELECT used_bytes, quota_bytes ... FOR UPDATE`, предпроверка Content-Length против остатка, `engine.Save` с лимитом остатка, `INSERT INTO files` + `UPDATE users SET used_bytes = used_bytes + $size`, commit; при ошибке после записи на диск — `os.Remove` + rollback + 500
- [x] 5.3 `DownloadHandler`: `SELECT user_id, filename FROM files WHERE id = $1`; нет записи или чужой владелец → единый 404
- [x] 5.4 `ListHandler`: SQL-выборка файлов владельца с семантикой `?folder_id=` (отсутствие/пусто → корень)
- [x] 5.5 `DeleteHandler` файла: валидация UUID, ownership-check, транзакция `DELETE FROM files` + `UPDATE users SET used_bytes = GREATEST(used_bytes - $size, 0)`, затем `os.Remove(storage_path)` с логированием ошибок

## 6. GREEN: FolderHandler на PostgreSQL (C2, H4, M7)

- [x] 6.1 Типизированный конструктор `NewFolderHandler(pool, engine)`; удалить `mu`/`folders` map и `collectSubfoldersLocked`
- [x] 6.2 `CreateHandler`: валидация UUID `parent_id` → 400; проверка родителя SQL-запросом с `user_id`; `INSERT` только в БД
- [x] 6.3 `ListHandler`: SQL-выборка папок владельца с семантикой `?parent_id=`
- [x] 6.4 `DeleteHandler`: ownership-check (чужая/неизвестная → 404), recursive CTE по поддереву, сбор файлов, одна транзакция (`DELETE FROM files`, `DELETE FROM folders`, декремент `used_bytes` с clamp), затем физическое удаление шардов с диска (best-effort, лог)

## 7. GREEN: роутинг и сборка (H4)

- [x] 7.1 `cmd/main.go`: передать `dbPool` в `NewFileHandler` и `NewFolderHandler`; зарегистрировать `DELETE /api/v1/files/:id` (диспетчер по методу на `/api/v1/files/`, без конфликта с `/api/v1/files/download/`)
- [x] 7.2 Убедиться, что проект собирается: `go build ./...` и `gofmt -l .` пустой

## 8. Верификация

- [ ] 8.1 `go test -cover ./...`: все тесты GREEN, покрытие `internal/*` ≥ 85%
- [ ] 8.2 Ручная приёмка в контейнерах (`docker compose up`): загрузка → рестарт storage-service → файл виден и скачивается только владельцем; переполнение квоты → 413; удаление файла/папки освобождает диск и квоту (сценарии «Приёмка» BUGS.md для C2/C4/H4)
