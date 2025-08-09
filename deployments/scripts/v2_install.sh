#!/bin/bash

# OceanProxy Menu-Driven Installation Script - COMPLETE STRUCTURAL IMPLEMENTATION
# This script provides a menu system to install components individually or all at once

set -euo pipefail

# Configuration
APP_NAME="oceanproxy"
APP_USER="oceanproxy"
APP_GROUP="oceanproxy"
APP_DIR="/opt/oceanproxy"
CONFIG_DIR="/etc/oceanproxy"
LOG_DIR="/var/log/oceanproxy"
DATA_DIR="/var/lib/oceanproxy"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Installation status tracking
declare -A COMPONENT_STATUS=(
    ["system_deps"]="not_installed"
    ["nginx"]="not_installed"
    ["3proxy"]="not_installed"
    ["go"]="not_installed"
    ["user_dirs"]="not_installed"
    ["oceanproxy"]="not_installed"
    ["config"]="not_installed"
    ["systemd"]="not_installed"
    ["firewall"]="not_installed"
    ["optimization"]="not_installed"
)

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_menu() {
    echo -e "${CYAN}$1${NC}"
}

log_highlight() {
    echo -e "${WHITE}$1${NC}"
}

# Check component status
check_component_status() {
    # Check system dependencies
    if command -v curl &> /dev/null && command -v wget &> /dev/null && command -v git &> /dev/null; then
        COMPONENT_STATUS["system_deps"]="installed"
    else
        COMPONENT_STATUS["system_deps"]="not_installed"
    fi
    
    # Check nginx
    if command -v nginx &> /dev/null && nginx -V 2>&1 | grep -q "with-stream"; then
        COMPONENT_STATUS["nginx"]="installed"
    else
        COMPONENT_STATUS["nginx"]="not_installed"
    fi
    
    # Check 3proxy
    if command -v 3proxy &> /dev/null || [[ -f "/usr/local/bin/3proxy" ]]; then
        COMPONENT_STATUS["3proxy"]="installed"
    else
        COMPONENT_STATUS["3proxy"]="not_installed"
    fi
    
    # Check Go
    if command -v go &> /dev/null && go version &> /dev/null; then
        COMPONENT_STATUS["go"]="installed"
    elif [[ -f "/usr/local/go/bin/go" ]] && /usr/local/go/bin/go version &> /dev/null; then
        COMPONENT_STATUS["go"]="installed"
    else
        COMPONENT_STATUS["go"]="not_installed"
    fi
    
    # Check user and directories
    if id "$APP_USER" &>/dev/null && [[ -d "$APP_DIR" ]] && [[ -d "$CONFIG_DIR" ]]; then
        COMPONENT_STATUS["user_dirs"]="installed"
    else
        COMPONENT_STATUS["user_dirs"]="not_installed"
    fi
    
    # Check OceanProxy binary
    if [[ -f "/usr/local/bin/oceanproxy" ]] && [[ -x "/usr/local/bin/oceanproxy" ]]; then
        COMPONENT_STATUS["oceanproxy"]="installed"
    else
        COMPONENT_STATUS["oceanproxy"]="not_installed"
    fi
    
    # Check configuration
    if [[ -f "$CONFIG_DIR/oceanproxy.env" ]] && [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        COMPONENT_STATUS["config"]="installed"
    else
        COMPONENT_STATUS["config"]="not_installed"
    fi
    
    # Check systemd service
    if [[ -f "/etc/systemd/system/oceanproxy.service" ]]; then
        COMPONENT_STATUS["systemd"]="installed"
    else
        COMPONENT_STATUS["systemd"]="not_installed"
    fi
    
    # Check firewall configuration
    if command -v ufw &> /dev/null; then
        if ufw status 2>/dev/null | grep -q "Status: active" && \
           ufw status 2>/dev/null | grep -q "8080\|1337\|1338"; then
            COMPONENT_STATUS["firewall"]="installed"
        else
            COMPONENT_STATUS["firewall"]="partial"
        fi
    elif command -v iptables &> /dev/null; then
        # Basic iptables check - if it exists, consider firewall as available
        COMPONENT_STATUS["firewall"]="installed"
    else
        COMPONENT_STATUS["firewall"]="not_installed"
    fi
    
    # Check system optimization
    if [[ -f "/etc/sysctl.d/99-oceanproxy.conf" ]] || \
       sysctl net.core.somaxconn 2>/dev/null | grep -q -E "(4096|65535)"; then
        COMPONENT_STATUS["optimization"]="installed"
    else
        COMPONENT_STATUS["optimization"]="not_installed"
    fi
}

# Get status icon
get_status_icon() {
    local status=$1
    case $status in
        "installed") echo -e "${GREEN}✅${NC}" ;;
        "partial") echo -e "${YELLOW}⚠️${NC}" ;;
        "not_installed") echo -e "${RED}❌${NC}" ;;
        *) echo -e "${YELLOW}⚠️${NC}" ;;
    esac
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect OS and version
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        OS=$(lsb_release -si)
        VER=$(lsb_release -sr)
    else
        log_error "Cannot detect operating system"
        exit 1
    fi
}

# Show system information
show_system_info() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}              SYSTEM INFORMATION                ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    echo -e "  ${BLUE}OS:${NC} $OS $VER"
    echo -e "  ${BLUE}Kernel:${NC} $(uname -r)"
    echo -e "  ${BLUE}Architecture:${NC} $(uname -m)"
    echo -e "  ${BLUE}Memory:${NC} $(free -h | grep Mem | awk '{print $2}')"
    echo -e "  ${BLUE}Disk Space:${NC} $(df -h / | tail -1 | awk '{print $4}') available"
    echo -e "  ${BLUE}Network:${NC} $(if ping -c1 8.8.8.8 &>/dev/null; then echo 'Connected'; else echo 'Disconnected'; fi)"
    echo
    read -p "Press Enter to continue..."
}

# Display main menu
show_main_menu() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}            🌊 OCEANPROXY INSTALLER             ${CYAN}║${NC}"
    echo -e "${CYAN}║${WHITE}         Complete Structural Implementation     ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    # Update component status
    check_component_status
    
    echo -e "${WHITE}Component Status:${NC}"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["system_deps"]}") System Dependencies"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["nginx"]}") Nginx (with stream module)"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["3proxy"]}") 3proxy"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["go"]}") Go Language"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["user_dirs"]}") User & Directories"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["oceanproxy"]}") OceanProxy Binary"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["config"]}") Configuration"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["systemd"]}") Systemd Service"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["firewall"]}") Firewall Configuration"
    echo -e "  $(get_status_icon "${COMPONENT_STATUS["optimization"]}") System Optimization"
    echo
    
    echo -e "${WHITE}Installation Options:${NC}"
    echo -e "  ${CYAN}1)${NC} 🚀 Full Installation (Recommended)"
    echo -e "  ${CYAN}2)${NC} 🔧 Component Selection Menu"
    echo -e "  ${CYAN}3)${NC} 📋 System Information"
    echo -e "  ${CYAN}4)${NC} 🔍 Run Diagnostics"
    echo -e "  ${CYAN}5)${NC} 📖 View Installation Logs"
    echo -e "  ${CYAN}6)${NC} ⚙️  Service Management"
    echo -e "  ${CYAN}7)${NC} 🗑️  Uninstall OceanProxy"
    echo -e "  ${CYAN}8)${NC} 🔑 Configure Secrets & Env (QoL)"
    echo -e "  ${CYAN}0)${NC} 🚪 Exit"
    echo
}

