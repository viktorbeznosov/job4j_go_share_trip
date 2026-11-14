package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/api"
	requestEntity "job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/business/trip/entity"
	testutils "job4j_go_share_trip/internal/test_utils"
)

// suggestPointsRequest выполняет GET /points с опциональным токеном.
// Пустой токен — запрос вовсе без заголовка X-Refresh-Token (анонимный
// клиент). query — уже закодированная строка после "?" или "" без параметров.
func suggestPointsRequest(t *testing.T, token, query string) *http.Response {
	t.Helper()

	path := "/points"
	if query != "" {
		path += "?" + query
	}

	req, err := http.NewRequest(http.MethodGet, path, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("X-Refresh-Token", token)
	}

	resp, err := testApp.Test(req, -1)
	require.NoError(t, err)
	return resp
}

// suggestTerm кодирует term так, как это сделал бы браузер.
func suggestTerm(term string) string {
	return url.Values{"term": {term}}.Encode()
}

// suggestItems возвращает data.items типизированной обёртки. Тело ответа
// закрывается через closeBody (проверка ошибки внутри).
func suggestItems(t *testing.T, token, query string) []string {
	t.Helper()

	resp := suggestPointsRequest(t, token, query)
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body api.SuggestPointsResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Success", body.Status)
	return body.Data.Items
}

// insertSuggestTrip вставляет поездку с произвольными пунктами и статусом
// напрямую в БД (хелперы fixtures_test.go жёстко прописывают пункты)
// и убирает её по завершении теста.
func insertSuggestTrip(t *testing.T, from, to string, status entity.Status) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := testPool.Exec(testCtx, `
		INSERT INTO trips
			(id, driver_id, from_point, to_point, departure_time, seats, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 4, $6, NOW())`,
		id, uuid.New(), from, to, time.Now().Add(24*time.Hour), string(status),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, CleanupTestData(testCtx, testPool, &TestData{TripID: id}))
	})
	return id
}

// insertSuggestRequest вставляет заявку с произвольными пунктами и статусом
// и убирает её по завершении теста.
func insertSuggestRequest(t *testing.T, from, to string, status requestEntity.Status) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := testPool.Exec(testCtx, `
		INSERT INTO trip_requests
			(id, passenger_id, from_point, to_point, departure_time, seats, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 1, $6, NOW())`,
		id, uuid.New(), from, to, time.Now().Add(24*time.Hour), string(status),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, CleanupRequest(testCtx, testPool, id))
	})
	return id
}

func suggestDriverToken() string {
	return testutils.GenerateTestToken(uuid.New().String(), "driver", "driver@example.com")
}

// suggestNoPermissionToken — валидные claims с ролью client, но без ни
// одного права sharetrip_* (ожидаемый 403).
func suggestNoPermissionToken() string {
	return testutils.GenerateTestTokenWithOptions(testutils.TestTokenOptions{
		UserID:      uuid.New().String(),
		Username:    "noperm",
		Email:       "noperm@example.com",
		Roles:       []string{"client"},
		Permissions: []string{},
	})
}

// Нижний регистр в запросе находит запись в верхнем регистре.
func TestSuggestPoints_LowerQueryFindsUpperPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Москва", "Тула", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("мос"))
	require.Contains(t, items, "Москва")
}

// Верхний регистр в запросе находит запись в нижнем регистре.
func TestSuggestPoints_UpperQueryFindsLowerPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "москва", "тула", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("МОС"))
	require.Contains(t, items, "москва")
}

// Одна буква находит несколько городов.
func TestSuggestPoints_SingleLetterFindsSeveralCities(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Иркутск", "Москва", entity.StatusDraft)
	insertSuggestTrip(t, "Ижевск", "Казань", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("И"))
	require.Contains(t, items, "Иркутск")
	require.Contains(t, items, "Ижевск")
}

// Нет совпадений — 200 с пустым срезом (не nil: в JSON это [], а не null).
// Срез не-nil гарантирует и пустая таблица: make([]string, 0) в
// SuggestPoints не зависит от количества строк в БД.
func TestSuggestPoints_NoMatches(t *testing.T) {
	token := suggestDriverToken()

	items := suggestItems(t, token, suggestTerm("какойтогородкоторогонет"))
	require.NotNil(t, items)
	require.Empty(t, items)
}

// Пункт только из поездки попадает в подсказку.
func TestSuggestPoints_TripOnlyPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Тверь", "Москва", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("твер"))
	require.Contains(t, items, "Тверь")
}

// Пункт только из заявки попадает в подсказку.
func TestSuggestPoints_RequestOnlyPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestRequest(t, "Тула", "Калуга", requestEntity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("калуг"))
	require.Contains(t, items, "Калуга")
}

// Пункт отменённой поездки доступен в подсказке.
func TestSuggestPoints_CanceledTripPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Самара", "Москва", entity.StatusCanceled)

	items := suggestItems(t, token, suggestTerm("самар"))
	require.Contains(t, items, "Самара")
}

// Пункт черновика заявки доступен в подсказке.
func TestSuggestPoints_DraftRequestPoint(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestRequest(t, "Рязань", "Тула", requestEntity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("рязан"))
	require.Contains(t, items, "Рязань")
}

