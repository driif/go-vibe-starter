
BIN_DIR := bin
APP_NAME := app

.PHONY: build run clean

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./

run: build
	./$(BIN_DIR)/$(APP_NAME) run

clean:
	@rm -rf $(BIN_DIR)
