.PHONY: dev build test migrate migrate-down seed lint up down web

# Start development server with hot reload
dev:
	air -c .air.toml

# Build production binary
build:
	go build -o bin/server ./cmd/server

# Run all tests
test:
	go test ./... -v -cover

# Run database migrations
migrate:
	migrate -database "$(DATABASE_URL)" -path migrations up

# Rollback last migration
migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down 1

# Seed database with sample data (1000 pages for search testing)
seed:
	go run ./cmd/seed/main.go

# Run linter
lint:
	golangci-lint run ./...

# Start all services via Docker Compose
up:
	docker compose up -d

# Stop all services
down:
	docker compose down

# Run Flutter web app
web:
	cd web && flutter run -d chrome
