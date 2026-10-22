# ShareTrip

Сервис совместных поездок. Водитель создаёт поездку, директор её апрувит, пассажир подаёт заявку, стороны подтверждают участие, поездка стартует и завершается.

Документ описывает **фактическую** логику текущей реализации и последние изменения по сервисам:

- `job4j_go_share_trip` — API поездок и заявок;
- `job4j_go_share_trip_contracts` — договоры компании и водителя;
- `job4j_go_share_trip_notification` — уведомления;
- `job4j_go_share_trip_front` — Angular UI.

## Сервисы и границы

| Сервис | Владеет | Не владеет |
|---|---|---|
| **ShareTrip** | агрегаты `Trip` и `Request`, outbox, воркеры | договором, доставкой уведомлений |
| **Contracts** | договоры компании/водителя, флаги услуг | поездками и заявками |
| **Notification** | таблица `notifications`, inbox `processed_events` | статусами поездок и контрактов |
| **Frontend** | экраны ролей, вызовы API | бизнес-правилами |

У каждого сервиса своя PostgreSQL. Связь между ними:

- ShareTrip → Contracts: синхронный HTTP `CheckService` перед публикацией и стартом поездки;
- ShareTrip → Kafka `trip.events`: асинхронные доменные события через outbox;
- Contracts и Notification читают `trip.events` независимо.

```text
Клиент (Keycloak JWT)
        │
        ▼
   ShareTrip API
        │  HTTP CheckService
        ├──────────────────► Contracts
        │
        │  INSERT outbox_events  (одна транзакция с агрегатом)
        ▼
 Outbox Publisher  ──►  Kafka topic trip.events
                              │
                ┌─────────────┴─────────────┐
                ▼                           ▼
           Contracts                    Notification
     (draft / active / terminated)   (уведомления по event_id)
```

## Роли и доступ

Авторизация: заголовок `X-Refresh-Token` → Keycloak refresh → access token. Права читаются из claim `sharetrip.permissions`.

| Permission | Кто | Что может |
|---|---|---|
| `sharetrip_driver` | водитель | создавать/публиковать/стартовать/завершать/отменять свои поездки; подтверждать и отклонять заявки на свою поездку |
| `sharetrip_passenger` | пассажир | создавать/публиковать заявки, прикреплять к поездке, отклонять свою заявку |
| `sharetrip_director` | директор | апрувить поездки в `preapproved`; в списках видит **только** `preapproved` |
| `sharetrip_admin` | админ | полный список поездок и заявок, фильтр `driverId` / `passengerId` |

Роль клиента Keycloak для API: `client`.

### Видимость поездок

- **Водитель** — свои поездки.
- **Пассажир** — опубликованные поездки **или** поездки, к которым прикреплена его заявка.
- **Директор** — только `preapproved`.
- **Админ** — все.

### Видимость заявок

- **Пассажир** — свои заявки.
- **Водитель** — заявки на свои поездки в статусах `pre_agreed`, `agreed`, `started`, `canceled`, `completed`.
- **Админ** — все. Фильтр `passengerId` только для админа.

## Поездка (Trip)

Статусы в БД:

`draft` → `preapproved` → `published` → `started` → `completed`  
параллельно: `canceled` (кроме `started` и уже завершённых).

```text
[*] --> draft
draft --> preapproved     : водитель PUT /trip/move_to_publish
draft --> canceled        : водитель PUT /trip/move_to_canceled
preapproved --> published : директор PUT /trip/approve
preapproved --> canceled
published --> started     : водитель PUT /trip/move_to_started
published --> canceled
started --> completed     : водитель PUT /trip/move_to_completed
started -x canceled       : запрещено, HTTP 409
completed --> [*]
canceled --> [*]
```

### Создание

`POST /api/trip/` (водитель).

Тело: `fromPoint`, `toPoint`, `departureTime` (`2006-01-02 15:04`, не в прошлом), `seats` (1–10), **`companyId`**.

Результат: поездка `draft`. В той же транзакции в outbox пишется **`trip_created`** (`trip_id`, `driver_id`, `company_id`). Publisher отправляет его в Kafka.

