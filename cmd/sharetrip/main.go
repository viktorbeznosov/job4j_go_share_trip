package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"job4j_go_share_trip/config"
	"job4j_go_share_trip/internal/api"
	"job4j_go_share_trip/internal/app"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/observability/tracing"
	"job4j_go_share_trip/internal/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	ctx := context.Background()

	// Загружаем .env файл
	if err := godotenv.Load("./.env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

    cfg := config.GetAppConfig()

	storageCfg := storage.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

    logger, logFile, err := app.NewLogger()
    if err != nil {
        panic(err)
    }
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("failed to close log file: %v", err)
		}
	}()

    tp, err := tracing.NewProvider(ctx, tracing.Config{
        ServiceName:    cfg.Tracing.ServiceName,
        ServiceVersion: cfg.Tracing.ServiceVersion,
        Environment:    cfg.Tracing.Environment,
        Endpoint:       cfg.Tracing.Endpoint,
    })
    if err != nil {
        logger.Error("init tracing failed", "error", err)
        os.Exit(1)
    }

    defer func() {
        shutdownCtx, cancel := context.WithTimeout(
            context.Background(),
            5 * time.Second,
        )
        defer cancel()

        if err := tp.Shutdown(shutdownCtx); err != nil {
            logger.Error("shutdown tracing failed", "error", err)
        }
    }()

	pool, err := storage.NewPool(ctx, storageCfg.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

    registry := prometheus.NewRegistry()
    m := metrics.New(registry)

	server := api.NewServer(pool, registry, m)

	app := fiber.New()

	app.Use(middleware.Correlation(logger))
	app.Use(middleware.NewHTTPMetricsMiddleware(m))

    app.Use(middleware.KeycloakRefreshTokenMiddleware(
        middleware.KeycloakConfig{
            Issuer:       cfg.Keycloak.Issuer,
            ClientID:     cfg.Keycloak.ClientID,
            ClientSecret: cfg.Keycloak.ClientSecret,
        },
    ))

	server.Route(app.Group("/api"))

	err = app.Listen(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
}
