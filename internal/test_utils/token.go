package testutils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// TestTokenOptions опции для генерации тестового токена
type TestTokenOptions struct {
	UserID      string
	Username    string
	Email       string
	Roles       []string
	Permissions []string
	ClientID    string
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
	if opts.Permissions == nil {
		opts.Permissions = []string{"sharetrip_driver"}
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
		"sharetrip": map[string]interface{}{
			"permissions": opts.Permissions,
		},
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	return fmt.Sprintf("%s.%s.", header, payload)
}

func GenerateAdminToken(userID, username, email string) string {
	return GenerateTestTokenWithOptions(TestTokenOptions{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       []string{"client"},
		Permissions: []string{"sharetrip_admin"},
	})
}

func GeneratePassengerToken(userID, username, email string) string {
	return GenerateTestTokenWithOptions(TestTokenOptions{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       []string{"client"},
		Permissions: []string{"sharetrip_passenger"},
	})
}

func GenerateDirectorToken(userID, username, email string) string {
	return GenerateTestTokenWithOptions(TestTokenOptions{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       []string{"client"},
		Permissions: []string{"sharetrip_director"},
	})
}