### Публикация водителем

`PUT /api/trip/move_to_publish`.

1. Синхронно Contracts: услуга `trip_publish` доступна компании.
2. Только владелец, статус `draft`.
3. Статус → `preapproved`.
4. Outbox: `trip_preapproved`.

Это **не** публикация для пассажиров. Пассажиры видят поездку после апрува директора.

### Апрув директором

`PUT /api/trip/approve`.

1. Статус должен быть `preapproved`.
2. Статус → `published`.
3. Outbox: `trip_published` (`driver_id`, `company_id`).

Именно это событие активирует контракт водителя.

### Старт

`PUT /api/trip/move_to_started`.

1. Contracts: услуга `trip_start`.
2. Только владелец, статус `published` (повторный вызов, если уже `started` — 204).
3. Поездка → `started`, событие `trip_started`.
4. Прикреплённые заявки:
   - `agreed` → `started` + `request_started`;
   - `pre_agreed` отцепляются обратно в `published` (`trip_id = null`).

Отдельного `PUT /request/move_to_started` нет: заявки стартуют вместе с поездкой.

### Завершение

`PUT /api/trip/move_to_completed`.

Только владелец, статус `started`. Поездка → `completed`. Заявки `started` → `completed` + `request_completed`. Событие `trip_completed`.

### Отмена

`PUT /api/trip/move_to_canceled`.

Только владелец. **Стартовавшую поездку отменить нельзя** (`409`, `Started trip cannot be canceled`).

Если отмена разрешена:

- поездка → `canceled`, событие `trip_canceled` с `passenger_ids` и `request_ids`;
- прикреплённые заявки отцепляются в `published` (`trip_id = null`).

## Заявка (Request)

Статусы:

`draft` → `published` → `pre_agreed` → `agreed` → `started` → `completed`  
отклонение: `pre_agreed` → `published`  
истечение: `draft|published|pre_agreed|agreed` → `canceled`

```text
[*] --> draft
draft --> published          : пассажир PUT /request/move_to_publish
published --> pre_agreed     : пассажир PUT /request/move_request_to_agreed
pre_agreed --> agreed        : водитель PUT /request/confirm confirm=true
pre_agreed --> published     : водитель или пассажир confirm=false
agreed --> started           : вместе со стартом поездки
started --> completed        : вместе с завершением поездки
(draft|published|pre_agreed|agreed) --> canceled : воркер expiry
```

Заявка живёт отдельно от поездки: маршрут и дата свои. К поездке она привязывается только в `pre_agreed` / `agreed` / `started`.

### Создание и публикация

`POST /api/request/` — пассажир, статус `draft`, событие `request_created`.  
`PUT /api/request/move_to_publish` — владелец заявки, `draft` → `published`, событие `request_published`.

### Запрос на поездку (preconfirm)

`PUT /api/request/move_request_to_agreed` (`requestId`, `tripId`).

- поездка должна быть `published`;
- заявка — `published` или уже `pre_agreed` (можно переприкрепить к другой поездке);
- сумма мест `pre_agreed` + `agreed` на поездке не превышает `seats`;
- заявка → `pre_agreed`, `trip_id` заполнен.

Событие **`request_preconfirmed`**: `passenger_id`, **`driver_id`**, `trip_id`.  
Notification шлёт уведомление **водителю**.

### Подтверждение / отклонение

`PUT /api/request/confirm` (`requestId`, `confirm`).

- `confirm=true` — только водитель поездки, заявка `pre_agreed` → `agreed`. Событие **`request_confirmed`** с `passenger_id`. Notification → **пассажиру**.
- `confirm=false` — водитель поездки или пассажир-владелец заявки. Заявка → `published`, `trip_id = null`. Событие **`request_rejected`** с `passenger_id`. Notification → **пассажиру**.

## Kafka: outbox → trip.events

ShareTrip никогда не пишет в Kafka внутри HTTP-запроса. В одной транзакции с агрегатом:

1. меняется статус;
2. пишется history;
3. `INSERT INTO outbox_events`.

