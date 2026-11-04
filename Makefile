# Variables
BINARY_NAME=spx-tf-api
CMD_DIR=./cmd/spx-tf-api
CLIENT_SWAGGER=docs/client/swagger.json
CLIENT_PKG=pkg/client/spx
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all build run test swagger clean help client-gen client-test

all: build

## build: Build the binary
build:
	go build -v -o $(BINARY_NAME) $(CMD_DIR)/main.go

## client-test: Build the client to ensure it compiles
client-test:
	go build -v ./pkg/client/spx/...

## run: Run the application
run: build
	@if [ -z "$(SPX_API_ENDPOINT)" ]; then \
		echo "Error: SPX_API_ENDPOINT is not set. Use: SPX_API_ENDPOINT=https://console.mon-superphenix.net/api make run"; \
		exit 1; \
	fi
	./$(BINARY_NAME)

## test: Run tests
test:
	go test -v ./...

## swagger: Generate swagger documentation
swagger:
	go run github.com/swaggo/swag/cmd/swag init -g $(CMD_DIR)/main.go

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf docs/

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
