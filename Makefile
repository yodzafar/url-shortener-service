.DEFAULT_GOAL := help
.PHONY: help tools run build clean test test-int coverage lint fmt vet wire swagger mocks generate \
		migrate-up migrate-down migrate-status migrate-create docker-up docker-down docker-logs check

APP 	:= api
BIN_DIR := bin
BIN 	:= $(BIN_DIR)/$(APP)
PKG		:= ./...
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

ifneq (,$(wildcard .env))
	include .env
	export
endif

# ---------- help ----------
help: ## command list
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

tools: ## set CLI
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/goforj/wire/cmd/wire@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v3@v3.8.0
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
	go install golang.org/x/tools/cmd/goimports@latest

# ---------- run / build ----------
run: ## run server
	go run ./cmd/$(APP)

build: ## build binary
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/$(APP)

clean: ## clean binary and coverage
	rm -rf $(BIN_DIR) cover.out coverage.html

# ---------- lint / test ----------
fmt: ## gofmt + go imports
	gofmt -s -w .
	goimports -w .

vet: ## go vet
	go vet $(PKG)

lint: ## golang ci-lint
	golangci-lint run

test: ## Unit tests (race + cover)
	go test -race -cover $(PKG)

test-int: ## integration test (Docker)
	go test -race -tags=integration -count=1 $(PKG)

coverage: ## HTML coverage report
	go test -coverprofile=cover.out $(PKG)
	go tool cover -html=cover.out -o coverage.html
	@echo "open coverage.html"

check: fmt vet lint test

# ---------- codegen ----------
sqlc: ## sqlc codegen (SQL -> Go)
	sqlc generate

wire: ## Wire DI codegen
	wire ./internal/app

swagger: ## Swagger docs
	swag init -g main.go -d cmd/api,internal -o api/swagger --parseDependency --parseInternal

mocks: ## Mockery (.mockery.yaml)
	mockery

generate: sqlc wire swagger mocks ## run all codegen

# ---------- migration ----------
migrate-up: ## accept all migrations
	goose up

migrate-down: ## revert last migration
	goose down

migrate-status: ## migration status
	goose status

migrate-create: ## create new migration file
ifndef name
	$(error name is required: make migrate-create name=create_users)
endif
	goose -dir migrations create $(name) sql

# ---------- docker ----------
docker-up: ## Postgres + Redis
	docker compose up -d

docker-down: ## Stop container
	docker compose down

docker-logs: ## Docker logs
	docker compose logs -f


