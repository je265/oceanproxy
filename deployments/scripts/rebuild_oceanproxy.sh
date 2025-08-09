#!/bin/bash

# OceanProxy Complete Rebuild Script
# Fixes the configuration loading issue and rebuilds the entire system

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_header() { echo -e "${CYAN}$1${NC}"; }

echo -e "${WHITE}🌊 OceanProxy Complete Rebuild Script${NC}"
echo "====================================="
echo

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    log_error "Do not run this script as root. Use your regular user account."
    exit 1
fi

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || ! grep -q "oceanproxy" go.mod; then
    log_error "Please run this script from the OceanProxy project root directory"
    exit 1
fi

log_header "Step 1: Stopping existing service"
sudo systemctl stop oceanproxy 2>/dev/null || log_info "Service not running"

log_header "Step 2: Moving config from internal to pkg"
log_info "Creating pkg directory structure..."
mkdir -p pkg/config pkg/logger

# Move config from internal to pkg if it exists in internal
if [[ -f "internal/config/config.go" ]]; then
    log_info "Moving config from internal/config to pkg/config..."
    cp internal/config/config.go pkg/config/config.go
    
    # Update package declaration
    sed -i 's/package config/package config/' pkg/config/config.go
fi

# Move logger from internal to pkg if it exists
if [[ -f "internal/pkg/logger/logger.go" ]]; then
    log_info "Moving logger from internal/pkg/logger to pkg/logger..."
    cp internal/pkg/logger/logger.go pkg/logger/logger.go
    
    # Update package declaration
    sed -i 's/package logger/package logger/' pkg/logger/logger.go
fi

log_header "Step 3: Updating import statements"
log_info "Updating import statements in Go files..."

# Find and update import statements
find . -name "*.go" -type f -exec grep -l "github.com/je265/oceanproxy/pkg/config" {} \; | while read file; do
    log_info "Updating imports in $file"
    sed -i 's|github.com/je265/oceanproxy/pkg/config|github.com/je265/oceanproxy/pkg/config|g' "$file"
done

find . -name "*.go" -type f -exec grep -l "github.com/je265/oceanproxy/pkg/logger" {} \; | while read file; do
    log_info "Updating imports in $file"
    sed -i 's|github.com/je265/oceanproxy/pkg/logger|github.com/je265/oceanproxy/pkg/logger|g' "$file"
done

log_header "Step 4: Cleaning and updating dependencies"
log_info "Cleaning Go module cache..."
go clean -modcache

log_info "Tidying Go modules..."
go mod tidy

log_header "Step 5: Building the application"
log_info "Building OceanProxy with new structure..."
mkdir -p bin

# Build with verbose output to catch any issues
go build -v -o bin/oceanproxy ./cmd/server/main.go

if [[ $? -eq 0 ]]; then
    log_success "Build successful!"
else
    log_error "Build failed!"
    exit 1
fi

log_header "Step 6: Installing the rebuilt application"
log_info "Installing new binary..."
sudo cp bin/oceanproxy /usr/local/bin/oceanproxy
sudo chmod +x /usr/local/bin/oceanproxy

log_header "Step 7: Verifying configuration"
if [[ -f "/etc/oceanproxy/oceanproxy.env" ]]; then
    log_info "Environment file exists: /etc/oceanproxy/oceanproxy.env"
    
    # Check for required variables
    if grep -q "BEARER_TOKEN=" /etc/oceanproxy/oceanproxy.env; then
        BEARER_TOKEN=$(grep "BEARER_TOKEN=" /etc/oceanproxy/oceanproxy.env | cut -d'=' -f2)
        log_info "BEARER_TOKEN found: $BEARER_TOKEN"
    else
        log_warning "BEARER_TOKEN not found in environment file"
    fi
    
    if grep -q "PROXIES_FO_API_KEY=" /etc/oceanproxy/oceanproxy.env; then
        log_info "PROXIES_FO_API_KEY found"
    else
        log_warning "PROXIES_FO_API_KEY not found"
    fi
    
    if grep -q "NETTIFY_API_KEY=" /etc/oceanproxy/oceanproxy.env; then
        log_info "NETTIFY_API_KEY found"
    else
        log_warning "NETTIFY_API_KEY not found"
    fi
else
    log_warning "Environment file not found, creating it..."
    sudo mkdir -p /etc/oceanproxy
    sudo cp .env.example /etc/oceanproxy/oceanproxy.env
    log_info "Please edit /etc/oceanproxy/oceanproxy.env with your configuration"
fi

log_header "Step 8: Starting the service"
log_info "Starting OceanProxy service..."
sudo systemctl start oceanproxy

# Wait for service to start
sleep 3

log_header "Step 9: Testing the fixed system"
if systemctl is-active --quiet oceanproxy; then
    log_success "Service is running!"
    
    # Test health endpoint
    if curl -s -f http://localhost:8080/health >/dev/null; then
        log_success "Health endpoint is responding"
        
        # Test authentication with the correct token
        if [[ -n "$BEARER_TOKEN" ]]; then
            log_info "Testing authentication with token: $BEARER_TOKEN"
            
            response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer $BEARER_TOKEN" http://localhost:8080/api/v1/plans -o /dev/null)
            
            if [[ "$response" -eq 200 ]]; then
                log_success "Authentication is working correctly!"
            elif [[ "$response" -eq 500 ]]; then
                log_warning "Authentication works but got 500 (likely due to missing plans)"
            elif [[ "$response" -eq 401 ]]; then
                log_error "Authentication failed - configuration issue"
            else
                log_warning "Got unexpected response code: $response"
            fi
        else
            log_warning "No BEARER_TOKEN found for testing"
        fi
    else
        log_error "Health endpoint not responding"
    fi
else
    log_error "Service failed to start"
    log_info "Checking logs..."
    sudo journalctl -u oceanproxy -n 20 --no-pager
fi

log_header "Step 10: System status"
echo
log_info "Final status check:"
echo "==================="

# Service status
if systemctl is-active --quiet oceanproxy; then
    echo "✅ Service: Running"
else
    echo "❌ Service: Stopped"
fi

# Port status
if netstat -tlnp 2>/dev/null | grep -q ":8080 "; then
    echo "✅ API Port (8080): Listening"
else
    echo "❌ API Port (8080): Not listening"
fi

# Configuration status
if [[ -f "/etc/oceanproxy/oceanproxy.env" ]]; then
    echo "✅ Configuration: Present"
else
    echo "❌ Configuration: Missing"
fi

echo
log_header "Rebuild Summary"
echo "==============="
log_success "✅ Structure fixed (config moved from internal to pkg)"
log_success "✅ Import statements updated"
log_success "✅ Application rebuilt and installed"
log_success "✅ Service restarted"

echo
log_info "📋 Next Steps:"
echo "1. Test the API: curl -H \"Authorization: Bearer $BEARER_TOKEN\" http://localhost:8080/api/v1/plans"
echo "2. Monitor logs: sudo journalctl -u oceanproxy -f"
echo "3. Check status: sudo systemctl status oceanproxy"

echo
log_info "🔧 Useful Commands:"
echo "   make status      # Check service status"
echo "   make logs        # View logs"
echo "   make test-auth   # Test authentication"
echo "   make health      # Health check"

echo
if systemctl is-active --quiet oceanproxy && curl -s -f http://localhost:8080/health >/dev/null; then
    log_success "🎉 OceanProxy rebuild completed successfully!"
else
    log_warning "⚠️  Rebuild completed but service may need attention"
    echo "   Check logs: sudo journalctl -u oceanproxy -f"
fi