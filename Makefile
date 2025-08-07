# OceanProxy Makefile - UPDATED for fixed structure

# Variables
APP_NAME := oceanproxy
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

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

# Configuration
CONFIG_DIR := /etc/oceanproxy
LOG_DIR := /var/log/oceanproxy
DATA_DIR := /var/lib/oceanproxy

.PHONY: help build clean test test-coverage lint fmt vet deps tidy run dev install uninstall restart logs status

# Default target
all: clean fmt vet test build

# Help
help: ## Display this help screen
	@echo "🌊 OceanProxy Build System - FIXED"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the application
build: ## Build the application
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/server/main.go
	@echo "✅ Build complete: $(BINARY_PATH)"

# Build CLI tool
build-cli: ## Build the CLI tool
	@echo "🔨 Building CLI tool..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/oceanproxy-cli ./cmd/cli/main.go
	@echo "✅ CLI build complete: $(BIN_DIR)/oceanproxy-cli"

# Build both
build-all: build build-cli ## Build both server and CLI

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	@echo "✅ Clean complete"

# Run tests
test: ## Run tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race -short ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "📊 Coverage report: $(BUILD_DIR)/coverage.html"

# Lint the code
lint: ## Lint the code
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format the code
fmt: ## Format the code
	@echo "🎨 Formatting code..."
	$(GOFMT) -s -w ./cmd ./internal ./pkg

# Vet the code
vet: ## Vet the code
	@echo "🔍 Vetting code..."
	$(GOCMD) vet ./...

# Download dependencies
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	$(GOGET) -d ./...

# Tidy dependencies
tidy: ## Tidy dependencies
	@echo "🗂️  Tidying dependencies..."
	$(GOMOD) tidy

# Run the application
run: build ## Run the application
	@echo "🚀 Running $(APP_NAME)..."
	$(BINARY_PATH)

# Run in development mode
dev: ## Run in development mode
	@echo "🔧 Running in development mode..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	$(GOCMD) run ./cmd/server/main.go

# Quick test authentication
test-auth: ## Test authentication system
	@echo "🔐 Testing authentication..."
	@chmod +x ./test_authentication.sh 2>/dev/null || echo "Creating test script..."
	@./test_authentication.sh || echo "Run 'make build && make restart' first"

# Install the application system-wide
install: build ## Install the application system-wide
	@echo "📥 Installing $(APP_NAME)..."
	@sudo mkdir -p $(CONFIG_DIR) $(LOG_DIR) $(DATA_DIR)
	@sudo cp $(BINARY_PATH) /usr/local/bin/
	@if [ -f configs/config.yaml ]; then sudo cp configs/config.yaml $(CONFIG_DIR)/; fi
	@if [ -f .env.example ]; then sudo cp .env.example $(CONFIG_DIR)/oceanproxy.env; fi
	@if [ -f deployments/systemd/oceanproxy.service ]; then \
		sudo cp deployments/systemd/oceanproxy.service /etc/systemd/system/; \
		sudo systemctl daemon-reload; \
		sudo systemctl enable oceanproxy; \
	fi
	@echo "✅ Installation complete"
	@echo "⚠️  Edit $(CONFIG_DIR)/oceanproxy.env with your configuration"

# Uninstall the application
uninstall: ## Uninstall the application
	@echo "🗑️  Uninstalling $(APP_NAME)..."
	@sudo systemctl stop oceanproxy 2>/dev/null || true
	@sudo systemctl disable oceanproxy 2>/dev/null || true
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -f /etc/systemd/system/oceanproxy.service
	@sudo systemctl daemon-reload
	@echo "✅ Uninstallation complete"

# Service management
restart: ## Restart the service
	@echo "🔄 Restarting OceanProxy service..."
	@sudo systemctl restart oceanproxy
	@sleep 2
	@make status

start: ## Start the service
	@echo "▶️  Starting OceanProxy service..."
	@sudo systemctl start oceanproxy
	@sleep 2
	@make status

stop: ## Stop the service
	@echo "⏹️  Stopping OceanProxy service..."
	@sudo systemctl stop oceanproxy

status: ## Check service status
	@echo "📊 OceanProxy Service Status:"
	@echo "=========================="
	@if systemctl is-active --quiet oceanproxy; then \
		echo "✅ Service: Running"; \
	else \
		echo "❌ Service: Stopped"; \
	fi
	@echo ""
	@echo "🌐 Network Status:"
	@if netstat -tlnp 2>/dev/null | grep -q ":8080 "; then \
		echo "✅ API Port (8080): Listening"; \
	else \
		echo "❌ API Port (8080): Not listening"; \
	fi
	@if netstat -tlnp 2>/dev/null | grep -q ":1337 "; then \
		echo "✅ USA Proxy Port (1337): Listening"; \
	else \
		echo "⚠️  USA Proxy Port (1337): Not listening"; \
	fi
	@echo ""
	@echo "⚙️  Quick Commands:"
	@echo "   Logs:    make logs"
	@echo "   Test:    make test-auth"
	@echo "   Restart: make restart"

# View logs
logs: ## View application logs
	@echo "📋 Viewing OceanProxy logs (Ctrl+C to exit)..."
	@sudo journalctl -u oceanproxy -f --no-pager

# View recent logs
logs-recent: ## View recent logs
	@echo "📋 Recent OceanProxy logs:"
	@sudo journalctl -u oceanproxy -n 50 --no-pager