# Component selection menu
show_component_menu() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}           COMPONENT SELECTION MENU             ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    check_component_status
    
    echo -e "${WHITE}Select components to install:${NC}"
    echo
    echo -e "  ${CYAN}1)${NC}  $(get_status_icon "${COMPONENT_STATUS["system_deps"]}") System Dependencies (curl, wget, git, build tools)"
    echo -e "  ${CYAN}2)${NC}  $(get_status_icon "${COMPONENT_STATUS["nginx"]}") Nginx with Stream Module"
    echo -e "  ${CYAN}3)${NC}  $(get_status_icon "${COMPONENT_STATUS["3proxy"]}") 3proxy (Multiple installation methods)"
    echo -e "  ${CYAN}4)${NC}  $(get_status_icon "${COMPONENT_STATUS["go"]}") Go Programming Language"
    echo -e "  ${CYAN}5)${NC}  $(get_status_icon "${COMPONENT_STATUS["user_dirs"]}") Create User & Directories"
    echo -e "  ${CYAN}6)${NC}  $(get_status_icon "${COMPONENT_STATUS["oceanproxy"]}") Build & Install OceanProxy"
    echo -e "  ${CYAN}7)${NC}  $(get_status_icon "${COMPONENT_STATUS["config"]}") Install Configuration Files"
    echo -e "  ${CYAN}8)${NC}  $(get_status_icon "${COMPONENT_STATUS["systemd"]}") Setup Systemd Service"
    echo -e "  ${CYAN}9)${NC}  $(get_status_icon "${COMPONENT_STATUS["firewall"]}") Configure Firewall"
    echo -e "  ${CYAN}10)${NC} $(get_status_icon "${COMPONENT_STATUS["optimization"]}") System Optimization"
    echo
    echo -e "  ${CYAN}11)${NC} 🔄 Refresh Status"
    echo -e "  ${CYAN}12)${NC} 🔧 Fix/Repair Component"
    echo -e "  ${CYAN}0)${NC}  ← Back to Main Menu"
    echo
}

# Service management menu
show_service_menu() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}           SERVICE MANAGEMENT MENU              ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    # Check service status
    local oceanproxy_status=$(systemctl is-active oceanproxy 2>/dev/null || echo "inactive")
    local nginx_status=$(systemctl is-active nginx 2>/dev/null || echo "inactive")
    local redis_status=$(systemctl is-active redis-server 2>/dev/null || systemctl is-active redis 2>/dev/null || echo "inactive")
    
    echo -e "${WHITE}Current Service Status:${NC}"
    echo -e "  OceanProxy: $(if [[ $oceanproxy_status == "active" ]]; then echo -e "${GREEN}Running${NC}"; else echo -e "${RED}Stopped${NC}"; fi)"
    echo -e "  Nginx:      $(if [[ $nginx_status == "active" ]]; then echo -e "${GREEN}Running${NC}"; else echo -e "${RED}Stopped${NC}"; fi)"
    echo -e "  Redis:      $(if [[ $redis_status == "active" ]]; then echo -e "${GREEN}Running${NC}"; else echo -e "${RED}Stopped${NC}"; fi)"
    echo
    
    echo -e "${WHITE}Service Management:${NC}"
    echo -e "  ${CYAN}1)${NC} 🟢 Start All Services"
    echo -e "  ${CYAN}2)${NC} 🔴 Stop All Services"
    echo -e "  ${CYAN}3)${NC} 🔄 Restart All Services"
    echo -e "  ${CYAN}4)${NC} 📊 View Service Status"
    echo -e "  ${CYAN}5)${NC} 📋 View OceanProxy Logs"
    echo -e "  ${CYAN}6)${NC} 📋 View Nginx Logs"
    echo -e "  ${CYAN}7)${NC} 🧪 Test Installation"
    echo -e "  ${CYAN}8)${NC} ⚙️  Edit Configuration"
    echo -e "  ${CYAN}0)${NC} ← Back to Main Menu"
    echo
}

# Installation functions (keeping the original implementations)

# Install basic system dependencies
install_basic_dependencies() {
    log_info "Installing basic system dependencies..."
    
    apt-get update
    apt-get install -y \
        curl \
        wget \
        git \
        jq \
        lsof \
        net-tools \
        htop \
        vim \
        unzip \
        ca-certificates \
        gnupg \
        lsb-release \
        build-essential \
        make \
        gcc \
        redis-server \
        postgresql-client \
        logrotate \
        fail2ban \
        software-properties-common \
        apt-transport-https
    
    COMPONENT_STATUS["system_deps"]="installed"
    log_success "Basic dependencies installed"
}

# Install nginx with stream module
install_nginx() {
    log_info "Installing nginx with stream module..."
    
    # Remove any existing nginx installation
    systemctl stop nginx 2>/dev/null || true
    apt-get remove -y nginx nginx-common nginx-core nginx-full 2>/dev/null || true
    
    # Add official nginx repository
    curl -fsSL https://nginx.org/keys/nginx_signing.key | gpg --dearmor -o /usr/share/keyrings/nginx-keyring.gpg
    echo "deb [signed-by=/usr/share/keyrings/nginx-keyring.gpg] https://nginx.org/packages/ubuntu $(lsb_release -cs) nginx" > /etc/apt/sources.list.d/nginx.list
    
    # Set repository preferences
    cat > /etc/apt/preferences.d/99nginx << 'EOF'
Package: *
Pin: origin nginx.org
Pin: release o=nginx
Pin-Priority: 900
EOF
    
    apt-get update
    apt-get install -y nginx
    
    # Verify stream module is available
    if nginx -V 2>&1 | grep -q "with-stream"; then
        log_success "Nginx installed with stream module"
    else
        log_error "Nginx does not have stream module - this is required for OceanProxy"
        return 1
    fi
    
    # Enable and start nginx
    systemctl enable nginx
    systemctl start nginx
    
    COMPONENT_STATUS["nginx"]="installed"
    log_success "Nginx with stream module installed"
}

