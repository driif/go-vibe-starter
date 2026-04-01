
BIN_DIR := bin
APP_NAME := app

.PHONY: build run clean gen-oapi

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./

run: build
	./$(BIN_DIR)/$(APP_NAME) run

clean:
	@rm -rf $(BIN_DIR)

gen-oapi:
	@oapi-codegen -generate types -o "internal/api/openapi_types.gen.go" -package "api" "oapi/openapi.yaml"
	oapi-codegen -generate chi-server,spec -o "internal/api/openapi_server.gen.go" -package "api" "oapi/openapi.yaml"