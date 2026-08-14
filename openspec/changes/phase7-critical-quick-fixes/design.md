# Design: phase7-critical-quick-fixes

## Context

Текущее состояние и ограничения — см. `proposal.md` (Why) и `BUGS.md` (findings C5, C1, C3 с точными файло-строчными ссылками). Ключевые факты:

- `cmd/main.go:54-57,72-74` — при сбое `InitDB` или пустом `POSTGRES_HOST` поднимается `auth.NewMockAuthService()` с захардкоженной парой `admin@simplecloud.local` / `adminpassword123` (`internal/auth/service.go:257-278`).
- `cmd/main.go:86` — `/api/v1/auth/me` зарегистрирован через `http.HandleFunc` без обёртки `requireAuth`.
- `docker-compose.yml:9-10,25-26` — публикация `5432:5432` и `8080:8080`.
- `MockAuthService` используется в тестах: `internal/auth/handler_test.go`, `internal/auth/auth_test.go`.
- Тесты в CI (`.github/workflows/ci.yml`) исполняются против live PostgreSQL service-контейнера — БД в тестах доступна, mock не нужен.

## Goals / Non-Goals

**Goals:**
- Сервис физически не может стартовать без БД.
- В production-коде (`services/`) нет захардкоженных кредов.
- `/api/v1/auth/me` работает за middleware: 200 с сессией, 401 без.
- Хост не публикует `8080`/`5432`; весь трафик — через `web-frontend:32214`.
- Покрытие `internal/*` ≥ 85% сохранено; все существующие тесты переведены на test doubles и зелёные.

**Non-Goals:**
- Перевод метаданных файлов в PostgreSQL (C2), квоты (C4), удаление файлов (H4) — следующий change блока 2.
- Таймауты/graceful shutdown (H1/M2), CSRF/Secure cookie (H3), непривилегированные контейнеры (H5).

## Decisions

### D1. Полный отказ от mock-режима вместо «безопасного mock»
`main.go`: пустой `POSTGRES_HOST` → `log.Fatalln`; ошибка `database.InitDB` → `log.Fatalf`. Альтернатива — оставить mock без кредов или read-only режим — отклонена: mock-режим не несёт ценности для облачного хранилища и создаёт класс уязвимостей «отказ БД = открытый доступ». Тесты, которым нужен `auth.Service`, получают лёгкий stub (см. D2).

### D2. Test double вместо MockAuthService
Удалить `NewMockAuthService` и все его методы из `internal/auth/service.go`. Для тестов (`handler_test.go`, `auth_test.go`) ввести stub, реализующий интерфейс `auth.Service`, у которого токены/пользователи задаются параметрами теста, без захардкоженных «реальных» кредов. Размещение stub-типа — внутри `_test.go` файлов пакета (или общий `testutil`), чтобы он не попадал в production-бинарь. Альтернатива (вынести mock в отдельный пакет `authtest`) избыточна для текущего числа потребителей.

### D3. Фикс C3 — одна строка регистрации роута
`http.Handle("/api/v1/auth/me", requireAuth(http.HandlerFunc(authHandler.MeHandler)))` — тот же паттерн, что у `/api/v1/files`. `MeHandler` уже читает `userID` из контекста; поведение 401 при отсутствии сессии даёт сам middleware. Никаких изменений логики `MeHandler` не требуется (проверить, что он корректно возвращает профиль; если читает только `userID` — расширять не нужно).

### D4. C5 — убрать `ports:` целиком, без expose
Для `storage-service` блок `ports:` удаляется полностью; `expose` не обязателен (внутри compose-сети сервис доступен по имени + порту и так). Для `postgres` — аналогично. `web-frontend` остаётся `32214:80` + сеть `caddy-public`. Альтернатива `expose:` признана избыточной.

### D5. Порядок фиксов внутри change
`C5` (compose, нулевой риск для кода) → `C1` (main.go + удаление mock + миграция тестов) → `C3` (роут + интеграционный тест). Каждый шаг завершается зелёным `go test ./...`.

## Risks / Trade-offs

- [Тесты завязаны на `MockAuthService`, после удаления падают] → `[TEST-AGENT]` сначала пишет RED-тесты и stub, `[CODE-AGENT]` удаляет mock только когда stub покрывает все сценарии; прогон `go test ./...` на каждом шаге.
- [После снятия публикации портов ломается локальная отладка/деплой-скрипт, обращающийся к `:8080` напрямую] → проверить `deploy.yml` и nginx upstream: nginx проксирует на `storage-service:8080` по имени в compose-сети — не зависит от host-портов. Локальная отладка БД — через `docker compose exec postgres psql`.
- [`pi5server` после деплоя: старые опубликованные порты остаются до пересоздания контейнеров] → `docker compose up -d` в CD пересоздаёт контейнеры при изменении compose; приёмка — `nc -vz <host> 8080/5432` → refused.
- [Падение coverage после удаления mock-кода] → mock-код не покрыт тестами осмысленно; удаление скорее поднимет процент. `[AUDIT-AGENT]` проверяет порог 85%.

## Migration Plan

1. Merge → CI (lint, тесты против Postgres, coverage) → автодеплой на `pi5server` (`docker compose up -d --build`).
2. Пост-деплой проверка: приложение доступно через домен/`32214`; `nc -vz pi5server 8080` и `nc -vz pi5server 5432` → connection refused; логин → перезагрузка страницы → модалка логина не появляется.
3. Rollback: `git revert` merge-коммита → автодеплой вернёт прежний compose и код. Состояние данных не затрагивается (миграций БД нет).
