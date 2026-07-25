package testutils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// TestTokenOptions опции для генерации тестового токена
type TestTokenOptions struct {
	UserID   string
	Username string
	Email    string
	Roles    []string // Роли пользователя
	ClientID string   // ID клиента (по умолчанию "sharetrip-api")
}

// GenerateTestToken создаёт тестовый JWT токен (без подписи) для тестов
// По умолчанию добавляет роль "client" (требуется для доступа к API)
func GenerateTestToken(userID, username, email string) string {
	return GenerateTestTokenWithOptions(TestTokenOptions{
		UserID:   userID,
		Username: username,
		Email:    email,
		Roles:    []string{"client"}, // ✅ Изменено с "user" на "client"
		ClientID: "sharetrip-api",
	})
}

// GenerateTestTokenWithOptions создаёт тестовый токен с кастомными опциями
func GenerateTestTokenWithOptions(opts TestTokenOptions) string {
	if opts.ClientID == "" {
		opts.ClientID = "sharetrip-api"
	}
	if opts.Roles == nil {
		opts.Roles = []string{"client"}
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))

	claims := map[string]interface{}{
		"sub":                opts.UserID,
		"preferred_username": opts.Username,
		"email":              opts.Email,
		"azp":                opts.ClientID,
		"resource_access": map[string]interface{}{
			opts.ClientID: map[string]interface{}{
				"roles": opts.Roles,
			},
		},
		"realm_access": map[string]interface{}{
			"roles": opts.Roles,
		},
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	return fmt.Sprintf("%s.%s.", header, payload)
}

// GenerateAdminToken создаёт токен с правами администратора
func GenerateAdminToken(userID, username, email string) string {
	return GenerateTestTokenWithOptions(TestTokenOptions{
		UserID:   userID,
		Username: username,
		Email:    email,
		Roles:    []string{"client"},
	})
}