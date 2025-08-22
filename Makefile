# Updated Makefile for SQLC-based API
BINARY_NAME=citystat-api
GOPROXY_URL=https://proxy.golang.org,direct

# Database operations
.PHONY: db-migrate
db-migrate:
	@echo ">> Running database migrations..."
	migrate -path sql/migrations -database ${DATABASE_URL} up

.PHONY: db-migrate-down
db-migrate-down:
	@echo ">> Rolling back database migrations..."
	migrate -path sql/migrations -database ${DATABASE_URL} down 1

.PHONY: db-reset
db-reset:
	@echo ">> Resetting database..."
	migrate -path sql/migrations -database ${DATABASE_URL} drop
	migrate -path sql/migrations -database ${DATABASE_URL} up

# SQLC operations
.PHONY: sqlc-generate
sqlc-generate:
	@echo ">> Generating SQLC code..."
	sqlc generate

.PHONY: sqlc-verify
sqlc-verify:
	@echo ">> Verifying SQLC queries..."
	sqlc verify

# Build operations
.PHONY: build
build: sqlc-generate
	@echo ">> Building $(BINARY_NAME)..."
	go build -tags netgo -ldflags '-s -w' -o $(BINARY_NAME) ./cmd/api

.PHONY: run
run: build
	@echo ">> Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

.PHONY: dev
dev: sqlc-generate
	@echo ">> Running in development mode..."
	go run ./cmd/api

# Testing
.PHONY: test
test:
	@echo ">> Running tests..."
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo ">> Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Dependencies
.PHONY: tidy
tidy:
	@echo ">> Tidying dependencies..."
	GOPROXY=$(GOPROXY_URL) go mod tidy

.PHONY: vendor
vendor: tidy
	@echo ">> Vendoring dependencies..."
	GOPROXY=$(GOPROXY_URL) go mod vendor

# Linting and formatting
.PHONY: fmt
fmt:
	@echo ">> Formatting code..."
	go fmt ./...

.PHONY: lint
lint:
	@echo ">> Running linter..."
	golangci-lint run

# Clean up
.PHONY: clean
clean:
	@echo ">> Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf vendor/
	rm -f coverage.out

# Docker operations
.PHONY: docker-build
docker-build:
	@echo ">> Building Docker image..."
	docker build -t $(BINARY_NAME) .

.PHONY: docker-run
docker-run:
	@echo ">> Running Docker container..."
	docker run --env-file .env -p 3333:3333 $(BINARY_NAME)

# Development setup
.PHONY: setup
setup:
	@echo ">> Setting up development environment..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: init-db
init-db:
	@echo ">> Initializing database..."
	createdb citystat || true
	make db-migrate

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build           Build the binary"
	@echo "  run             Build and run the application"
	@echo "  dev             Run in development mode"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage"
	@echo "  sqlc-generate   Generate SQLC code"
	@echo "  sqlc-verify     Verify SQLC queries"
	@echo "  db-migrate      Run database migrations"
	@echo "  db-migrate-down Rollback one migration"
	@echo "  db-reset        Reset database"
	@echo "  fmt             Format code"
	@echo "  lint            Run linter"
	@echo "  clean           Clean build artifacts"
	@echo "  setup           Setup development environment"
	@echo "  docker-build    Build Docker image"
	@echo "  docker-run      Run Docker container"