# Install 3proxy with multiple fallback methods
install_3proxy() {
    log_info "Installing 3proxy from source..."
    
    # Check if 3proxy is already installed
    if command -v 3proxy &> /dev/null; then
        local version_info=$(3proxy 2>&1 | head -1 || echo 'version unknown')
        log_info "3proxy already installed: $version_info"
        COMPONENT_STATUS["3proxy"]="installed"
        return
    fi
    
    # Install build dependencies
    log_info "Installing build dependencies..."
    apt-get install -y build-essential wget curl
    
    # Create temporary directory with better error handling
    local TEMP_DIR
    TEMP_DIR=$(mktemp -d) || {
        log_error "Failed to create temporary directory"
        return 1
    }
    
    local original_dir=$(pwd)
    cd "$TEMP_DIR" || {
        log_error "Failed to change to temporary directory"
        return 1
    }
    
    log_info "Working in temporary directory: $TEMP_DIR"
    
    # Try multiple download methods and sources
    local downloaded=false
    local archive_file=""
    
    # Method 1: Try GitHub releases (more reliable)
    log_info "Attempting to download 3proxy from GitHub releases..."
    if wget --timeout=30 --tries=3 -q "https://github.com/3proxy/3proxy/archive/refs/tags/0.9.4.tar.gz" -O "3proxy-0.9.4.tar.gz"; then
        archive_file="3proxy-0.9.4.tar.gz"
        downloaded=true
        log_info "Downloaded from GitHub releases"
    fi
    
    # Method 2: Try alternative GitHub URL if first failed
    if [[ "$downloaded" == false ]]; then
        log_info "Trying alternative GitHub URL..."
        if wget --timeout=30 --tries=3 -q "https://github.com/z3APA3A/3proxy/archive/0.9.4.tar.gz" -O "3proxy-alt.tar.gz"; then
            archive_file="3proxy-alt.tar.gz"
            downloaded=true
            log_info "Downloaded from alternative GitHub URL"
        fi
    fi
    
    # Method 3: Try using curl if wget failed
    if [[ "$downloaded" == false ]]; then
        log_info "Trying with curl..."
        if curl -L --connect-timeout 30 --max-time 120 -s "https://github.com/3proxy/3proxy/archive/refs/tags/0.9.4.tar.gz" -o "3proxy-curl.tar.gz" && [[ -s "3proxy-curl.tar.gz" ]]; then
            archive_file="3proxy-curl.tar.gz"
            downloaded=true
            log_info "Downloaded using curl"
        fi
    fi
    
    # Method 4: Try package manager as fallback
    if [[ "$downloaded" == false ]]; then
        log_warning "Source download failed, trying package manager..."
        cd "$original_dir"
        rm -rf "$TEMP_DIR"
        
        if apt-get update && apt-get install -y 3proxy; then
            COMPONENT_STATUS["3proxy"]="installed"
            log_success "3proxy installed via package manager"
            return
        else
            log_error "Failed to install 3proxy via all methods"
            return 1
        fi
    fi
    
    # Extract the archive
    log_info "Extracting 3proxy source from: $archive_file"
    if ! tar -xzf "$archive_file" 2>/dev/null; then
        log_error "Failed to extract 3proxy archive"
        cd "$original_dir"
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    # Find the extracted directory
    local source_dir=""
    for dir in 3proxy-* */; do
        if [[ -d "$dir" && -f "$dir/Makefile.Linux" ]]; then
            source_dir="$dir"
            break
        fi
    done
    
    if [[ -z "$source_dir" ]]; then
        if [[ -f "Makefile.Linux" ]]; then
            source_dir="."
        else
            log_error "Could not find 3proxy source directory with Makefile.Linux"
            cd "$original_dir"
            rm -rf "$TEMP_DIR"
            return 1
        fi
    fi
    
    cd "$source_dir" || {
        log_error "Failed to enter source directory: $source_dir"
        cd "$original_dir"
        rm -rf "$TEMP_DIR"
        return 1
    }
    
    log_info "Building 3proxy from source directory: $source_dir"
    
    # Build 3proxy with error handling
    if ! make -f Makefile.Linux; then
        log_error "Failed to build 3proxy"
        log_info "Trying alternative build method..."
        
        if ! make; then
            log_error "All build methods failed"
            cd "$original_dir"
            rm -rf "$TEMP_DIR"
            return 1
        fi
    fi
    
    # Find the compiled binary
    local binary_path=""
    for path in bin/3proxy src/3proxy 3proxy; do
        if [[ -x "$path" ]]; then
            binary_path="$path"
            break
        fi
    done
    
    if [[ -z "$binary_path" ]]; then
        log_error "Could not find compiled 3proxy binary"
        cd "$original_dir"
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    log_info "Found 3proxy binary at: $binary_path"
    
    # Install binary
    cp "$binary_path" /usr/local/bin/3proxy || {
        log_error "Failed to copy 3proxy binary"
        cd "$original_dir"
        rm -rf "$TEMP_DIR"
        return 1
    }
    
    chmod +x /usr/local/bin/3proxy
    ln -sf /usr/local/bin/3proxy /usr/bin/3proxy
    
    # Clean up
    cd "$original_dir"
    rm -rf "$TEMP_DIR"
    
    # Verify installation
    if command -v 3proxy &> /dev/null; then
        local version_check=$(/usr/local/bin/3proxy 2>&1 | head -1 || echo 'installed successfully')
        COMPONENT_STATUS["3proxy"]="installed"
        log_success "3proxy installed successfully: $version_check"
    else
        log_error "3proxy installation verification failed"
        return 1
    fi
}

# Install Go
install_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go is already installed: $GO_VERSION"
        
        if [[ $(echo "$GO_VERSION 1.19" | tr " " "\n" | sort -V | head -n1) == "1.19" ]]; then
            COMPONENT_STATUS["go"]="installed"
            return
        else
            log_info "Go version is too old, installing newer version..."
        fi
    fi
    
    log_info "Installing Go..."
    
    GO_VERSION="1.21.5"
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) GO_ARCH="amd64" ;;
        aarch64|arm64) GO_ARCH="arm64" ;;
        armv7l) GO_ARCH="armv6l" ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            return 1
            ;;
    esac
    
    cd /tmp
    
    if ! wget "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"; then
        log_error "Failed to download Go"
        return 1
    fi
    
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    
    COMPONENT_STATUS["go"]="installed"
    log_success "Go installed: $(/usr/local/go/bin/go version)"
}

# Create user and directories
setup_user_and_directories() {
    log_info "Setting up user and directories..."
    
    if ! id "$APP_USER" &>/dev/null; then
        useradd --system --shell /bin/false --home "$APP_DIR" --create-home "$APP_USER"
        log_success "Created user: $APP_USER"
    else
        log_info "User $APP_USER already exists"
    fi
    
    mkdir -p "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    mkdir -p /etc/3proxy
    mkdir -p /etc/nginx/conf.d
    
    chown -R "$APP_USER:$APP_USER" "$APP_DIR" "$LOG_DIR" "$DATA_DIR"
    chown -R "$APP_USER:$APP_USER" /etc/3proxy
    chmod 755 "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    chmod 755 /etc/3proxy
    
    COMPONENT_STATUS["user_dirs"]="installed"
    log_success "User and directories created"
}

# Build and install OceanProxy binary
build_and_install_oceanproxy() {
    log_info "Building and installing OceanProxy..."
    
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    
    if [[ -f "$(pwd)/go.mod" ]]; then
        SOURCE_DIR="$(pwd)"
        log_info "Using current directory: $SOURCE_DIR"
    else
        CURRENT_DIR="$(pwd)"
        SOURCE_DIR=""
        while [[ "$CURRENT_DIR" != "/" ]]; do
            if [[ -f "$CURRENT_DIR/go.mod" ]]; then
                SOURCE_DIR="$CURRENT_DIR"
                break
            fi
            CURRENT_DIR="$(dirname "$CURRENT_DIR")"
        done
        
        if [[ -z "$SOURCE_DIR" ]]; then
            log_error "Could not find go.mod file. Make sure you're in the OceanProxy directory"
            return 1
        fi
        
        log_info "Found source directory: $SOURCE_DIR"
    fi
    
    cd "$SOURCE_DIR"
    
    log_info "Building OceanProxy from source..."
    
    if [[ ! -f go.mod ]]; then
        log_error "go.mod not found in $SOURCE_DIR"
        return 1
    fi
    
    /usr/local/go/bin/go mod download
    /usr/local/go/bin/go mod tidy
    
    mkdir -p bin
    /usr/local/go/bin/go build -o bin/oceanproxy cmd/server/main.go
    /usr/local/go/bin/go build -o bin/oceanproxy-cli cmd/cli/main.go
    
    cp bin/oceanproxy /usr/local/bin/
    cp bin/oceanproxy-cli /usr/local/bin/
    chmod +x /usr/local/bin/oceanproxy
    chmod +x /usr/local/bin/oceanproxy-cli
    
    COMPONENT_STATUS["oceanproxy"]="installed"
    log_success "OceanProxy binaries installed"
}

