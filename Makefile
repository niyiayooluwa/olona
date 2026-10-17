.PHONY: all run build test swag tidy

# Server paths
SERVER_DIR := server

all: run

run:
	@echo "Starting server..."
	cd $(SERVER_DIR) && go run cmd/api/main.go

build:
	@echo "Building server..."
	cd $(SERVER_DIR) && go build -o bin/api cmd/api/main.go

test:
	@echo "Running tests..."
	cd $(SERVER_DIR) && go test ./... -v

swag:
	@echo "Generating Swagger documentation..."
	cd $(SERVER_DIR) && swag init -g cmd/api/main.go -d .

tidy:
	@echo "Tidying go modules..."
	cd $(SERVER_DIR) && go mod tidy
