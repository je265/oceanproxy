# OceanProxy Makefile

# Variables
APP_NAME := oceanproxy
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Build variables
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
BUILD_DIR := ./build
BIN_DIR := ./bin
BINARY_NAME := oceanproxy
BINARY_PATH := $(BIN_DIR)/$(BINARY_NAME)

# Docker variables
DOCKER_IMAGE := oceanproxy
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY := your-registry.com

# Directories
SRC_DIRS := ./cmd ./internal
CONFIG_DIR := ./configs
SCRIPT_DIR := ./scripts

.PHONY: help build clean test test-coverage lint fmt vet deps tidy run dev docker docker-build docker-push deploy setup install uninstall

# Default target
all: clean fmt vet test build

# Help
help: ## Display this help screen
	@echo "OceanProxy Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the application
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/server/main.go
	@echo "Build complete: $(BINARY_PATH)"

# Build for multiple platforms
build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server/main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/server/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server/main.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server/main.go
	@echo "Multi-platform build complete"

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)/bin
	@echo "Clean complete"

# Run tests
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -short ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"

# Run integration tests
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./test/integration/...

# Lint the code
lint: ## Lint the code
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format the code
fmt: ## Format the code
	@echo "Formatting code..."
	$(GOFMT) -s -w $(SRC_DIRS)

# Vet the code
vet: ## Vet the code
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Download dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOGET) -d ./...

# Tidy dependencies
tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Run the application
run: build ## Run the application
	@echo "Running $(APP_NAME)..."
	$(BINARY_PATH)

# Run in development mode
dev: ## Run in development mode
	@echo "Running in development mode..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	$(GOCMD) run ./cmd/server/main.go

# Watch and reload during development
watch: ## Watch for changes and reload
	@echo "Watching for changes..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install it with: go install github.com/cosmtrek/air@latest"; \
		make dev; \
	fi

# Docker targets
docker: docker-build ## Build Docker image

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f build/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-v /var/log/oceanproxy:/var/log/oceanproxy \
		-v /etc/3proxy:/etc/3proxy \
		--env-file .env \
		$(DOCKER_IMAGE):latest

docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

# Compose targets
compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose -f build/docker-compose.yml up -d

compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	docker-compose -f build/docker-compose.yml down

compose-logs: ## View logs from docker-compose
	docker-compose -f build/docker-compose.yml logs -f

# Setup and installation
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@make setup-dirs
	@make setup-config
	@make deps
	@make tidy
	@echo "Development environment setup complete"

setup-dirs: ## Create necessary directories
	@echo "Creating directories..."
	@mkdir -p /var/log/oceanproxy
	@mkdir -p /etc/3proxy
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(BUILD_DIR)

setup-config: ## Setup configuration files
	@echo "Setting up configuration..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@if [ ! -f configs/config.local.yaml ]; then cp configs/config.yaml configs/config.local.yaml; fi

setup-dev: setup ## Setup development environment with additional tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest

# System installation
install: build ## Install the application system-wide
	@echo "Installing $(APP_NAME)..."
	@sudo cp $(BINARY_PATH) /usr/local/bin/
	@sudo cp configs/config.yaml /etc/oceanproxy/
	@sudo cp deployments/systemd/oceanproxy.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable oceanproxy
	@echo "Installation complete"

uninstall: ## Uninstall the application
	@echo "Uninstalling $(APP_NAME)..."
	@sudo systemctl stop oceanproxy || true
	@sudo systemctl disable oceanproxy || true
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -rf /etc/oceanproxy
	@sudo rm -f /etc/systemd/system/oceanproxy.service
	@sudo systemctl daemon-reload
	@echo "Uninstallation complete"

# Database migrations (if using SQL database in the future)
migrate-up: ## Run database migrations up
	@echo "Running database migrations up..."
	# Add migration command here

migrate-down: ## Run database migrations down
	@echo "Running database migrations down..."
	# Add migration command here

# API documentation
docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/server/main.go -o ./api; \
	else \
		echo "swag not installed. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# Deployment
deploy: ## Deploy the application
	@echo "Deploying $(APP_NAME)..."
	@./deployments/scripts/deploy.sh

# Health checks
health: ## Check application health
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Health check failed"

# Logs
logs: ## View application logs
	@echo "Viewing logs..."
	@tail -f /var/log/oceanproxy/app.log

# Backup
backup: ## Backup data
	@echo "Creating backup..."
	@./scripts/system/backup.sh

# Restore
restore: ## Restore from backup
	@echo "Restoring from backup..."
	@./scripts/system/restore.sh

# Performance benchmarks
bench: ## Run performance benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Security scan
security: ## Run security scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Version information
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Show current status
status: ## Show application status
	@echo "Application Status:"
	@systemctl is-active oceanproxy || echo "Not running"
	@echo ""
	@echo "Port Usage:"
	@netstat -tlnp | grep :8080 || echo "Port 8080 not in use"
	@echo ""
	@echo "Recent logs:"
	@tail -5 /var/log/oceanproxy/app.log 2>/dev/null || echo "No logs found"