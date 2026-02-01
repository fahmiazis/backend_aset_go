# Variables
APP_NAME=backend-go
DOCKER_COMPOSE=docker-compose
GO=go
GOOSE=goose

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

.PHONY: help build run stop clean test migrate-up migrate-down migrate-status docker-build docker-up docker-down docker-logs

# Default target
help:
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  $(YELLOW)make build$(NC)         - Build the Go application"
	@echo "  $(YELLOW)make run$(NC)           - Run the application locally"
	@echo "  $(YELLOW)make test$(NC)          - Run tests"
	@echo "  $(YELLOW)make clean$(NC)         - Clean build artifacts"
	@echo ""
	@echo "$(GREEN)Migration commands:$(NC)"
	@echo "  $(YELLOW)make migrate-up$(NC)    - Run all migrations"
	@echo "  $(YELLOW)make migrate-down$(NC)  - Rollback last migration"
	@echo "  $(YELLOW)make migrate-status$(NC) - Check migration status"
	@echo "  $(YELLOW)make migrate-create NAME=<name>$(NC) - Create new migration"
	@echo ""
	@echo "$(GREEN)Docker commands:$(NC)"
	@echo "  $(YELLOW)make docker-build$(NC)  - Build Docker image"
	@echo "  $(YELLOW)make docker-up$(NC)     - Start all services with Docker Compose"
	@echo "  $(YELLOW)make docker-down$(NC)   - Stop all services"
	@echo "  $(YELLOW)make docker-logs$(NC)   - View logs"
	@echo "  $(YELLOW)make docker-restart$(NC) - Restart all services"
	@echo "  $(YELLOW)make docker-clean$(NC)  - Remove all containers and volumes"

# Build the application
build:
	@echo "$(GREEN)Building application...$(NC)"
	$(GO) build -o bin/$(APP_NAME) .

# Run the application locally
run:
	@echo "$(GREEN)Running application...$(NC)"
	$(GO) run main.go

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GO) test -v ./...

# Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -rf bin/
	$(GO) clean

# Install dependencies
deps:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy

# Migration commands
migrate-up:
	@echo "$(GREEN)Running migrations...$(NC)"
	@. ./.env && $(GOOSE) -dir migrations mysql "$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true" up

migrate-down:
	@echo "$(YELLOW)Rolling back migration...$(NC)"
	@. ./.env && $(GOOSE) -dir migrations mysql "$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true" down

migrate-status:
	@echo "$(GREEN)Migration status:$(NC)"
	@. ./.env && $(GOOSE) -dir migrations mysql "$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true" status

migrate-create:
	@echo "$(GREEN)Creating migration: $(NAME)$(NC)"
	$(GOOSE) -dir migrations create $(NAME) sql

# Docker commands
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	$(DOCKER_COMPOSE) build

docker-up:
	@echo "$(GREEN)Starting services...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Services started!$(NC)"
	@echo "$(YELLOW)Run 'make docker-logs' to view logs$(NC)"

docker-down:
	@echo "$(YELLOW)Stopping services...$(NC)"
	$(DOCKER_COMPOSE) down

docker-logs:
	@echo "$(GREEN)Viewing logs...$(NC)"
	$(DOCKER_COMPOSE) logs -f

docker-restart:
	@echo "$(YELLOW)Restarting services...$(NC)"
	$(DOCKER_COMPOSE) restart

docker-clean:
	@echo "$(YELLOW)Removing all containers and volumes...$(NC)"
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

# Docker migration commands
docker-migrate-up:
	@echo "$(GREEN)Running migrations in Docker...$(NC)"
	$(DOCKER_COMPOSE) exec app sh -c "goose -dir migrations mysql \"$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true\" up"

docker-migrate-down:
	@echo "$(YELLOW)Rolling back migration in Docker...$(NC)"
	$(DOCKER_COMPOSE) exec app sh -c "goose -dir migrations mysql \"$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true\" down"

docker-migrate-status:
	@echo "$(GREEN)Migration status in Docker:$(NC)"
	$(DOCKER_COMPOSE) exec app sh -c "goose -dir migrations mysql \"$$DB_USER:$$DB_PASSWORD@tcp($$DB_HOST:$$DB_PORT)/$$DB_NAME?parseTime=true\" status"

# Development workflow
dev:
	@echo "$(GREEN)Starting development environment...$(NC)"
	$(DOCKER_COMPOSE) up mysql -d
	@echo "$(YELLOW)Waiting for MySQL to be ready...$(NC)"
	@sleep 10
	@$(MAKE) migrate-up
	@$(MAKE) run

# Production deployment
deploy:
	@echo "$(GREEN)Deploying to production...$(NC)"
	$(DOCKER_COMPOSE) down
	$(DOCKER_COMPOSE) build --no-cache
	$(DOCKER_COMPOSE) up -d
	@sleep 15
	@$(MAKE) docker-migrate-up
	@echo "$(GREEN)Deployment complete!$(NC)"