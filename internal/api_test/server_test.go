package api_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"job4j_go_share_trip/internal/api"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/metrics"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	testCtx       context.Context
	testDB        *sql.DB
	testPool      *pgxpool.Pool
	testApp       *fiber.App
	testContainer *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	testCtx = context.Background()

	var err error

	testContainer, err = postgres.Run(
		testCtx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	dsn, err := testContainer.ConnectionString(
		testCtx,
		"sslmode=disable",
	)
	if err != nil {
		log.Fatalf("get connection string: %v", err)
	}

	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open sql db: %v", err)
	}

	waitReady(testDB)

	if err = goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}

	if err = goose.Up(testDB, "../../migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	testPool, err = pgxpool.New(testCtx, dsn)
	if err != nil {
		log.Fatalf("create pgx pool: %v", err)
	}

	registry := prometheus.NewRegistry()
	metrix := metrics.New(registry)

	server := api.NewServer(testPool, registry, metrix)

	testApp = fiber.New()

	// ✅ Исправленный мок: парсит Subject из токена
	testApp.Use(func(c *fiber.Ctx) error {
		token := c.Get("X-Refresh-Token")
		if token == "" {
			return c.Next()
		}

		// Парсим токен, чтобы получить Subject
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token payload",
			})
		}

		var claimsMap map[string]interface{}
		if err := json.Unmarshal(payload, &claimsMap); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		subject, ok := claimsMap["sub"].(string)
		if !ok || subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing subject in token",
			})
		}

		// Создаём claims с Subject из токена
		claims := &middleware.KeycloakClaims{
			Subject:           subject,
			PreferredUsername: "testuser",
			Email:             "test@example.com",
			ResourceAccess: map[string]struct {
				Roles []string `json:"roles"`
			}{
				"sharetrip-api": {
					Roles: []string{"client"},
				},
			},
		}
		c.Locals(middleware.KeycloakClaimsKey, claims)
		return c.Next()
	})

	server.Route(testApp.Group(""))

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}
	if testDB != nil {
		_ = testDB.Close()
	}
	if testContainer != nil {
		_ = testContainer.Terminate(testCtx)
	}

	os.Exit(code)
}

func waitReady(db *sql.DB) {
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		err := db.PingContext(ctx)
		cancel()

		if err == nil {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Fatalf("database is not ready after timeout")
}