#!/bin/bash

# Quick 3proxy Fix Script for OceanProxy
# Run this to fix the missing 3proxy binary issue

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "🔧 3proxy Quick Fix for OceanProxy"
echo "=================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Method 1: Try package manager (fastest)
log_info "Method 1: Trying package manager installation..."
apt-get update
if apt-get install -y 3proxy; then
    if command -v 3proxy &> /dev/null; then
        log_success "3proxy installed via package manager!"
        echo "Version: $(3proxy 2>&1 | head -1)"
        echo "Location: $(which 3proxy)"
        exit 0
    fi
fi

log_warning "Package manager failed, trying compilation from source..."

# Method 2: Compile from source with multiple URLs
log_info "Method 2: Compiling from source..."

# Install dependencies
apt-get install -y build-essential wget curl

# Create temp directory
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Try multiple download sources
DOWNLOAD_SUCCESS=false
SOURCES=(
    "https://github.com/3proxy/3proxy/archive/refs/tags/0.9.4.tar.gz"
    "https://github.com/z3APA3A/3proxy/archive/0.9.4.tar.gz"
    "https://sourceforge.net/projects/3proxy/files/3proxy/0.9.4/3proxy-0.9.4.tar.gz/download"
)

for i in "${!SOURCES[@]}"; do
    url="${SOURCES[$i]}"
    filename="3proxy-$i.tar.gz"
    
    log_info "Trying source $((i+1)): ${url:0:50}..."
    
    if timeout 30 wget -q "$url" -O "$filename" 2>/dev/null && [[ -s "$filename" ]]; then
        log_success "Downloaded successfully!"
        ARCHIVE_FILE="$filename"
        DOWNLOAD_SUCCESS=true
        break
    elif timeout 30 curl -L -s "$url" -o "$filename" 2>/dev/null && [[ -s "$filename" ]]; then
        log_success "Downloaded with curl!"
        ARCHIVE_FILE="$filename"  
        DOWNLOAD_SUCCESS=true
        break
    fi
    
    rm -f "$filename"
done

if [[ "$DOWNLOAD_SUCCESS" == false ]]; then
    log_error "All download attempts failed!"
    
    # Method 3: Manual wget with specific version
    log_info "Method 3: Trying direct wget..."
    if wget https://3proxy.org/0.9.4/3proxy-0.9.4.tar.gz -O manual.tar.gz 2>/dev/null && [[ -s "manual.tar.gz" ]]; then
        ARCHIVE_FILE="manual.tar.gz"
        DOWNLOAD_SUCCESS=true
    fi
fi

if [[ "$DOWNLOAD_SUCCESS" == false ]]; then
    log_error "Could not download 3proxy source code"
    log_info "Please check your internet connection and try again"
    exit 1
fi

# Extract
log_info "Extracting archive..."
if ! tar -xzf "$ARCHIVE_FILE" 2>/dev/null; then
    log_error "Failed to extract archive"
    exit 1
fi

# Find source directory
SOURCE_DIR=""
for dir in 3proxy-* */; do
    if [[ -d "$dir" ]] && ([[ -f "$dir/Makefile.Linux" ]] || [[ -f "$dir/Makefile" ]]); then
        SOURCE_DIR="$dir"
        break
    fi
done

if [[ -z "$SOURCE_DIR" ]]; then
    log_error "Could not find 3proxy source directory"
    exit 1
fi

cd "$SOURCE_DIR"
log_info "Building 3proxy in: $SOURCE_DIR"

# Try different build methods
BUILD_SUCCESS=false

if [[ -f "Makefile.Linux" ]]; then
    log_info "Building with Makefile.Linux..."
    if make -f Makefile.Linux 2>/dev/null; then
        BUILD_SUCCESS=true
    fi
fi

if [[ "$BUILD_SUCCESS" == false ]] && [[ -f "Makefile" ]]; then
    log_info "Building with standard Makefile..."
    if make 2>/dev/null; then
        BUILD_SUCCESS=true
    fi
fi

if [[ "$BUILD_SUCCESS" == false ]]; then
    log_error "Build failed with all methods"
    exit 1
fi

# Find binary
BINARY_PATH=""
for path in bin/3proxy src/3proxy 3proxy; do
    if [[ -x "$path" ]]; then
        BINARY_PATH="$path"
        break
    fi
done

if [[ -z "$BINARY_PATH" ]]; then
    log_error "Could not find compiled 3proxy binary"
    exit 1
fi

# Install binary
log_info "Installing 3proxy binary..."
cp "$BINARY_PATH" /usr/local/bin/3proxy
chmod +x /usr/local/bin/3proxy
ln -sf /usr/local/bin/3proxy /usr/bin/3proxy
ln -sf /usr/local/bin/3proxy /bin/3proxy

# Clean up
cd /
rm -rf "$TEMP_DIR"

# Verify installation
if command -v 3proxy &> /dev/null; then
    log_success "3proxy installed successfully!"
    echo "Version: $(3proxy 2>&1 | head -1)"
    echo "Location: $(which 3proxy)"
    
    # Test binary
    if /usr/local/bin/3proxy >/dev/null 2>&1 || /usr/local/bin/3proxy --help >/dev/null 2>&1; then
        log_success "3proxy binary is working correctly"
    else
        log_warning "3proxy binary installed but may have issues"
    fi
    
    # Now restart OceanProxy to detect the new binary
    log_info "Restarting OceanProxy service to detect 3proxy..."
    systemctl restart oceanproxy
    sleep 3
    
    if systemctl is-active --quiet oceanproxy; then
        log_success "OceanProxy restarted successfully!"
        log_info "Now test creating a proxy plan to verify full functionality"
    else
        log_warning "OceanProxy restart had issues - check logs with: journalctl -u oceanproxy"
    fi
    
else
    log_error "3proxy installation verification failed"
    exit 1
fi

echo
log_success "🎉 3proxy fix completed!"
echo
echo "Next steps:"
echo "1. Test OceanProxy again using the installer menu"
echo "2. Create a test proxy plan to verify everything works"
echo "3. Check that proxy ports are now listening"