Publisher раз в секунду: `FOR UPDATE SKIP LOCKED` → Kafka → `sent`. При ошибке Kafka событие остаётся `pending` / `failed` и уйдёт позже.

Topic: `trip.events` (env `KAFKA_TRIP_EVENTS_TOPIC`). Ключ: `trip_id` или `request_id`.

### События поездки

| event_type | Когда | Главные поля | Contracts | Notification |
|---|---|---|---|---|
| `trip_created` | создание поездки | `driver_id`, `company_id` | создать контракт **draft** | не слушает |
| `trip_preapproved` | водитель отправил на апрув | `driver_id`, `company_id` | игнорирует | уведомление водителю |
| `trip_published` | директор апрувил | `driver_id`, `company_id` | контракт → **active** | уведомление водителю |
| `trip_started` | старт | `driver_id` | — | уведомление водителю |
| `trip_canceled` | отмена | `passenger_ids`, `request_ids` | контракт → **terminated** | уведомления пассажирам |
| `trip_completed` | завершение | `driver_id` | контракт → **terminated** | — |
| `trip_publish_notificated` | воркер матчинга | `passenger_ids`, маршрут | — | уведомления пассажирам |

### События заявки

| event_type | Когда | Главные поля | Notification |
|---|---|---|---|
| `request_created` / `request_published` | создание / публикация | `passenger_id` | не слушает |
| `request_preconfirmed` | пассажир запросил поездку | `driver_id`, `passenger_id`, `trip_id` | **водителю** |
| `request_confirmed` | водитель принял | `passenger_id` | **пассажиру** |
| `request_rejected` | отклонение | `passenger_id` | **пассажиру** |
| `request_started` | вместе со стартом поездки | `passenger_id` | не слушает |
| `request_completed` | вместе с завершением поездки | `passenger_id` | пассажиру |
| `request_expired` | воркер expiry | `passenger_id` | пассажиру |

Consumers используют inbox `processed_events` по `event_id`: повтор Kafka не создаёт второй контракт и второе уведомление.

## Contracts

Синхронный HTTP (ShareTrip спрашивает перед командой):

```text
GET /api/companies/{companyId}/services/{service}/availability
```

Услуги: `trip_publish`, `trip_start`. Если `available=false` или сервис недоступен — поездка не меняется (`403`).

Асинхронно по Kafka:

1. **`trip_created`** — если у водителя нет open-контракта (`draft` / `active` / `suspended`), создаётся **draft** (`start_date` пустой). Второй open не создаётся.
2. **`trip_published`** (апрув директора) — open draft активируется, ставится `start_date`. Если open нет — создаётся сразу `active`.
3. **`trip_preapproved` не трогает контракт.**
4. **`trip_completed` / `trip_canceled`** — open-контракт → `terminated`, пишется `end_date`.

Цепочка для водителя:

```text
создал поездку  →  контракт draft
директор апрувил →  контракт active
завершил/отменил →  контракт terminated
```

## Notification

Читает `trip.events`, пишет строку в `notifications` (`recipient_id`, `type`, `payload`).

| Событие | Кому |
|---|---|
| `trip_preapproved`, `trip_published`, `trip_started` | водитель |
| `trip_canceled`, `trip_publish_notificated` | пассажиры из `passenger_ids` |
| `request_preconfirmed` | водитель (`driver_id`) |
| `request_confirmed`, `request_rejected`, `request_completed`, `request_expired` | пассажир (`passenger_id`) |

`trip_created` notification не обрабатывает.

## Фоновые воркеры (ShareTrip)

Интервал: 1 минута.

### TripPublishNotifier

Для каждой `published` поездки, у которой ещё есть свободные места, ищет `published` заявки с тем же маршрутом и датой выезда, ещё не уведомлённые по этой поездке (`trip_publish_notified`). Пишет `trip_publish_notificated` с `passenger_ids`.

### RequestExpiry

Заявки `draft` / `published` / `pre_agreed` / `agreed`, у которых дата выезда уже в прошлом (UTC-день), переводятся в `canceled`, `trip_id` сбрасывается, событие `request_expired`.

## HTTP API

Префикс `/api`. Ответ: `{ "status": "Success"|"Error", "data": ..., "message": ... }`.

