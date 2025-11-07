# Go parameters
BINARY_NAME=secretapi
BINARY_UNIX=$(BINARY_NAME)
BINARY_CLI_NAME=secretcli
BINARY_CLI_UNIX=$(BINARY_NAME)

# Docker parameters
DOCKER_IMAGE_NAME=secretapi
DOCKER_TAG=latest
DOCKER_HUB_REPO=smallwat3r/secretapi

.PHONY: all build run test clean fmt lint docker-build docker-run docker-stop docker-release help

help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: build

frontend-build: ## Build the Preact frontend application
	@echo "Building frontend..."
	@cd web && npm ci && npm run build

build: frontend-build ## Build the Go application
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_UNIX) ./cmd/server

build-cli: ## Build the Go CLI application
	@echo "Building $(BINARY_CLI_NAME)..."
	@go build -o $(BINARY_CLI_UNIX) ./cmd/cli

run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_UNIX)

test: ## Run all Go tests
	@echo "Running tests..."
	@go test ./...

clean: ## Remove compiled binaries, build cache, and frontend artifacts
	@echo "Cleaning..."
	@if [ -f $(BINARY_UNIX) ] ; then rm $(BINARY_UNIX); fi
	@if [ -d "web/node_modules" ] ; then rm -rf web/node_modules; fi
	@if [ -d "web/static/dist" ] ; then rm -rf web/static/dist; fi

fmt: ## Format Go source files
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter on Go code
	@echo "Linting code..."
	@golangci-lint run

docker-build: ## Build Docker image for the application
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

docker-run: ## Run the application stack with Docker Compose
	@echo "Running with Docker Compose..."
	@docker compose up --build

docker-stop: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker compose down

docker-release: docker-build ## Tag and push Docker image to Docker Hub
	@echo "Releasing Docker image to Docker Hub..."
	@docker tag $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) $(DOCKER_HUB_REPO):$(DOCKER_TAG)
	@docker push $(DOCKER_HUB_REPO):$(DOCKER_TAG)
