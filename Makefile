# Переменные
GO := go
GO_PKG := ./...
DOCKER_COMPOSE := docker compose --project-directory ./deploy

DB_USER ?= postgres
DB_PASSWORD ?= password
DB_NAME ?= share_trip
DB_HOST ?= localhost
DB_PORT ?= 6543
DB_SSLMODE ?= disable
DB_DSN = user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) host=$(DB_HOST) port=$(DB_PORT) sslmode=$(DB_SSLMODE)

run:
	go1.25.5 run cmd/sharetrip/main.go

# Цель по умолчанию
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  test        - Run all tests"
	@echo "  coverage    - Run tests and generate HTML coverage report"
	@echo "  cover       - Alias for coverage"
	@echo "  lint        - Run golangci-lint"
	@echo "  all         - Run lint, tests and coverage"
	@echo "  help        - Show this help"

# Запуск всех тестов
.PHONY: test
test:
	$(GO) test -v ./internal/...

# --- ПОКРЫТИЕ (ИСПРАВЛЕНО) ---

# Генерация отчёта о покрытии
.PHONY: coverage
coverage:
	@echo "Generating coverage..."
	@mkdir -p reports
	GOTOOLCHAIN=local go test -coverprofile=reports/coverage.out \
		-coverpkg=./internal/... \
		./internal/api_test/... \
		./internal/domain/trip/...
	@echo ""
	@echo "Coverage summary:"
	GOTOOLCHAIN=local go tool cover -func=reports/coverage.out | grep total

# Открыть покрытие в браузере
.PHONY: coverage-html
coverage-html: coverage
	GOTOOLCHAIN=local go tool cover -html=reports/coverage.out

# Вывод покрытия в терминал (исправлено!)
.PHONY: coverage-report
coverage-report:
	@echo "Running tests with coverage..."
	GOTOOLCHAIN=local go test -coverprofile=reports/coverage.out \
		-coverpkg=./internal/... \
		./internal/api_test/... \
		./internal/domain/trip/...
	@echo ""
	@echo "Coverage report:"
	GOTOOLCHAIN=local go tool cover -func=reports/coverage.out

# Только итоговый процент
.PHONY: coverage-total
coverage-total:
	GOTOOLCHAIN=local go test -coverprofile=reports/coverage.out \
		-coverpkg=./internal/... \
		./internal/api_test/... \
		./internal/domain/trip/...
	GOTOOLCHAIN=local go tool cover -func=reports/coverage.out | grep total

# --- ЛИНТЕР ---

# Проверка кода с помощью golangci-lint
.PHONY: lint
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "❌ golangci-lint is not installed. Please install it:"; \
		echo "   https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	golangci-lint run ./...

tidy:
	go1.25.5 mod tidy

# Запуск всех проверок
.PHONY: all
all: lint test coverage

build: 
	$(DOCKER_COMPOSE) build

up: 
	$(DOCKER_COMPOSE) up -d

down: 
	$(DOCKER_COMPOSE) down

migrate-up: ## Применить все миграции
	goose -dir ./migrations postgres "$(DB_DSN)" up

migrate-down: ## Откатить одну миграцию
	goose -dir ./migrations postgres "$(DB_DSN)" down

migrate-status: ## Показать статус миграций
	goose -dir ./migrations postgres "$(DB_DSN)" status

migrate-reset: ## Откатить все миграции
	goose -dir ./migrations postgres "$(DB_DSN)" reset

migrate-create: ## Создать новую миграцию (использовать: make migrate-create NAME=имя_миграции)
	goose -dir ./migrations create $(NAME) sql
