.PHONY: help migrate-up migrate-down migrate-create docker-up docker-down docker-build run

# Переменные
MIGRATE_DSN := postgres://user:pass@localhost:5555/pr_review_new?sslmode=disable
MIGRATIONS_PATH := internal/db/migrations

help:
	@echo "Available commands:"
	@echo "  make migrate-up          - Run all pending migrations"
	@echo "  make migrate-down        - Rollback last migration"
	@echo "  make migrate-create NAME - Create new migration (NAME=add_users_table)"
	@echo "  make docker-build        - Build Docker image"
	@echo "  make docker-up           - Start services with docker-compose"
	@echo "  make docker-down         - Stop services"
	@echo "  make run                 - Build and run application"

# Миграции
migrate-up:
	@echo "Running migrations up..."
	migrate -path $(MIGRATIONS_PATH) -database "$(MIGRATE_DSN)" up

migrate-down:
	@echo "Rolling back last migration..."
	migrate -path $(MIGRATIONS_PATH) -database "$(MIGRATE_DSN)" down

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

# Docker
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

# Приложение
run: docker-build docker-up
	@echo "✅ Services started. Logs:"
	@make docker-logs

.DEFAULT_GOAL := help