// Повторяющийся пункт выдаётся ровно один раз.
func TestSuggestPoints_DuplicatesRemoved(t *testing.T) {
	token := suggestDriverToken()
	for i := 0; i < 3; i++ {
		insertSuggestTrip(t, "Москва", "Омск", entity.StatusDraft)
	}
	insertSuggestRequest(t, "Омск", "Москва", requestEntity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("москва"))
	require.Equal(t, []string{"Москва"}, items)
}

// Больше десяти совпадений — ровно 10, в алфавитном порядке.
func TestSuggestPoints_LimitTen(t *testing.T) {
	token := suggestDriverToken()
	for i := 1; i <= 25; i++ {
		insertSuggestTrip(t, fmt.Sprintf("ТестПункт%02d", i), "ТестНазначение", entity.StatusDraft)
	}

	items := suggestItems(t, token, suggestTerm("тестпункт"))
	require.Len(t, items, 10)

	want := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		want = append(want, fmt.Sprintf("ТестПункт%02d", i))
	}
	require.Equal(t, want, items)
}

// Меньше десяти совпадений — столько, сколько нашлось.
func TestSuggestPoints_FewerThanTen(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "ТройкаПункт1", "Назначение", entity.StatusDraft)
	insertSuggestTrip(t, "ТройкаПункт2", "Назначение", entity.StatusDraft)
	insertSuggestTrip(t, "ТройкаПункт3", "Назначение", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("тройка"))
	require.Len(t, items, 3)
}

// Без параметра term — начало словаря (до 10 пунктов).
func TestSuggestPoints_AbsentTerm(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Заречный", "Отрог", entity.StatusDraft)

	items := suggestItems(t, token, "")
	require.NotEmpty(t, items)
	require.LessOrEqual(t, len(items), 10)
	require.Contains(t, items, "Заречный")
}

// Пустой term — начало словаря (до 10 пунктов).
func TestSuggestPoints_EmptyTerm(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Заречный", "Отрог", entity.StatusDraft)

	items := suggestItems(t, token, "term=")
	require.NotEmpty(t, items)
	require.LessOrEqual(t, len(items), 10)
	require.Contains(t, items, "Заречный")
}

// Детерминированный алфавитный порядок.
func TestSuggestPoints_AlphabeticalOrder(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Воронеж", "Омск", entity.StatusDraft)
	insertSuggestTrip(t, "Владивосток", "Омск", entity.StatusDraft)
	insertSuggestTrip(t, "Волгоград", "Омск", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("во"))
	require.Equal(t, []string{"Владивосток", "Волгоград", "Воронеж"}, items)
}

// Процент в запросе не матчит всё: без экранирования вернулись бы все
// пункты системы, а остаётся пустой список (пунктов с литеральным % нет).
func TestSuggestPoints_PercentIsLiteral(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Москва", "Казань", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("%"))
	require.NotContains(t, items, "Москва")
	require.Empty(t, items)
}

// Подчёркивание в запросе не матчит произвольный символ.
func TestSuggestPoints_UnderscoreIsLiteral(t *testing.T) {
	token := suggestDriverToken()
	insertSuggestTrip(t, "Москва", "Казань", entity.StatusDraft)

	items := suggestItems(t, token, suggestTerm("м_сква"))
	require.NotContains(t, items, "Москва")
	require.Empty(t, items)
}

// Слишком длинный term — 400.
func TestSuggestPoints_TermTooLong(t *testing.T) {
	token := suggestDriverToken()

	resp := suggestPointsRequest(t, token, suggestTerm(strings.Repeat("а", 257)))
	defer closeBody(t, resp)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body api.ErrorResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Error", body.Status)
	require.Contains(t, body.Message, "at most 256 characters")
}

// Все четыре роли получают подсказку.
func TestSuggestPoints_AllRolesAllowed(t *testing.T) {
	insertSuggestTrip(t, "РолевойПункт", "Назначение", entity.StatusDraft)

	cases := []struct {
		name  string
		token string
	}{
		{"driver", testutils.GenerateTestToken(uuid.New().String(), "driver", "d@example.com")},
		{"passenger", testutils.GeneratePassengerToken(uuid.New().String(), "p", "p@example.com")},
		{"director", testutils.GenerateDirectorToken(uuid.New().String(), "dir", "dir@example.com")},
		{"admin", testutils.GenerateAdminToken(uuid.New().String(), "adm", "adm@example.com")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := suggestItems(t, tc.token, suggestTerm("ролевой"))
			require.Contains(t, items, "РолевойПункт")
		})
	}
}

// Неаутентифицированный запрос — 401.
func TestSuggestPoints_Unauthorized(t *testing.T) {
	resp := suggestPointsRequest(t, "", suggestTerm("мос"))
	defer closeBody(t, resp)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Токен без ни одного права sharetrip_* — 403.
func TestSuggestPoints_ForbiddenWithoutPermissions(t *testing.T) {
	resp := suggestPointsRequest(t, suggestNoPermissionToken(), suggestTerm("мос"))
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}
