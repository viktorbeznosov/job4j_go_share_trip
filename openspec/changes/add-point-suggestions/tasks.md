# Tasks

## 1. Слой доступа к данным

- [x] 1.1 Добавить в `internal/business/trip/repository/` метод `SuggestPoints(ctx context.Context, term string) ([]string, error)` с параметризованным `UNION ALL` по `from_point`/`to_point` таблиц `trips` и `trip_requests`, `DISTINCT`, `ILIKE`-фильтром, `ORDER BY lower(point)`, `LIMIT 10` (design D1, D7) и проверить компиляцию `go build ./...`
- [x] 1.2 Реализовать экранирование `%`, `_`, `\` в `term` перед подстановкой в ILIKE-шаблон (design D2) и покрыть юнит-тестом в том же пакете: литеральный `%` не матчит всё, литеральный `_` не матчит произвольный символ — проверка `go test ./internal/business/trip/repository/ -run TestEscape -v`
- [x] 1.3 Проверить, что пустой `term` не фильтрует и возвращает до 10 пунктов, а запрос к таблицам без записей возвращает пустой срез (не `nil`) — тест в `internal/api/api_test/fixtures_test.go`-стиле, проверка `go test ./internal/api/api_test/ -run 'TestSuggestPoints' -count=1`

## 2. API-слой и маршрут

- [x] 2.1 Добавить `SuggestPointsResponse` и `SuggestPointsResponseWrapper` в `internal/api/response.go` по форме существующих обёрток (design D4) и проверить компиляцию `go build ./...`
- [x] 2.2 Добавить константу ошибки превышения длины `term` (256 символов) в `internal/api/errors/` с маппингом HTTP-статуса и проверить `go vet ./...`
- [x] 2.3 Реализовать обработчик `GET /points` в новом файле `internal/api/suggest_points.go`: чтение query `term`, валидация длины через `error_mapper`, вызов репозитория, сборка обёртки; подключить маршрут в `internal/api/route.go` с `RequireClientRole` + `RequireAnyUserPermission(driver, passenger, director, admin)` (design D5) и проверить `go build ./...`
- [x] 2.4 Написать интеграционные тесты endpoint'а в `internal/api/api_test/` по сценариям спека: кириллица в нижнем/верхнем регистре, `И` находит `Иркутск`+`Ижевск`, дедупликация повторяющегося пункта, `LIMIT` 10 при 25 совпадениях, пункты `canceled`-поездки и `draft`-заявки в подсказке, спецсимволы `%`/`_` буквально, 401 без claims, 403 без `sharetrip_*`-прав, по одной проверке 200 для всех четырёх ролей; тесты обязаны использовать типизированные обёртки и закрывать тело ответа (AGENTS.md) — проверка `go test ./internal/api/api_test/ -run 'TestSuggestPoints' -count=1 -v`
- [x] 2.5 Задокументировать новый endpoint в `README.md` (маршрут, параметр `term`, формат ответа, доступ) и сверить описание с реальным поведением тестов из 2.4

## 3. Интеграционные проверки

- [x] 3.1 `make lint` — 0 issues (включая `paralleltest`: в `api_test` он исключён, в пакете репозитория новые тесты должны быть параллельными)
- [x] 3.2 `make test` — все пакеты зелёные (требует запущенный Docker: `api_test` поднимает postgres:16)

## Workflow follow-up

- После ревью: `/opsx-apply` для реализации, затем `/opsx-archive` для синхронизации delta-спеки в `openspec/specs/point-suggestions/spec.md`.
- Изменение фронтенда (`mat-autocomplete` в `route-form.component.ts` / фильтрах списка) — отдельный change в репозитории `job4j_go_share_trip_front`, вне границ этого изменения.
