# Tasks: phase7-critical-quick-fixes

Порядок фиксов по `design.md` D5: C5 → C1 → C3. Секции `[TEST-AGENT]` — только `*_test.go` (RED), `[CODE-AGENT]` — только production-код (GREEN). После каждой секции прогон `go test ./...` в `services/storage-service/`.

## 1. [TEST-AGENT] RED: регрессионные тесты блока C5 → C1 → C3

- [ ] 1.1 Тест C1: конфигурация старта — сервис не использует mock-фолбэк: `POSTGRES_HOST` пустой или `InitDB` ошибается → startup завершается фатально (тест на функцию/точку входа, без обращения к `MockAuthService`).
- [ ] 1.2 Тест C1: grep-гард или тест-ассерт, что в production-коде `services/` нет захардкоженных кредов (`adminpassword123`); подготовить stub, реализующий `auth.Service` без реальных кредов, и переписать использующие mock тесты `internal/auth/handler_test.go`, `internal/auth/auth_test.go` на него (RED до удаления mock).
- [ ] 1.3 Тест C3: интеграционный — `GET /api/v1/auth/me` с валидной сессией → `200` с профилем (id/email/quota); без сессии → `401`.
- [ ] 1.4 Тест C5: верификация `docker-compose.yml` — у сервисов `postgres` и `storage-service` отсутствуют блоки `ports` (декларативный тест/проверка в CI-стиле); `web-frontend` остаётся на `32214`.
- [ ] 1.5 Зафиксировать RED: `go test ./...` падает на новых тестах, существующие, не затронутые stub-миграцией, зелёные.

## 2. [CODE-AGENT] GREEN: фикс C5 (docker-compose)

- [ ] 2.1 Удалить блок `ports` у `postgres` (`docker-compose.yml:9-10`) и у `storage-service` (`:25-26`) целиком; убедиться, что nginx upstream в `web-frontend` ссылается на `storage-service:8080` по имени сети (править ничего не должно требоваться — проверить).
- [ ] 2.2 Прогнать `docker compose up -d` локально: приложение работает через `32214`, порты `8080`/`5432` на хосте закрыты. Тест 1.4 зелёный.

## 3. [CODE-AGENT] GREEN: фикс C1 (обязательная БД, удаление mock)

- [ ] 3.1 В `cmd/main.go`: пустой `POSTGRES_HOST` → `log.Fatalln`; ошибка `database.InitDB` → `log.Fatalf` (убрать ветки с `auth.NewMockAuthService()`).
- [ ] 3.2 Полностью удалить `MockAuthService` и `NewMockAuthService` из `internal/auth/service.go`.
- [ ] 3.3 Проверить `grep -rn "adminpassword123" services/` — ноль совпадений (остаться может только в `.github/workflows/ci.yml`).
- [ ] 3.4 Тесты 1.1, 1.2 зелёные; `go test ./...` полностью зелёный; `gofmt -l .` пустой.

## 4. [CODE-AGENT] GREEN: фикс C3 (auth/me за middleware)

- [ ] 4.1 В `cmd/main.go:86` заменить `http.HandleFunc("/api/v1/auth/me", ...)` на `http.Handle("/api/v1/auth/me", requireAuth(http.HandlerFunc(authHandler.MeHandler)))`.
- [ ] 4.2 Проверить, что `MeHandler` возвращает профиль (id, email, quota) из сервиса по `userID` из контекста; тест 1.3 зелёный.

## 5. [AUDIT-AGENT] Верификация фазы

- [ ] 5.1 Глубокий ревью всех изменённых `*.go`/`*_test.go` (`git diff --name-only origin/main`), `go test -v -cover ./...` (порог 85% в `internal/*`), `gofmt`, `docker compose up`, ручные приёмки из `BUGS.md` (C5: `nc -vz` refused; C1: без БД контейнер падает; C3: логин → перезагрузка страницы без модалки, `curl -b cookies /api/v1/auth/me` → 200), статус CI/CD `gh run list --limit 3`.