# Install configuration files
install_config() {
    log_info "Installing configuration files..."
    
    if [[ -f "$(pwd)/go.mod" ]]; then
        SOURCE_DIR="$(pwd)"
    else
        CURRENT_DIR="$(pwd)"
        SOURCE_DIR=""
        while [[ "$CURRENT_DIR" != "/" ]]; do
            if [[ -f "$CURRENT_DIR/go.mod" ]]; then
                SOURCE_DIR="$CURRENT_DIR"
                break
            fi
            CURRENT_DIR="$(dirname "$CURRENT_DIR")"
        done
    fi
    
    log_info "Using source directory: $SOURCE_DIR"
    
    # Copy configuration files
    if [[ -f "$SOURCE_DIR/configs/config.yaml" ]]; then
        cp "$SOURCE_DIR/configs/config.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_USER" "$CONFIG_DIR/config.yaml"
    fi
    
    if [[ -f "$SOURCE_DIR/configs/regions.yaml" ]]; then
        cp "$SOURCE_DIR/configs/regions.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_USER" "$CONFIG_DIR/regions.yaml"
    fi
    
    if [[ -f "$SOURCE_DIR/configs/proxy-plans.yaml" ]]; then
        cp "$SOURCE_DIR/configs/proxy-plans.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_USER" "$CONFIG_DIR/proxy-plans.yaml"
    fi
    
    # Create environment file template
    cat > "$CONFIG_DIR/oceanproxy.env" << 'EOF'
# OceanProxy Environment Configuration
ENVIRONMENT=production

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# SECURITY - CHANGE THESE IMMEDIATELY!
BEARER_TOKEN=CHANGE-THIS-TO-A-SECURE-RANDOM-TOKEN-AT-LEAST-32-CHARACTERS-LONG
JWT_SECRET=CHANGE-THIS-TO-ANOTHER-SECURE-RANDOM-KEY-FOR-JWT-TOKENS

# Provider API Keys - GET THESE FROM YOUR PROVIDER ACCOUNTS
PROXIES_FO_API_KEY=your-proxies-fo-api-key-here
NETTIFY_API_KEY=your-nettify-api-key-here

# Proxy Configuration
PROXY_DOMAIN=oceanproxy.io
PROXY_START_PORT=10000
PROXY_END_PORT=30000

# Database Configuration
DATABASE_DRIVER=json
DATABASE_DSN=/var/log/oceanproxy/proxies.json

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json

# Directories
PROXY_CONFIG_DIR=/etc/3proxy
PROXY_LOG_DIR=/var/log/oceanproxy
PROXY_SCRIPT_DIR=/opt/oceanproxy/scripts
NGINX_CONF_DIR=/etc/nginx/conf.d
EOF
    
    chown "$APP_USER:$APP_USER" "$CONFIG_DIR/oceanproxy.env"
    chmod 600 "$CONFIG_DIR/oceanproxy.env"
    
    # Copy scripts directory
    if [[ -d "$SOURCE_DIR/scripts" ]]; then
        cp -r "$SOURCE_DIR/scripts" "$APP_DIR/"
        chown -R "$APP_USER:$APP_USER" "$APP_DIR/scripts"
        find "$APP_DIR/scripts" -name "*.sh" -exec chmod +x {} \;
    fi
    
    COMPONENT_STATUS["config"]="installed"
    log_success "Configuration files installed"
    log_warning "IMPORTANT: Edit $CONFIG_DIR/oceanproxy.env with your API keys and secure tokens"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    cat > /etc/systemd/system/oceanproxy.service << EOF
[Unit]
Description=OceanProxy - White-label HTTP Proxy Service
Documentation=https://github.com/je265/oceanproxy
After=network.target network-online.target
Wants=network-online.target
Requires=nginx.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=$APP_DIR
ExecStart=/usr/local/bin/oceanproxy
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Environment
Environment=ENVIRONMENT=production
EnvironmentFile=-$CONFIG_DIR/oceanproxy.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=strict
ReadWritePaths=$LOG_DIR /etc/3proxy $DATA_DIR
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768
TasksMax=32768

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=oceanproxy

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable oceanproxy
    COMPONENT_STATUS["systemd"]="installed"
    log_success "Systemd service installed and enabled"
}

