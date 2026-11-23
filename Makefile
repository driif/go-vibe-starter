APP_NAME := app
BIN := bin/$(APP_NAME)
TOOLS := $(CURDIR)/bin/tools

# the only pins for these four tools; `make tools` installs them into bin/tools
BUF_VERSION := v1.75.0
SQLC_VERSION := v1.31.1
GOLANGCI_LINT_VERSION := v2.15.0
GOVULNCHECK_VERSION := v1.10.0

VERSION ?= $(shell git describe --tags --match 'v*' --always --dirty 2>/dev/null || echo dev)
IMAGE ?= $(APP_NAME):dev
LDFLAGS = -X $(shell go list -m)/cmd.version=$(VERSION)
GEN_PATHS := go.mod go.sum internal/gen internal/db/gen
KC_USER ?= alice
KEYCLOAK_ISSUER_URL ?= http://localhost:8181/realms/app

STAMPS := $(TOOLS)/.buf-$(BUF_VERSION) $(TOOLS)/.sqlc-$(SQLC_VERSION) \
          $(TOOLS)/.golangci-lint-$(GOLANGCI_LINT_VERSION) $(TOOLS)/.govulncheck-$(GOVULNCHECK_VERSION)

.DEFAULT_GOAL := help

.PHONY: help init tools tools-check build run test cover lint fmt gen gen-proto sqlc drift check \
        vuln image image-smoke migrate-up migrate-down migrate-status dev-up dev-down keycloak-up \
        smoke token labels sync-skills clean

help: ## list every target
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

init: ## bootstrap a new project from this template
	bash scripts/init.sh

# $(call install-tool,name,package,version)
define install-tool
	GOBIN=$(TOOLS) go install $(2)@$(3)
	@rm -f $(TOOLS)/.$(1)-*
	@touch $(TOOLS)/.$(1)-$(3)
endef

tools: ## install the pinned buf, sqlc, golangci-lint and govulncheck into bin/tools
	@mkdir -p $(TOOLS)
	$(call install-tool,buf,github.com/bufbuild/buf/cmd/buf,$(BUF_VERSION))
	$(call install-tool,sqlc,github.com/sqlc-dev/sqlc/cmd/sqlc,$(SQLC_VERSION))
	$(call install-tool,golangci-lint,github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	$(call install-tool,govulncheck,golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

tools-check: ## fail when a pinned tool is missing or outdated
	@for stamp in $(STAMPS); do \
		test -f $$stamp || { echo "tools missing or outdated: run 'make tools'"; exit 1; }; \
	done

build: ## compile the server to bin/
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .

.env.local:
	cp .env.example .env.local

run: build .env.local ## build and start the server
	./$(BIN) run

test: ## go test -race ./...
	go test -race ./...

cover: ## coverage profile and total
	go test -race -coverprofile=profile.cov ./...
	@go tool cover -func=profile.cov | tail -n 1

lint: tools-check ## golangci-lint, buf lint, buf format check
	$(TOOLS)/golangci-lint run
	$(TOOLS)/buf lint
	$(TOOLS)/buf format -d --exit-code

fmt: tools-check ## format Go and proto files
	$(TOOLS)/golangci-lint fmt
	$(TOOLS)/buf format -w

gen: gen-proto sqlc ## regenerate all generated code

gen-proto: tools-check ## regenerate internal/gen from proto/
	$(TOOLS)/buf generate

sqlc: tools-check ## regenerate internal/db/gen from sql/
	@rm -rf internal/db/gen
	@if ls sql/queries/*.sql >/dev/null 2>&1; then \
		$(TOOLS)/sqlc generate; \
	else \
		echo "sqlc: no query files, skipped"; \
	fi

drift: ## fail when generated code or go.mod is stale
	@tmp=$$(mktemp -d); \
	for p in $(GEN_PATHS); do \
		if [ -e $$p ]; then mkdir -p $$tmp/$$(dirname $$p) && cp -R $$p $$tmp/$$p; fi; \
	done; \
	$(MAKE) gen && go mod tidy || { rm -rf $$tmp; exit 1; }; \
	stale=; \
	for p in $(GEN_PATHS); do \
		if [ -e $$p ] || [ -e $$tmp/$$p ]; then \
			diff -r $$tmp/$$p $$p >/dev/null 2>&1 || stale="$$stale $$p"; \
		fi; \
	done; \
	rm -rf $$tmp; \
	if [ -n "$$stale" ]; then \
		echo "stale:$$stale"; \
		echo "generated files or go.mod are stale: commit the output of 'make gen' and 'go mod tidy'"; \
		exit 1; \
	fi

check: tools-check ## the gate: drift, lint, test, skills mirror
	@$(MAKE) drift
	@$(MAKE) lint
	@$(MAKE) test
	scripts/sync-skills.sh --check

vuln: tools-check ## govulncheck ./...
	$(TOOLS)/govulncheck ./...

image: ## build the container image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

image-smoke: image ## run the image and probe it
	scripts/image-smoke.sh $(IMAGE) $(APP_NAME) $(VERSION)

migrate-up: build .env.local ## apply pending migrations
	./$(BIN) db migrate up

migrate-down: build .env.local ## roll back one migration
	./$(BIN) db migrate down

migrate-status: build .env.local ## list applied and pending migrations
	./$(BIN) db migrate status

dev-up: ## start Postgres
	docker compose up -d --wait postgres

dev-down: ## stop the compose stack, keeping volumes
	docker compose down

keycloak-up: ## start a fresh Keycloak
	docker compose up -d --wait --force-recreate keycloak

smoke: keycloak-up ## run the smoke tests against compose Keycloak
	go test -race -count=1 -tags smoke -run '^TestSmoke' ./internal/test/

token: ## print an access token for KC_USER from compose Keycloak
	@curl -fsS -d grant_type=password -d client_id=app-test -d username=$(KC_USER) -d password=password \
		$(KEYCLOAK_ISSUER_URL)/protocol/openid-connect/token | sed -E 's/.*"access_token":"([^"]+)".*/\1/'

labels: ## create the GitHub issue labels the skills use
	scripts/setup-labels.sh

sync-skills: ## mirror .claude/skills to agents/skills
	scripts/sync-skills.sh

clean: ## remove build output
	rm -rf bin profile.cov