### Поездки

| Метод | Путь | Кто | Действие |
|---|---|---|---|
| POST | `/trip/` | водитель | создать `draft` |
| PUT | `/trip/move_to_publish` | водитель | `draft` → `preapproved` |
| PUT | `/trip/approve` | директор | `preapproved` → `published` |
| PUT | `/trip/move_to_started` | водитель | `published` → `started` |
| PUT | `/trip/move_to_completed` | водитель | `started` → `completed` |
| PUT | `/trip/move_to_canceled` | водитель | отмена, кроме `started` |
| GET | `/trip/` | driver/passenger/director/admin | список с видимостью роли |
| GET | `/trip/:uuid` | то же | карточка с видимостью роли |

Мутации кроме create принимают `tripId` и `companyId`. Create — `companyId` в теле вместе с маршрутом.

### Заявки

| Метод | Путь | Кто | Действие |
|---|---|---|---|
| POST | `/request/` | пассажир | создать `draft` |
| PUT | `/request/move_to_publish` | пассажир | `draft` → `published` |
| PUT | `/request/move_request_to_agreed` | пассажир | прикрепить к поездке |
| PUT | `/request/confirm` | водитель / пассажир | принять или отклонить |
| GET | `/request/` | passenger/driver/admin | список с видимостью роли |
| GET | `/request/:uuid` | то же | карточка |

Служебные: `GET /api/ready`, `GET /api/metrics`.

## Надёжная доставка событий

Пишущий шаг (смена статуса + событие) — локальная транзакция PostgreSQL. Kafka вне транзакции, поэтому:

- **Outbox** на стороне ShareTrip: не потерять событие, если Kafka недоступна в момент HTTP.
- **Inbox** на стороне Contracts и Notification: одно и то же `event_id` применяется один раз.

Retry HTTP-команды не заменяет outbox: если поездка уже `published`, повтор апрува вернёт 204 и больше не пишет событие.

Слепой retry пишущих вызовов (создать уведомление, списать оплату) опасен. Читающий `CheckService` можно повторять: повтор не меняет чужое состояние.

Расследование «уведомление не пришло»:

`request_id` → contract check → статус агрегата → `outbox_events` (`pending`/`sent`/`failed`) → Kafka lag → inbox `processed_events` → строка в `notifications`.

## Frontend

Минимальные экраны под роли. Создание поездки добавляет `companyId` из конфига приложения.

На карточке поездки водителя:

- `draft` — «На публикацию»;
- `published` — «Начать поездку»;
- `started` — «Завершить поездку» (кнопки «Отменить» нет);
- не `canceled` / не `completed` / не `started` — «Отменить поездку».

Директор видит «Approve» только для `preapproved`.

## Последние изменения

Зафиксировано в коде и тестах:

1. Контракт создаётся **после создания поездки**, сразу в **`draft`**, не `active`. Для этого ShareTrip публикует Kafka-событие `trip_created` (`company_id` обязателен на `POST /trip`).
2. После апрува директора (`trip_published`) контракт переходит в **`active`**. Событие `trip_preapproved` контракт больше не активирует.
3. В `request_preconfirmed` в payload добавлен **`driver_id`**. Notification шлёт уведомление водителю: «пассажир запросил поездку».
4. В `request_confirmed` и `request_rejected` в payload есть **`passenger_id`**. Notification шлёт уведомление пассажиру: заявка принята или отклонена.
5. Поездку в статусе **`started` нельзя отменить** (API `409`, кнопка на UI скрыта).

Ранее по тому же контуру:

- отмена поездки отцепляет заявки и кладёт `passenger_ids` в `trip_canceled`;
- старт/завершение поездки синхронизируют статусы заявок, отдельного `move_to_started` у заявки нет;
- воркеры `TripPublishNotifier` и `RequestExpiry`;
- директор видит и апрувит только `preapproved`;
- видимость списков поездок и заявок по ролям.

## Проверки

```bash
make lint
make test
```

То же в репозиториях `job4j_go_share_trip_contracts` и `job4j_go_share_trip_notification`.
