BIN_DIR := bin
APP_NAME := app
COVER_PROFILE := profile.cov

# pinned so every machine and CI runner generates byte-identical output
SQLC_VERSION := v1.30.0
OAPI_CODEGEN_VERSION := v2.6.0
GOLANGCI_LINT_VERSION := v2.12.2
GOOSE_VERSION := v3.27.0

.PHONY: help init build run clean test cover lint fmt gen gen-oapi sqlc \
        migrate-up db-up db-down labels sync-skills tools

help:
	@echo "go-vibe-starter targets:"
	@echo "  init        bootstrap a new project from this template"
	@echo "  build       compile the server to $(BIN_DIR)/$(APP_NAME)"
	@echo "  run         build and start the server"
	@echo "  test        go test -race ./..."
	@echo "  cover       coverage profile ($(COVER_PROFILE)) + summary"
	@echo "  lint        golangci-lint run"
	@echo "  fmt         gofmt + goimports"
	@echo "  gen         gen-oapi + sqlc"
	@echo "  gen-oapi    regenerate the API layer from oapi/openapi.yaml"
	@echo "  sqlc        regenerate internal/db/gen from sql/queries"
	@echo "  migrate-up  apply pending database migrations"
	@echo "  db-up       start the docker compose stack"
	@echo "  db-down     stop the docker compose stack"
	@echo "  labels      create the GitHub issue label vocabulary"
	@echo "  sync-skills mirror .claude/skills to agents/skills"
	@echo "  tools       install the pinned code generation and lint tools"

init:
	./scripts/init.sh

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./

run: build
	./$(BIN_DIR)/$(APP_NAME) run

clean:
	@rm -rf $(BIN_DIR) $(COVER_PROFILE)

test:
	go test -race ./...

cover:
	go test -race -coverprofile=$(COVER_PROFILE) -covermode=atomic ./...
	@go tool cover -func=$(COVER_PROFILE) | tail -n 1

lint:
	golangci-lint run

fmt:
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then goimports -w .; else echo "goimports not installed, skipping — run make tools"; fi

gen: gen-oapi sqlc

gen-oapi:
	@oapi-codegen -generate types -o "internal/api/openapi_types.gen.go" -package "api" "oapi/openapi.yaml"
	oapi-codegen -generate chi-server,spec -o "internal/api/openapi_server.gen.go" -package "api" "oapi/openapi.yaml"

sqlc:
	@if ls sql/queries/*.sql >/dev/null 2>&1; then \
		sqlc generate; \
	else \
		echo "no .sql files in sql/queries, skipping sqlc"; \
	fi

migrate-up: build
	./$(BIN_DIR)/$(APP_NAME) db migrate up

db-up:
	docker compose up -d

db-down:
	docker compose down

labels:
	./scripts/setup-labels.sh

sync-skills:
	./scripts/sync-skills.sh

tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	go install golang.org/x/tools/cmd/goimports@latest
