# AGENTS.md

Правила для AI-агентов, работающихся с проектом ShareTrip.

## Обязательные проверки

Перед завершением задачи (в этом порядке):

```bash
make lint   # golangci-lint, ~20 сек
make test   # go test -v ./internal/...
```

Не ослаблять и не отключать проверки линтера/тестов ради зелёного результата.
Подробности — в `skill.md`.

## Команды

| Команда | Что делает |
|---|---|
| `make lint` | `golangci-lint run ./...` (нужен установленный бинарь) |
| `make test` | Все тесты `./internal/...` |
| `make coverage` | Покрытие, отчёт в `reports/coverage.out` |
| `make all` | lint + test + coverage |
| `make generate-mocks` | `go generate ./...` (mockgen v0.6.0) |
| `make run` | `go1.25.5 run cmd/sharetrip/main.go` |
| `make migrate-up` / `migrate-create NAME=x` | Миграции через `goose`, DSN на порту **6543** |

Один тест:

```bash
go test ./internal/api/api_test/ -run 'Test_CreateTrip' -count=1 -v
```

Имена интеграционных тестов начинаются с `Test_` или `Test`, проверяй точное
имя в файле — `-run` работает по регулярному совпадению, опечатка даёт
«no tests to run» без ошибки.

## Тесты: интеграционные и юнитные

- **`internal/api/api_test/` — интеграционные, требуют запущенный Docker.**
  `TestMain` поднимает контейнер `postgres:16` (testcontainers) и применяет
  миграции из `migrations/` на каждый запуск пакета: даже один тест стоит
  ~5–30 секунд оверхеда на старт контейнера. Без Docker пакет падает сразу.
- Быстрая итерация без Docker: `go test ./internal/middleware/...` (юнитные,
  <1 сек).
- `make coverage` покрывает только `internal/api/api_test` +
  `internal/business/trip` — это осознанный выбор, не все пакеты.
- Фикстуры: `internal/api/api_test/fixtures_test.go`
  (`CreateTestTrip`, `CreateTestTripWithStatus`, `InsertTestRequest`,
  `CleanupTestData`, мок контракта `NewMockContractClient`).
- Токены для ролей: `internal/test_utils/token.go`
  (`GenerateTestToken`, `GeneratePassengerToken`, `GenerateDirectorToken`,
  `GenerateAdminToken`).
- `paralleltest` включён в `.golangci.yml`, но исключён для
  `internal/api/api_test/`. В остальных пакетах новые тесты должны быть
  параллельными (`t.Parallel()`), иначе линтер упадёт.

## Проверка ответов API в тестах

**Проверяй модель целиком через типизированную обёртку, а не поля через
`map[string]interface{}`.**

Обёртки живут в **`internal/api/response.go`** (файла `dto.go` в проекте нет):
`CreateTripResponseWrapper`, `ApproveTripResponseWrapper`,
`MoveTripDraftToPublishResponseWrapper`, `ErrorResponseWrapper` и т.д. Для
нового endpoint-а добавляй свою обёртку туда же.

```go
// ✅ правильно
var response api.CreateTripResponseWrapper
require.NoError(t, json.Unmarshal(respBody, &response))
require.Equal(t, "Success", response.Status)
require.Equal(t, payload.FromPoint, response.Data.FromPoint)
require.Equal(t, payload.Seats, response.Data.Seats)
require.Equal(t, "draft", response.Data.Status)

// ❌ запрещено
data, ok := response.Data.(map[string]interface{})
require.Equal(t, payload.Seats, int(data["seats"].(float64)))
```

Ошибка:

```go
var response api.ErrorResponseWrapper
require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
require.Equal(t, "Error", response.Status)
require.Contains(t, response.Message, "invalid trip status")
```

Тело ответа всегда закрывать с проверкой ошибки:

```go
defer func() {
    if err := resp.Body.Close(); err != nil {
        t.Errorf("failed to close response body: %v", err)
    }
}()
```

Зачем: типобезопасность на этапе компиляции, единый стиль, рефакторинг без
тихих падений.

## Структура проекта

Точка входа: `cmd/sharetrip/main.go` (загружает `./.env`, паникует без
настроек логгера). HTTP: Fiber, роуты в `internal/api/route.go`.

```
internal/
  api/              handlers, route.go, response.go, error_mapper.go
    api_test/       интеграционные тесты (Docker)
  business/
    trip/           entity / repository / service / domain
    request/        entity / repository / service / domain
    slot/           пустая заглушка: .go-файлов нет, в build не входит
    tripissue/      пустая заглушка: .go-файлов нет, в build не входит
  events/           outbox -> Kafka (trip.events)
  jobs/             воркеры (публикация, матчинг, expiry)
  access/           permissions из claim `sharetrip.permissions`
  middleware/       Keycloak (заголовок X-Refresh-Token), логирование
  shared/outbox/    паттерн outbox
  storage/          pgx pool, конфиг БД
```

Слои внутри домена зависят внутрь: `handler -> service -> repository ->
entity`. Доменная логика (`domain/`) не зависит от Fiber и pgx.

Следи за путём в `rule.md`: он частично устарел (упоминает
`internal/domain/`, слой `handler/` внутри домена, `golang-migrate`) —
актуальная структура и инструмент миграций (`goose`) — в коде и Makefile.

## Домен

Маршрут статусов поездки (константы в `internal/business/trip/entity`):

```
draft --> preapproved --> published --> started --> completed
             |
             +--> reject --> draft
canceled доступен из draft/preapproved/published/completed, но НЕ из started
(см. move_trip_to_canceled.go: единственная проверка — StatusStarted).
README утверждает, что completed тоже нельзя отменять, но код и тест
TestMoveTripToCanceled_FromAnyStatus это опровергают — доверяй коду.
```

Переходы реализованы отдельными файлами в
`internal/business/trip/domain/move_*.go`, каждый со своей проверкой
статуса и владельца. Права и видимость (роли driver/passenger/director/admin)
описаны в `README.md` — сверяйся с ним, а не с памятью.

## Git

Ветка по умолчанию `master`. Коммиты по-русски с номером задачи job4j:
`Outbox Inbox Transaction [#505678]`.
