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
	$(GO) test -v $(GO_PKG)

# Генерация отчёта о покрытии в формате HTML
.PHONY: coverage cover
coverage cover:
	$(GO) test -coverprofile=coverage.out $(GO_PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: file://$(shell pwd)/coverage.html"

# Вывод покрытия в терминал (опционально)
.PHONY: cover-report
cover-report:
	$(GO) test -cover $(GO_PKG)

# Проверка кода с помощью golangci-lint
.PHONY: lint
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "❌ golangci-lint is not installed. Please install it:"; \
		echo "   https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	golangci-lint run

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