# Health check
health: ## Check application health
	@echo "🏥 Checking application health..."
	@if curl -s -f http://localhost:8080/health >/dev/null; then \
		echo "✅ Health check passed"; \
		curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health; \
	else \
		echo "❌ Health check failed"; \
		echo "   Is the service running? Check: make status"; \
	fi

# Fix permissions
fix-permissions: ## Fix file permissions
	@echo "🔧 Fixing permissions..."
	@sudo chown -R oceanproxy:oceanproxy $(LOG_DIR) $(DATA_DIR) 2>/dev/null || true
	@sudo chmod 755 $(LOG_DIR) $(DATA_DIR) 2>/dev/null || true
	@sudo chmod 600 $(CONFIG_DIR)/oceanproxy.env 2>/dev/null || true
	@echo "✅ Permissions fixed"

# Update and rebuild
update: ## Update, rebuild and restart
	@echo "🔄 Updating OceanProxy..."
	@git pull 2>/dev/null || echo "Not a git repository"
	@make clean
	@make build
	@if systemctl is-active --quiet oceanproxy; then \
		make restart; \
	else \
		echo "Service not running, use 'make start' to start it"; \
	fi

# Development setup
setup-dev: ## Setup development environment
	@echo "🔧 Setting up development environment..."
	@make deps
	@make tidy
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@echo "✅ Development environment setup complete"
	@echo "📝 Next steps:"
	@echo "   1. Edit .env with your configuration"
	@echo "   2. Run: make dev"

# Production deployment
deploy: build ## Deploy to production
	@echo "🚀 Deploying to production..."
	@make install
	@make fix-permissions
	@make start
	@sleep 3
	@make health
	@echo "✅ Deployment complete"

# Backup data
backup: ## Backup data and configuration
	@echo "💾 Creating backup..."
	@mkdir -p ./backups
	@sudo cp -r $(CONFIG_DIR) ./backups/config-$(shell date +%Y%m%d_%H%M%S) 2>/dev/null || true
	@sudo cp -r $(LOG_DIR)/*.json ./backups/data-$(shell date +%Y%m%d_%H%M%S) 2>/dev/null || true
	@echo "✅ Backup created in ./backups/"

# Show version
version: ## Show version information
	@echo "🌊 OceanProxy Version Information"
	@echo "================================"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@if [ -f $(BINARY_PATH) ]; then \
		echo "Binary: $(BINARY_PATH)"; \
		echo "Binary Size: $(du -h $(BINARY_PATH) | cut -f1)"; \
	fi

# Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) -f build/Dockerfile .
	@docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	@docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-p 1337:1337 \
		-p 1338:1338 \
		-p 9876:9876 \
		--env-file .env \
		$(APP_NAME):latest

docker-stop: ## Stop Docker container
	@docker stop $(APP_NAME) 2>/dev/null || true
	@docker rm $(APP_NAME) 2>/dev/null || true

# Quick fixes
fix-config: ## Fix common configuration issues
	@echo "🔧 Fixing common configuration issues..."
	@if [ ! -f $(CONFIG_DIR)/oceanproxy.env ]; then \
		echo "Creating environment file..."; \
		sudo cp .env.example $(CONFIG_DIR)/oceanproxy.env 2>/dev/null || cp .env.example ./oceanproxy.env; \
	fi
	@echo "✅ Configuration check complete"
	@echo "📝 Remember to edit $(CONFIG_DIR)/oceanproxy.env with your API keys"

fix-auth: ## Fix authentication issues
	@echo "🔐 Diagnosing authentication issues..."
	@echo "Current BEARER_TOKEN in environment file:"
	@grep "BEARER_TOKEN=" $(CONFIG_DIR)/oceanproxy.env 2>/dev/null || grep "BEARER_TOKEN=" .env 2>/dev/null || echo "Not found"
	@echo ""
	@echo "Testing authentication..."
	@make test-auth

# Complete reset
reset: ## Complete reset (stops service, cleans, rebuilds)
	@echo "🔄 Performing complete reset..."
	@make stop 2>/dev/null || true
	@make clean
	@make build
	@make fix-permissions
	@make fix-config
	@echo "✅ Reset complete. Use 'make start' to start service"

# Help with common issues
troubleshoot: ## Troubleshoot common issues
	@echo "🔍 OceanProxy Troubleshooting Guide"
	@echo "=================================="
	@echo ""
	@echo "1. 🔐 Authentication Issues:"
	@echo "   Problem: 'Invalid bearer token' errors"
	@echo "   Solution: make fix-auth"
	@echo ""
	@echo "2. 📦 Build Issues:"
	@echo "   Problem: Build failures"
	@echo "   Solution: make clean && make deps && make build"
	@echo ""
	@echo "3. 🚫 Permission Issues:"
	@echo "   Problem: Cannot write to logs/data directories"
	@echo "   Solution: make fix-permissions"
	@echo ""
	@echo "4. ⚙️  Service Issues:"
	@echo "   Problem: Service won't start"
	@echo "   Solution: make status && make logs-recent"
	@echo ""
	@echo "5. 🌐 Network Issues:"
	@echo "   Problem: Ports not listening"
	@echo "   Solution: make status && sudo netstat -tlnp | grep oceanproxy"
	@echo ""
	@echo "Quick fixes:"
	@echo "  make reset      # Complete reset"
	@echo "  make fix-auth   # Fix authentication"
	@echo "  make fix-config # Fix configuration"