# Configure nginx
configure_nginx() {
    log_info "Configuring nginx with proper stream support..."
    
    # Backup original nginx.conf
    cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup
    
    # Create new nginx.conf with proper stream configuration
    cat > /etc/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;

    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;

    server {
        listen 80 default_server;
        server_name _;

        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }

        location /api/ {
            proxy_pass http://127.0.0.1:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            proxy_connect_timeout 30s;
            proxy_send_timeout 30s;
            proxy_read_timeout 30s;
        }

        location ~ ^/(plan|nettify)/ {
            proxy_pass http://127.0.0.1:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}

stream {
    log_format proxy '$remote_addr [$time_local] $protocol $status '
                     '$bytes_sent $bytes_received $session_time';

    upstream oceanproxy_usa {
        least_conn;
        server 127.0.0.1:10001 backup;
    }
    
    server {
        listen 1337;
        proxy_pass oceanproxy_usa;
        proxy_timeout 60s;
        proxy_responses 1;
        
        access_log /var/log/nginx/usa_proxy.log proxy;
        error_log /var/log/nginx/usa_proxy_error.log;
    }
    
    upstream oceanproxy_eu {
        least_conn;
        server 127.0.0.1:16001 backup;
    }
    
    server {
        listen 1338;
        proxy_pass oceanproxy_eu;
        proxy_timeout 60s;
        proxy_responses 1;
        
        access_log /var/log/nginx/eu_proxy.log proxy;
        error_log /var/log/nginx/eu_proxy_error.log;
    }
    
    upstream oceanproxy_alpha {
        least_conn;
        server 127.0.0.1:22001 backup;
    }
    
    server {
        listen 9876;
        proxy_pass oceanproxy_alpha;
        proxy_timeout 60s;
        proxy_responses 1;
        
        access_log /var/log/nginx/alpha_proxy.log proxy;
        error_log /var/log/nginx/alpha_proxy_error.log;
    }

    # Additional mapped upstreams for unique subdomain ports
    upstream oceanproxy_beta {
        least_conn;
        server 127.0.0.1:24001 backup;
    }

    server {
        listen 8765;
        proxy_pass oceanproxy_beta;
        proxy_timeout 60s;
        proxy_responses 1;

        access_log /var/log/nginx/beta_proxy.log proxy;
        error_log /var/log/nginx/beta_proxy_error.log;
    }

    upstream oceanproxy_datacenter {
        least_conn;
        server 127.0.0.1:12001 backup;
    }

    server {
        listen 1339;
        proxy_pass oceanproxy_datacenter;
        proxy_timeout 60s;
        proxy_responses 1;

        access_log /var/log/nginx/datacenter_proxy.log proxy;
        error_log /var/log/nginx/datacenter_proxy_error.log;
    }

    upstream oceanproxy_isp {
        least_conn;
        server 127.0.0.1:14001 backup;
    }

    server {
        listen 1340;
        proxy_pass oceanproxy_isp;
        proxy_timeout 60s;
        proxy_responses 1;

        access_log /var/log/nginx/isp_proxy.log proxy;
        error_log /var/log/nginx/isp_proxy_error.log;
    }

    upstream oceanproxy_mobile {
        least_conn;
        server 127.0.0.1:26001 backup;
    }

    server {
        listen 7654;
        proxy_pass oceanproxy_mobile;
        proxy_timeout 60s;
        proxy_responses 1;

        access_log /var/log/nginx/mobile_proxy.log proxy;
        error_log /var/log/nginx/mobile_proxy_error.log;
    }

    upstream oceanproxy_unlimited {
        least_conn;
        server 127.0.0.1:28001 backup;
    }

    server {
        listen 6543;
        proxy_pass oceanproxy_unlimited;
        proxy_timeout 60s;
        proxy_responses 1;

        access_log /var/log/nginx/unlimited_proxy.log proxy;
        error_log /var/log/nginx/unlimited_proxy_error.log;
    }

    include /etc/nginx/conf.d/*.stream;
}
EOF
    
    if nginx -t; then
        log_success "Nginx configuration test passed"
    else
        log_error "Nginx configuration test failed"
        cp /etc/nginx/nginx.conf.backup /etc/nginx/nginx.conf
        return 1
    fi
    
    systemctl restart nginx
    log_success "Nginx configured with stream support"
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        ufw --force enable
        ufw allow 22/tcp      # SSH
        ufw allow 80/tcp      # HTTP
        ufw allow 443/tcp     # HTTPS
        ufw allow 8080/tcp    # OceanProxy API
        ufw allow 1337/tcp    # USA proxy port
        ufw allow 1338/tcp    # EU proxy port
        ufw allow 9876/tcp    # Alpha proxy port
        ufw allow 8765/tcp    # Beta proxy port
        ufw allow 1339/tcp    # Datacenter proxy port
        ufw allow 7654/tcp    # Mobile proxy port
        ufw allow 6543/tcp    # Unlimited proxy port
        ufw allow 10000:30000/tcp  # Local proxy instance ports
        
        COMPONENT_STATUS["firewall"]="installed"
        log_success "UFW firewall rules configured"
    else
        log_warning "UFW not found. Please configure firewall manually:"
        log_info "Required ports: 22, 80, 443, 8080, 1337, 1338, 9876, 10000-30000"
    fi
}

# System optimization
optimize_system() {
    log_info "Optimizing system settings..."
    
    # Increase file descriptor limits
    cat >> /etc/security/limits.conf << EOF
$APP_USER soft nofile 65536
$APP_USER hard nofile 65536
* soft nofile 65536
* hard nofile 65536
EOF
    
    # Optimize network settings
    cat >> /etc/sysctl.conf << EOF

# OceanProxy network optimizations
net.core.somaxconn = 65536
net.ipv4.tcp_max_syn_backlog = 65536
net.ipv4.tcp_keepalive_time = 600
net.ipv4.tcp_keepalive_intvl = 60
net.ipv4.tcp_keepalive_probes = 3
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 65536 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
EOF
    
    sysctl -p
    
    COMPONENT_STATUS["optimization"]="installed"
    log_success "System optimizations applied"
}

# Start services
start_services() {
    log_info "Starting services..."
    
    # Start Redis
    systemctl enable redis-server 2>/dev/null || systemctl enable redis
    systemctl start redis-server 2>/dev/null || systemctl start redis
    
    # Start OceanProxy
    systemctl start oceanproxy
    
    # Wait a moment for startup
    sleep 3
    
    # Check service status
    if systemctl is-active --quiet oceanproxy; then
        log_success "OceanProxy service started successfully"
    else
        log_error "Failed to start OceanProxy service"
        log_info "Checking service status..."
        systemctl status oceanproxy --no-pager
        log_info "Checking logs..."
        journalctl -u oceanproxy --no-pager -n 20
        return 1
    fi
    
    # Verify nginx is running
    if systemctl is-active --quiet nginx; then
        log_success "Nginx service running"
    else
        log_warning "Nginx service not running - attempting to start..."
        systemctl start nginx
    fi
}

# Full installation
full_installation() {
    log_info "🚀 Starting full OceanProxy installation..."
    
    detect_os
    
    echo
    log_info "This will install all OceanProxy components:"
    echo "  • System dependencies"
    echo "  • Nginx with stream module"
    echo "  • 3proxy"
    echo "  • Go programming language"
    echo "  • User accounts and directories"
    echo "  • OceanProxy application"
    echo "  • Configuration files"
    echo "  • Systemd service"
    echo "  • Firewall rules"
    echo "  • System optimizations"
    echo
    read -p "Continue with full installation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return
    fi
    
    echo
    log_info "🔄 Starting installation process..."
    
    # Run all installation steps
    install_basic_dependencies && sleep 1
    install_nginx && sleep 1
    install_3proxy && sleep 1
    install_go && sleep 1
    setup_user_and_directories && sleep 1
    build_and_install_oceanproxy && sleep 1
    install_config && sleep 1
    install_systemd_service && sleep 1
    configure_nginx && sleep 1
    configure_firewall && sleep 1
    optimize_system && sleep 1
    start_services
    
    echo
    log_success "🎉 Full installation completed!"
    display_installation_summary
}

# Handle component selection
handle_component_selection() {
    case $1 in
        1) install_basic_dependencies ;;
        2) install_nginx ;;
        3) install_3proxy ;;
        4) install_go ;;
        5) setup_user_and_directories ;;
        6) build_and_install_oceanproxy ;;
        7) install_config ;;
        8) install_systemd_service ;;
        9) configure_firewall ;;
        10) optimize_system ;;
        11) 
            log_info "Refreshing component status..."
            check_component_status
            sleep 2
            ;;
        12) repair_component_menu ;;
        *) log_error "Invalid selection" ;;
    esac
    
    if [[ $1 -ge 1 && $1 -le 10 ]]; then
        echo
        read -p "Press Enter to continue..."
    fi
}

# Repair component menu
repair_component_menu() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}              REPAIR COMPONENT                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    check_component_status
    
    echo -e "${WHITE}Select component to repair/reinstall:${NC}"
    echo
    echo -e "  ${CYAN}1)${NC}  $(get_status_icon "${COMPONENT_STATUS["nginx"]}") Nginx Configuration"
    echo -e "  ${CYAN}2)${NC}  $(get_status_icon "${COMPONENT_STATUS["3proxy"]}") 3proxy Binary"
    echo -e "  ${CYAN}3)${NC}  $(get_status_icon "${COMPONENT_STATUS["oceanproxy"]}") OceanProxy Binary"
    echo -e "  ${CYAN}4)${NC}  $(get_status_icon "${COMPONENT_STATUS["systemd"]}") Systemd Service"
    echo -e "  ${CYAN}5)${NC}  Fix Permissions"
    echo -e "  ${CYAN}6)${NC}  Reset Configuration"
    echo -e "  ${CYAN}0)${NC}  ← Back"
    echo
    
    read -p "Select option [0-6]: " repair_choice
    
    case $repair_choice in
        1) configure_nginx ;;
        2) install_3proxy ;;
        3) build_and_install_oceanproxy ;;
        4) install_systemd_service ;;
        5) fix_permissions ;;
        6) reset_configuration ;;
        0) return ;;
        *) log_error "Invalid selection" ;;
    esac
    
    echo
    read -p "Press Enter to continue..."
}

# Fix permissions
fix_permissions() {
    log_info "Fixing file permissions..."
    
    chown -R "$APP_USER:$APP_USER" "$APP_DIR" "$LOG_DIR" "$DATA_DIR"
    chown -R "$APP_USER:$APP_USER" /etc/3proxy
    chmod 755 "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    chmod 755 /etc/3proxy
    chmod 600 "$CONFIG_DIR/oceanproxy.env" 2>/dev/null || true
    chmod +x /usr/local/bin/oceanproxy 2>/dev/null || true
    chmod +x /usr/local/bin/oceanproxy-cli 2>/dev/null || true
    
    log_success "Permissions fixed"
}

# Reset configuration
reset_configuration() {
    read -p "This will reset all configuration files. Continue? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        install_config
        log_success "Configuration reset completed"
    fi
}

# Service management functions
handle_service_management() {
    case $1 in
        1) start_all_services ;;
        2) stop_all_services ;;
        3) restart_all_services ;;
        4) show_service_status ;;
        5) view_oceanproxy_logs ;;
        6) view_nginx_logs ;;
        7) test_installation ;;
        8) edit_configuration ;;
        *) log_error "Invalid selection" ;;
    esac
    
    if [[ $1 -ge 1 && $1 -le 8 ]]; then
        echo
        read -p "Press Enter to continue..."
    fi
}

# Start all services
start_all_services() {
    log_info "Starting all services..."
    
    systemctl start nginx
    systemctl start redis-server 2>/dev/null || systemctl start redis
    systemctl start oceanproxy
    
    log_success "All services started"
}

# Stop all services
stop_all_services() {
    log_info "Stopping all services..."
    
    systemctl stop oceanproxy
    systemctl stop nginx
    
    log_success "All services stopped"
}

# Restart all services
restart_all_services() {
    log_info "Restarting all services..."
    
    systemctl restart nginx
    systemctl restart redis-server 2>/dev/null || systemctl restart redis
    systemctl restart oceanproxy
    
    log_success "All services restarted"
}

# Show service status
show_service_status() {
    echo -e "${WHITE}Detailed Service Status:${NC}"
    echo
    
    echo -e "${CYAN}OceanProxy Service:${NC}"
    systemctl status oceanproxy --no-pager -l
    echo
    
    echo -e "${CYAN}Nginx Service:${NC}"
    systemctl status nginx --no-pager -l
    echo
    
    echo -e "${CYAN}Redis Service:${NC}"
    systemctl status redis-server --no-pager -l 2>/dev/null || systemctl status redis --no-pager -l
}

# View logs
view_oceanproxy_logs() {
    echo -e "${WHITE}OceanProxy Logs (last 50 lines):${NC}"
    echo
    journalctl -u oceanproxy -n 50 --no-pager
}

view_nginx_logs() {
    echo -e "${WHITE}Nginx Error Logs (last 20 lines):${NC}"
    echo
    tail -20 /var/log/nginx/error.log 2>/dev/null || echo "No nginx error logs found"
    
    echo
    echo -e "${WHITE}Nginx Access Logs (last 10 lines):${NC}"
    echo
    tail -10 /var/log/nginx/access.log 2>/dev/null || echo "No nginx access logs found"
}

# Test installation
test_installation() {
    log_info "Testing OceanProxy installation..."
    echo
    
    # Test 1: Check if binaries exist and are executable
    echo -e "${CYAN}Test 1: Binary Installation${NC}"
    if [[ -x "/usr/local/bin/oceanproxy" ]]; then
        echo "  ✅ OceanProxy binary installed"
    else
        echo "  ❌ OceanProxy binary missing"
    fi
    
    if [[ -x "/usr/local/bin/3proxy" ]]; then
        echo "  ✅ 3proxy binary installed"
    else
        echo "  ❌ 3proxy binary missing"
    fi
    
    # Test 2: Check services
    echo -e "${CYAN}Test 2: Service Status${NC}"
    if systemctl is-active --quiet oceanproxy; then
        echo "  ✅ OceanProxy service running"
    else
        echo "  ❌ OceanProxy service not running"
    fi
    
    if systemctl is-active --quiet nginx; then
        echo "  ✅ Nginx service running"
    else
        echo "  ❌ Nginx service not running"
    fi
    
    # Test 3: Check API connectivity
    echo -e "${CYAN}Test 3: API Connectivity${NC}"
    if curl -s --connect-timeout 5 http://localhost:8080/health >/dev/null 2>&1; then
        echo "  ✅ API health endpoint responding"
    else
        echo "  ❌ API health endpoint not responding"
    fi
    
    # Test 4: Check proxy ports
    echo -e "${CYAN}Test 4: Proxy Ports${NC}"
    for port in 1337 1338 9876 7777; do
        if netstat -ln 2>/dev/null | grep -q ":$port "; then
            echo "  ✅ Port $port is listening"
        else
            echo "  ❌ Port $port not listening"
        fi
    done
    
    # Test 5: Configuration files
    echo -e "${CYAN}Test 5: Configuration Files${NC}"
    if [[ -f "$CONFIG_DIR/oceanproxy.env" ]]; then
        echo "  ✅ Environment configuration exists"
    else
        echo "  ❌ Environment configuration missing"
    fi
    
    if [[ -f "/etc/nginx/nginx.conf" ]] && grep -q "stream" /etc/nginx/nginx.conf; then
        echo "  ✅ Nginx stream configuration present"
    else
        echo "  ❌ Nginx stream configuration missing"
    fi
    
    echo
    log_info "Installation test completed"
}

# Edit configuration
edit_configuration() {
    echo -e "${WHITE}Configuration Files:${NC}"
    echo
    echo -e "  ${CYAN}1)${NC} Main Environment ($CONFIG_DIR/oceanproxy.env)"
    echo -e "  ${CYAN}2)${NC} Proxy Plans ($CONFIG_DIR/proxy-plans.yaml)"
    echo -e "  ${CYAN}3)${NC} Regions ($CONFIG_DIR/regions.yaml)"
    echo -e "  ${CYAN}4)${NC} Nginx Configuration (/etc/nginx/nginx.conf)"
    echo -e "  ${CYAN}0)${NC} Back"
    echo
    
    read -p "Select file to edit [0-4]: " edit_choice
    
    case $edit_choice in
        1) nano "$CONFIG_DIR/oceanproxy.env" ;;
        2) nano "$CONFIG_DIR/proxy-plans.yaml" ;;
        3) nano "$CONFIG_DIR/regions.yaml" ;;
        4) nano /etc/nginx/nginx.conf ;;
        0) return ;;
        *) log_error "Invalid selection" ;;
    esac
    
    if [[ $edit_choice -ge 1 && $edit_choice -le 4 ]]; then
        echo
        read -p "Restart services to apply changes? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if [[ $edit_choice -eq 4 ]]; then
                nginx -t && systemctl restart nginx
            fi
            systemctl restart oceanproxy
            log_success "Services restarted"
        fi
    fi
}

# Run diagnostics
run_diagnostics() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}            SYSTEM DIAGNOSTICS                  ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    # System info
    echo -e "${WHITE}System Information:${NC}"
    echo "  OS: $(lsb_release -d 2>/dev/null | cut -f2 || echo 'Unknown')"
    echo "  Kernel: $(uname -r)"
    echo "  Architecture: $(uname -m)"
    echo "  Uptime: $(uptime -p 2>/dev/null || echo 'Unknown')"
    echo
    
    # Network connectivity
    echo -e "${WHITE}Network Tests:${NC}"
    test_urls=("https://github.com" "https://nginx.org" "https://golang.org")
    for url in "${test_urls[@]}"; do
        if timeout 5 curl -s --head "$url" >/dev/null 2>&1; then
            echo "  ✅ $url - OK"
        else
            echo "  ❌ $url - FAILED"
        fi
    done
    echo
    
    # Disk space
    echo -e "${WHITE}Disk Space:${NC}"
    df -h / | tail -1 | while read filesystem size used available percent mountpoint; do
        echo "  Available: $available ($percent used)"
    done
    echo
    
    # Memory usage
    echo -e "${WHITE}Memory Usage:${NC}"
    free -h | grep Mem | awk '{print "  Used: "$3" / "$2" ("$3/$2*100"% used)"}'
    echo
    
    read -p "Press Enter to continue..."
}

# View installation logs
view_installation_logs() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}            INSTALLATION LOGS                   ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    echo -e "${WHITE}Recent System Logs (last 20 lines):${NC}"
    echo
    journalctl --no-pager -n 20 | grep -E "(oceanproxy|nginx|3proxy|install)" || echo "No relevant logs found"
    echo
    
    echo -e "${WHITE}Recent Package Installations:${NC}"
    echo
    if [[ -f "/var/log/dpkg.log" ]]; then
        tail -10 /var/log/dpkg.log | grep -E "(install|configure)" || echo "No recent installations"
    else
        echo "Package log not found"
    fi
    echo
    
    read -p "Press Enter to continue..."
}

# Uninstall OceanProxy
uninstall_oceanproxy() {
    clear
    echo -e "${RED}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║${WHITE}              UNINSTALL OCEANPROXY              ${RED}║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    log_warning "This will completely remove OceanProxy from your system:"
    echo "  • Stop all OceanProxy services"
    echo "  • Remove binaries and configuration files"
    echo "  • Remove systemd service"
    echo "  • Remove user account"
    echo "  • Remove log files"
    echo
    echo -e "${RED}This action cannot be undone!${NC}"
    echo
    
    read -p "Are you sure you want to uninstall? (type 'yes' to confirm): " confirm
    
    if [[ "$confirm" == "yes" ]]; then
        log_info "Starting uninstallation..."
        
        # Stop services
        systemctl stop oceanproxy 2>/dev/null || true
        systemctl disable oceanproxy 2>/dev/null || true
        
        # Remove systemd service
        rm -f /etc/systemd/system/oceanproxy.service
        systemctl daemon-reload
        
        # Remove binaries
        rm -f /usr/local/bin/oceanproxy
        rm -f /usr/local/bin/oceanproxy-cli
        
        # Remove directories
        rm -rf "$APP_DIR"
        rm -rf "$CONFIG_DIR"
        rm -rf "$LOG_DIR"
        rm -rf "$DATA_DIR"
        rm -rf /etc/3proxy
        
        # Remove user
        userdel "$APP_USER" 2>/dev/null || true
        
        # Clean up nginx configuration (restore backup if exists)
        if [[ -f /etc/nginx/nginx.conf.backup ]]; then
            cp /etc/nginx/nginx.conf.backup /etc/nginx/nginx.conf
            systemctl restart nginx 2>/dev/null || true
        fi
        
        log_success "OceanProxy has been completely uninstalled"
        echo
        log_info "Note: System dependencies (nginx, 3proxy, Go) were not removed"
        log_info "You can remove them manually if no longer needed"
        echo
        read -p "Press Enter to exit..."
        exit 0
    else
        log_info "Uninstallation cancelled"
        sleep 2
    fi
}

# Display installation summary
display_installation_summary() {
    local SERVER_IP=$(curl -s http://ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")
    
    clear
    echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${WHITE}         🎉 INSTALLATION COMPLETED! 🎉         ${GREEN}║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    echo -e "${WHITE}🌐 Network Endpoints:${NC}"
    echo -e "  • API Server: ${CYAN}http://$SERVER_IP:8080${NC}"
    echo -e "  • Health Check: ${CYAN}http://$SERVER_IP:8080/health${NC}"
    echo -e "  • USA Proxy Port: ${CYAN}$SERVER_IP:1337${NC}"
    echo -e "  • EU Proxy Port: ${CYAN}$SERVER_IP:1338${NC}"
    echo -e "  • Datacenter Proxy Port: ${CYAN}$SERVER_IP:1339${NC}"
    echo -e "  • Alpha Proxy Port: ${CYAN}$SERVER_IP:9876${NC}"
    echo -e "  • Beta Proxy Port: ${CYAN}$SERVER_IP:8765${NC}"
    echo -e "  • Mobile Proxy Port: ${CYAN}$SERVER_IP:7654${NC}"
    echo -e "  • Unlimited Proxy Port: ${CYAN}$SERVER_IP:6543${NC}"
    echo
    
    echo -e "${WHITE}🚨 CRITICAL NEXT STEPS:${NC}"
    echo -e "  ${YELLOW}1.${NC} Configure Environment:"
    echo -e "     ${CYAN}sudo nano $CONFIG_DIR/oceanproxy.env${NC}"
    echo -e "     • Set secure BEARER_TOKEN (32+ characters)"
    echo -e "     • Set secure JWT_SECRET"
    echo -e "     • Add your PROXIES_FO_API_KEY"
    echo -e "     • Add your NETTIFY_API_KEY"
    echo -e "     • Update PROXY_DOMAIN to your domain"
    echo
    echo -e "  ${YELLOW}2.${NC} Restart after configuration:"
    echo -e "     ${CYAN}sudo systemctl restart oceanproxy${NC}"
    echo
    echo -e "  ${YELLOW}3.${NC} Test installation:"
    echo -e "     ${CYAN}curl http://localhost:8080/health${NC}"
    echo
    
    echo -e "${WHITE}📋 Useful Commands:${NC}"
    echo -e "  • View logs: ${CYAN}sudo journalctl -u oceanproxy -f${NC}"
    echo -e "  • Restart service: ${CYAN}sudo systemctl restart oceanproxy${NC}"
    echo -e "  • Check status: ${CYAN}sudo systemctl status oceanproxy${NC}"
    echo -e "  • Run installer again: ${CYAN}sudo ./install.sh${NC}"
    echo
    
    read -p "Press Enter to return to main menu..."
}

# Configure secrets and environment variables
verify_env_file() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}         CONFIGURE SECRETS & ENVIRONMENT        ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    local env_file="$CONFIG_DIR/oceanproxy.env"
    
    if [[ ! -f "$env_file" ]]; then
        log_warning "Environment file not found. Creating $env_file..."
        mkdir -p "$CONFIG_DIR"
        cat > "$env_file" << 'EOF'
# OceanProxy Environment Configuration
BEARER_TOKEN=simple123
JWT_SECRET=your-jwt-secret-here
PROXIES_FO_API_KEY=your-proxies-fo-api-key
NETTIFY_API_KEY=your-nettify-api-key
PROXY_DOMAIN=oceanproxy.io
DATABASE_URL=postgres://oceanproxy:password@localhost:5432/oceanproxy
REDIS_URL=redis://localhost:6379
LOG_LEVEL=info
EOF
        chown "$APP_USER:$APP_USER" "$env_file"
        chmod 600 "$env_file"
    fi
    
    echo -e "${WHITE}Current environment file status:${NC}"
    echo -e "  • Location: ${CYAN}$env_file${NC}"
    echo -e "  • Permissions: ${CYAN}$(stat -c '%a' "$env_file" 2>/dev/null || echo 'N/A')${NC}"
    echo
    
    # Check for weak tokens
    local bearer_token=$(grep "^BEARER_TOKEN=" "$env_file" 2>/dev/null | cut -d'=' -f2)
    local jwt_secret=$(grep "^JWT_SECRET=" "$env_file" 2>/dev/null | cut -d'=' -f2)
    
    echo -e "${WHITE}Security Check:${NC}"
    if [[ "$bearer_token" == "simple123" || ${#bearer_token} -lt 32 ]]; then
        echo -e "  • Bearer Token: ${RED}WEAK/DEFAULT${NC} (should be 32+ chars)"
        echo -e "    Current: ${YELLOW}$bearer_token${NC}"
        echo
        read -p "Generate secure 64-char bearer token? (y/n): " gen_bearer
        if [[ "$gen_bearer" =~ ^[Yy]$ ]]; then
            local new_bearer=$(openssl rand -hex 32)
            sed -i "s/^BEARER_TOKEN=.*/BEARER_TOKEN=$new_bearer/" "$env_file"
            echo -e "    ${GREEN}✓ Generated new bearer token${NC}"
        fi
    else
        echo -e "  • Bearer Token: ${GREEN}SECURE${NC}"
    fi
    
    if [[ "$jwt_secret" == "your-jwt-secret-here" || ${#jwt_secret} -lt 32 ]]; then
        echo -e "  • JWT Secret: ${RED}WEAK/DEFAULT${NC} (should be 32+ chars)"
        echo
        read -p "Generate secure 64-char JWT secret? (y/n): " gen_jwt
        if [[ "$gen_jwt" =~ ^[Yy]$ ]]; then
            local new_jwt=$(openssl rand -hex 32)
            sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$new_jwt/" "$env_file"
            echo -e "    ${GREEN}✓ Generated new JWT secret${NC}"
        fi
    else
        echo -e "  • JWT Secret: ${GREEN}SECURE${NC}"
    fi
    
    echo
    echo -e "${WHITE}Provider API Keys:${NC}"
    
    # Check Proxies.fo API key
    local proxies_key=$(grep "^PROXIES_FO_API_KEY=" "$env_file" 2>/dev/null | cut -d'=' -f2)
    if [[ "$proxies_key" == "your-proxies-fo-api-key" || -z "$proxies_key" ]]; then
        echo -e "  • Proxies.fo: ${RED}NOT CONFIGURED${NC}"
        read -p "Enter your Proxies.fo API key (or press Enter to skip): " new_proxies_key
        if [[ -n "$new_proxies_key" ]]; then
            sed -i "s/^PROXIES_FO_API_KEY=.*/PROXIES_FO_API_KEY=$new_proxies_key/" "$env_file"
            echo -e "    ${GREEN}✓ Updated Proxies.fo API key${NC}"
        fi
    else
        echo -e "  • Proxies.fo: ${GREEN}CONFIGURED${NC}"
    fi
    
    # Check Nettify API key
    local nettify_key=$(grep "^NETTIFY_API_KEY=" "$env_file" 2>/dev/null | cut -d'=' -f2)
    if [[ "$nettify_key" == "your-nettify-api-key" || -z "$nettify_key" ]]; then
        echo -e "  • Nettify: ${RED}NOT CONFIGURED${NC}"
        read -p "Enter your Nettify API key (or press Enter to skip): " new_nettify_key
        if [[ -n "$new_nettify_key" ]]; then
            sed -i "s/^NETTIFY_API_KEY=.*/NETTIFY_API_KEY=$new_nettify_key/" "$env_file"
            echo -e "    ${GREEN}✓ Updated Nettify API key${NC}"
        fi
    else
        echo -e "  • Nettify: ${GREEN}CONFIGURED${NC}"
    fi
    
    echo
    echo -e "${WHITE}Domain Configuration:${NC}"
    local proxy_domain=$(grep "^PROXY_DOMAIN=" "$env_file" 2>/dev/null | cut -d'=' -f2)
    echo -e "  • Proxy Domain: ${CYAN}$proxy_domain${NC}"
    read -p "Update proxy domain? (current: $proxy_domain) [Enter to keep]: " new_domain
    if [[ -n "$new_domain" ]]; then
        sed -i "s/^PROXY_DOMAIN=.*/PROXY_DOMAIN=$new_domain/" "$env_file"
        echo -e "    ${GREEN}✓ Updated proxy domain${NC}"
    fi
    
    echo
    echo -e "${GREEN}✓ Environment configuration completed${NC}"
    read -p "Press Enter to continue..."
}

# Test provider connectivity
provider_connectivity_test() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}           PROVIDER CONNECTIVITY TEST           ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════╝${NC}"
    echo
    
    local env_file="$CONFIG_DIR/oceanproxy.env"
    
    if [[ ! -f "$env_file" ]]; then
        log_error "Environment file not found: $env_file"
        read -p "Press Enter to continue..."
        return 1
    fi
    
    # Source the environment file
    source "$env_file"
    
    echo -e "${WHITE}Testing Proxies.fo API connectivity...${NC}"
    if [[ -n "$PROXIES_FO_API_KEY" && "$PROXIES_FO_API_KEY" != "your-proxies-fo-api-key" ]]; then
        local proxies_response=$(curl -s -H "X-Api-Auth: $PROXIES_FO_API_KEY" https://app.proxies.fo/api/me)
        if echo "$proxies_response" | grep -q '"Success":true'; then
            echo -e "  ${GREEN}✓ Proxies.fo API: Connected${NC}"
            local balance=$(echo "$proxies_response" | grep -o '"BalanceFormatted":"[^"]*"' | cut -d'"' -f4)
            echo -e "    Balance: ${CYAN}$balance${NC}"
        else
            echo -e "  ${RED}✗ Proxies.fo API: Failed${NC}"
            echo -e "    Response: ${YELLOW}$proxies_response${NC}"
        fi
    else
        echo -e "  ${YELLOW}⚠ Proxies.fo API key not configured${NC}"
    fi
    
    echo
    echo -e "${WHITE}Testing Nettify API connectivity...${NC}"
    if [[ -n "$NETTIFY_API_KEY" && "$NETTIFY_API_KEY" != "your-nettify-api-key" ]]; then
        local nettify_response=$(curl -s -H "Authorization: Bearer $NETTIFY_API_KEY" https://api.nettify.xyz/balance)
        if echo "$nettify_response" | grep -q '"balance":'; then
            echo -e "  ${GREEN}✓ Nettify API: Connected${NC}"
            local balance=$(echo "$nettify_response" | grep -o '"balance":[0-9.]*' | cut -d':' -f2)
            echo -e "    Balance: ${CYAN}\$${balance}${NC}"
        else
            echo -e "  ${RED}✗ Nettify API: Failed${NC}"
            echo -e "    Response: ${YELLOW}$nettify_response${NC}"
        fi
    else
        echo -e "  ${YELLOW}⚠ Nettify API key not configured${NC}"
    fi
    
    echo
    echo -e "${WHITE}Testing OceanProxy API...${NC}"
    if systemctl is-active --quiet oceanproxy; then
        local health_response=$(curl -s http://localhost:8080/ready 2>/dev/null)
        if echo "$health_response" | grep -q '"status":"ready"'; then
            echo -e "  ${GREEN}✓ OceanProxy API: Running${NC}"
            # Show database status
            if echo "$health_response" | grep -q '"database":{"status":"healthy"'; then
                echo -e "    Database: ${GREEN}✓ Healthy${NC}"
            else
                echo -e "    Database: ${YELLOW}⚠ Check required${NC}"
            fi
            # Show provider status
            if echo "$health_response" | grep -q '"providers":{"status":"healthy"'; then
                echo -e "    Providers: ${GREEN}✓ Healthy${NC}"
            else
                echo -e "    Providers: ${YELLOW}⚠ Check required${NC}"
            fi
        else
            echo -e "  ${RED}✗ OceanProxy API: Not responding${NC}"
            echo -e "    Try: ${CYAN}sudo systemctl restart oceanproxy${NC}"
        fi
    else
        echo -e "  ${RED}✗ OceanProxy service not running${NC}"
        echo -e "    Try: ${CYAN}sudo systemctl start oceanproxy${NC}"
    fi
    
    echo
    read -p "Press Enter to continue..."
}

# Main program loop
main() {
    check_root
    detect_os
    
    while true; do
        show_main_menu
        read -p "Select option [0-8]: " choice
        
        case $choice in
            1) full_installation ;;
            2) 
                while true; do
                    show_component_menu
                    read -p "Select component [0-12]: " comp_choice
                    if [[ $comp_choice -eq 0 ]]; then
                        break
                    else
                        handle_component_selection "$comp_choice"
                    fi
                done
                ;;
            3) show_system_info ;;
            4) run_diagnostics ;;
            5) view_installation_logs ;;
            6) 
                while true; do
                    show_service_menu
                    read -p "Select option [0-8]: " svc_choice
                    if [[ $svc_choice -eq 0 ]]; then
                        break
                    else
                        handle_service_management "$svc_choice"
                    fi
                done
                ;;
            7) uninstall_oceanproxy ;;
            8) verify_env_file; provider_connectivity_test ;;
            0) 
                echo
                log_info "Thank you for using OceanProxy installer!"
                exit 0
                ;;
            *) 
                log_error "Invalid selection. Please choose 0-7."
                sleep 2
                ;;
        esac
    done
}

# Run the main program
main "$@"