# AGENTS.md

Правила для AI-агентов, работающих с проектом ShareTrip.

## Проверка моделей в тестах

### Главное правило

**В тестах проверяй модель целиком, а не отдельные поля через `map[string]interface{}`.**

### ❌ Неправильно

```go
var response api.Response
err = json.Unmarshal(respBody, &response)
require.NoError(t, err)

data, ok := response.Data.(map[string]interface{})
require.True(t, ok)

require.NotEmpty(t, data["id"])
require.Equal(t, payload.FromPoint, data["fromPoint"])
require.Equal(t, payload.Seats, int(data["seats"].(float64)))
```

### ✅ Правильно

```go
var response api.CreateTripResponseWrapper
err = json.Unmarshal(respBody, &response)
require.NoError(t, err)

require.Equal(t, "Success", response.Status)
require.NotEmpty(t, response.Data.ID)
require.Equal(t, userID.String(), response.Data.DriverID)
require.Equal(t, payload.FromPoint, response.Data.FromPoint)
require.Equal(t, payload.ToPoint, response.Data.ToPoint)
require.Equal(t, payload.Seats, response.Data.Seats)
require.Equal(t, "draft", response.Data.Status)
require.NotZero(t, response.Data.CreatedAt)
```

### Обёртки для ответов

Для каждого типа ответа создавай обёртку в `internal/api/dto.go`:

```go
type CreateTripResponseWrapper struct {
    Status  string             `json:"status"`
    Message string             `json:"message,omitempty"`
    Data    CreateTripResponse `json:"data"`
    Error   string             `json:"error,omitempty"`
}

type MoveTripDraftToPublishResponseWrapper struct {
    Status  string                         `json:"status"`
    Message string                         `json:"message,omitempty"`
    Data    MoveTripDraftToPublishResponse `json:"data"`
    Error   string                         `json:"error,omitempty"`
}

type MoveTripFromPublishToStartedResponseWrapper struct {
    Status  string                               `json:"status"`
    Message string                               `json:"message,omitempty"`
    Data    MoveTripFromPublishToStartedResponse `json:"data"`
    Error   string                               `json:"error,omitempty"`
}

type ErrorResponseWrapper struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

### Проверка ошибок

```go
var response api.ErrorResponseWrapper
err = json.NewDecoder(resp.Body).Decode(&response)
require.NoError(t, err)

require.Equal(t, "Error", response.Status)
require.Contains(t, response.Message, "invalid trip status")
```

### Закрытие тела ответа

```go
defer func() {
    if err := resp.Body.Close(); err != nil {
        t.Errorf("failed to close response body: %v", err)
    }
}()
```

### Почему так

1. **Типобезопасность** — поля известны на этапе компиляции, нельзя опечататься в имени.
2. **Читаемость** — сразу видно, какие поля проверяются.
3. **Рефакторинг** — при изменении модели компилятор укажет на все места.
4. **Единый стиль** — все тесты проверяют модели одинаково.

### Запрещено

- ❌ `response.Data.(map[string]interface{})`
- ❌ Приведение типов вручную (`int(data["seats"].(float64))`)
- ❌ Проверка полей через `data["field"]`
- ❌ Игнорирование ошибок `resp.Body.Close()`

### Обязательные проверки

Перед завершением задачи запусти:

```bash
make lint
make test
```