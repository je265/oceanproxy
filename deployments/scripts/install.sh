#!/bin/bash

# OceanProxy Installation Script - Fixed Version
# This script installs OceanProxy on Ubuntu/Debian systems

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
NC='\033[0m' # No Color

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
    
    log_info "Detected OS: $OS $VER"
}

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
        software-properties-common
    
    log_success "Basic dependencies installed"
}

# Install nginx with stream module
install_nginx() {
    log_info "Installing nginx with stream module..."
    
    # Install nginx
    apt-get install -y nginx
    
    # Check if stream module is available
    if nginx -V 2>&1 | grep -q "stream"; then
        log_success "Nginx with stream module already available"
    else
        log_info "Installing nginx with additional modules..."
        
        # Add official nginx repository for full-featured nginx
        curl -fsSL https://nginx.org/keys/nginx_signing.key | apt-key add -
        echo "deb https://nginx.org/packages/ubuntu/ $(lsb_release -cs) nginx" > /etc/apt/sources.list.d/nginx.list
        
        apt-get update
        apt-get install -y nginx-module-stream 2>/dev/null || {
            log_warning "nginx-module-stream not available via package manager"
            log_info "Using default nginx installation (stream module may need manual compilation)"
        }
    fi
    
    # Enable and start nginx
    systemctl enable nginx
    systemctl start nginx
    
    log_success "Nginx installed"
}

# Install 3proxy from source
install_3proxy() {
    log_info "Installing 3proxy from source..."
    
    # Check if 3proxy is already installed
    if command -v 3proxy &> /dev/null; then
        log_info "3proxy already installed: $(3proxy --version 2>&1 | head -1)"
        return
    fi
    
    # Install build dependencies
    apt-get install -y build-essential
    
    # Download and build 3proxy
    cd /tmp
    
    # Try different download methods
    if ! wget https://github.com/z3APA3A/3proxy/archive/0.9.4.tar.gz; then
        log_info "GitHub download failed, trying alternative..."
        if ! wget https://3proxy.org/0.9.4/3proxy-0.9.4.tar.gz; then
            log_error "Could not download 3proxy source"
            exit 1
        fi
    fi
    
    # Extract and build
    tar -xzf 3proxy-0.9.4.tar.gz 2>/dev/null || tar -xzf 0.9.4.tar.gz
    cd 3proxy-0.9.4 || cd 3proxy-*
    
    # Build 3proxy
    make -f Makefile.Linux
    
    # Install binary
    cp bin/3proxy /usr/local/bin/
    chmod +x /usr/local/bin/3proxy
    
    # Create symlink for compatibility
    ln -sf /usr/local/bin/3proxy /usr/bin/3proxy
    
    # Clean up
    cd /
    rm -rf /tmp/3proxy-* /tmp/0.9.4.tar.gz
    
    log_success "3proxy installed: $(/usr/local/bin/3proxy --version 2>&1 | head -1)"
}

# Create system user and directories
setup_user_and_directories() {
    log_info "Setting up user and directories..."
    
    # Create system user
    if ! id "$APP_USER" &>/dev/null; then
        useradd --system --shell /bin/false --home "$APP_DIR" --create-home "$APP_USER"
        log_success "Created user: $APP_USER"
    else
        log_info "User $APP_USER already exists"
    fi
    
    # Create directories
    mkdir -p "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    mkdir -p /etc/3proxy
    mkdir -p /etc/nginx/conf.d
    
    # Set ownership and permissions
    chown -R "$APP_USER:$APP_USER" "$APP_DIR" "$LOG_DIR" "$DATA_DIR"
    chown -R "$APP_USER:$APP_USER" /etc/3proxy
    chmod 755 "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    chmod 755 /etc/3proxy
    
    log_success "User and directories created"
}

# Install Go
install_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go is already installed: $GO_VERSION"
        
        # Check if version is recent enough (1.19+)
        if [[ $(echo "$GO_VERSION 1.19" | tr " " "\n" | sort -V | head -n1) == "1.19" ]]; then
            return
        else
            log_info "Go version is too old, installing newer version..."
        fi
    fi
    
    log_info "Installing Go..."
    
    GO_VERSION="1.21.5"
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        armv7l)
            GO_ARCH="armv6l"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    cd /tmp
    
    # Download Go
    if ! wget "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"; then
        log_error "Failed to download Go"
        exit 1
    fi
    
    # Remove old Go installation
    rm -rf /usr/local/go
    
    # Install new Go
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    # Add Go to PATH
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    
    log_success "Go installed: $(/usr/local/go/bin/go version)"
}

# Build and install OceanProxy binary
build_and_install_oceanproxy() {
    log_info "Building and installing OceanProxy..."
    
    # Set Go path
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    
    # Find the source directory - use current working directory if go.mod exists
    if [[ -f "$(pwd)/go.mod" ]]; then
        SOURCE_DIR="$(pwd)"
        log_info "Using current directory: $SOURCE_DIR"
    else
        # Search for go.mod in parent directories
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
            exit 1
        fi
        
        log_info "Found source directory: $SOURCE_DIR"
    fi
    
    cd "$SOURCE_DIR"
    
    log_info "Building OceanProxy from source..."
    
    # Ensure go.mod exists and download dependencies
    if [[ ! -f go.mod ]]; then
        log_error "go.mod not found in $SOURCE_DIR"
        exit 1
    fi
    
    # Download dependencies
    /usr/local/go/bin/go mod download
    /usr/local/go/bin/go mod tidy
    
    # Build the main application
    mkdir -p bin
    /usr/local/go/bin/go build -o bin/oceanproxy cmd/server/main.go
    
    # Build the CLI tool
    /usr/local/go/bin/go build -o bin/oceanproxy-cli cmd/cli/main.go
    
    # Install binaries
    cp bin/oceanproxy /usr/local/bin/
    cp bin/oceanproxy-cli /usr/local/bin/
    chmod +x /usr/local/bin/oceanproxy
    chmod +x /usr/local/bin/oceanproxy-cli
    
    log_success "OceanProxy binaries installed"
}

# Install configuration files
install_config() {
    log_info "Installing configuration files..."
    
    # Get the source directory (same logic as build function)
    if [[ -f "$(pwd)/go.mod" ]]; then
        SOURCE_DIR="$(pwd)"
    else
        # Search for go.mod in parent directories
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
    
    # Copy main configuration
    if [[ -f "$SOURCE_DIR/configs/config.yaml" ]]; then
        cp "$SOURCE_DIR/configs/config.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_USER" "$CONFIG_DIR/config.yaml"
    fi
    
    # Copy region and plan configurations
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
    
    log_success "Configuration files installed"
    log_warning "IMPORTANT: Edit $CONFIG_DIR/oceanproxy.env with your API keys and secure tokens"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    # Get the source directory (same logic as other functions)
    if [[ -f "$(pwd)/go.mod" ]]; then
        SOURCE_DIR="$(pwd)"
    else
        # Search for go.mod in parent directories
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
    
    if [[ -f "$SOURCE_DIR/deployments/systemd/oceanproxy.service" ]]; then
        cp "$SOURCE_DIR/deployments/systemd/oceanproxy.service" /etc/systemd/system/
    else
        # Create systemd service file
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
    fi
    
    systemctl daemon-reload
    systemctl enable oceanproxy
    log_success "Systemd service installed and enabled"
}

# Configure nginx
configure_nginx() {
    log_info "Configuring nginx..."
    
    # Backup original nginx.conf
    cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup
    
    # Check if stream module is available
    STREAM_MODULE_AVAILABLE=false
    if nginx -V 2>&1 | grep -q "stream"; then
        STREAM_MODULE_AVAILABLE=true
        log_info "Stream module is available"
    else
        log_warning "Stream module not available - proxy load balancing will be limited"
    fi
    
    # Create basic nginx configuration
    cat > /etc/nginx/nginx.conf << 'EOF'
user www-data;
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
    worker_connections 768;
    use epoll;
    multi_accept on;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    ssl_protocols TLSv1 TLSv1.1 TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;

    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

    gzip on;

    # Default server for API
    server {
        listen 80 default_server;
        listen [::]:80 default_server;

        location /health {
            proxy_pass http://127.0.0.1:8080/health;
        }

        location /api/ {
            proxy_pass http://127.0.0.1:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }

    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
EOF

    # Add stream configuration if module is available
    if [[ "$STREAM_MODULE_AVAILABLE" == "true" ]]; then
        cat >> /etc/nginx/nginx.conf << 'EOF'

# Stream configuration for proxy load balancing
stream {
    log_format basic '$remote_addr [$time_local] '
                     '$protocol $status $bytes_sent $bytes_received '
                     '$session_time';

    # Default upstream (will be populated by OceanProxy)
    upstream oceanproxy_default {
        server 127.0.0.1:10001;
    }

    # Proxy ports
    server {
        listen 1337;
        proxy_pass oceanproxy_default;
        proxy_timeout 1s;
        proxy_responses 1;
        access_log /var/log/nginx/proxy_1337.log basic;
    }

    server {
        listen 1338;
        proxy_pass oceanproxy_default;
        proxy_timeout 1s;
        proxy_responses 1;
        access_log /var/log/nginx/proxy_1338.log basic;
    }

    server {
        listen 9876;
        proxy_pass oceanproxy_default;
        proxy_timeout 1s;
        proxy_responses 1;
        access_log /var/log/nginx/proxy_9876.log basic;
    }

    include /etc/nginx/conf.d/*.conf;
}
EOF
    fi
    
    # Test nginx configuration
    if nginx -t; then
        systemctl restart nginx
        log_success "Nginx configured and restarted"
    else
        log_error "Nginx configuration test failed"
        # Restore backup
        cp /etc/nginx/nginx.conf.backup /etc/nginx/nginx.conf
        exit 1
    fi
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        # Configure UFW
        ufw --force enable
        ufw allow 22/tcp
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw allow 8080/tcp
        ufw allow 1337/tcp
        ufw allow 1338/tcp
        ufw allow 9876/tcp
        ufw allow 10000:30000/tcp
        
        log_success "UFW firewall rules configured"
    elif command -v firewall-cmd &> /dev/null; then
        # Configure firewalld
        firewall-cmd --permanent --add-port=22/tcp
        firewall-cmd --permanent --add-port=80/tcp
        firewall-cmd --permanent --add-port=443/tcp
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --permanent --add-port=1337/tcp
        firewall-cmd --permanent --add-port=1338/tcp
        firewall-cmd --permanent --add-port=9876/tcp
        firewall-cmd --permanent --add-port=10000-30000/tcp
        firewall-cmd --reload
        
        log_success "Firewalld rules configured"
    else
        log_warning "No supported firewall found. Please configure manually:"
        log_info "Allow ports: 22, 80, 443, 8080, 1337, 1338, 9876, 10000-30000"
    fi
}

# Configure log rotation
configure_logrotate() {
    log_info "Configuring log rotation..."
    
    cat > /etc/logrotate.d/oceanproxy << EOF
$LOG_DIR/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 $APP_USER $APP_USER
    postrotate
        systemctl reload oceanproxy
    endscript
}

/etc/3proxy/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 $APP_USER $APP_USER
}
EOF
    
    log_success "Log rotation configured"
}

# Optimize system settings
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

# OceanProxy optimizations
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
        exit 1
    fi
}

# Display installation summary
display_summary() {
    local SERVER_IP=$(curl -s http://ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")
    
    log_success "🌊 OceanProxy installation completed successfully!"
    echo
    echo "Configuration:"
    echo "  - Application directory: $APP_DIR"
    echo "  - Configuration directory: $CONFIG_DIR" 
    echo "  - Log directory: $LOG_DIR"
    echo "  - Data directory: $DATA_DIR"
    echo
    echo "Services:"
    echo "  - OceanProxy: $(systemctl is-active oceanproxy)"
    echo "  - Nginx: $(systemctl is-active nginx)"
    echo "  - Redis: $(systemctl is-active redis-server 2>/dev/null || systemctl is-active redis)"
    echo
    echo "🚨 CRITICAL NEXT STEPS:"
    echo "  1. Edit $CONFIG_DIR/oceanproxy.env with your settings:"
    echo "     - Set secure BEARER_TOKEN and JWT_SECRET"
    echo "     - Add your PROXIES_FO_API_KEY and NETTIFY_API_KEY"
    echo "     - Update PROXY_DOMAIN to your domain"
    echo
    echo "  2. Restart the service after configuration:"
    echo "     sudo systemctl restart oceanproxy"
    echo
    echo "  3. Test the installation:"
    echo "     curl http://localhost:8080/health"
    echo "     curl http://$SERVER_IP:8080/health"
    echo
    echo "  4. Create your first customer:"
    echo "     curl -X POST http://localhost:8080/api/v1/plans \\"
    echo "       -H \"Authorization: Bearer YOUR_TOKEN\" \\"
    echo "       -H \"Content-Type: application/json\" \\"
    echo "       -d '{\"customer_id\":\"test\",\"plan_type\":\"residential\",\"provider\":\"proxies_fo\",\"region\":\"usa\",\"username\":\"testuser\",\"password\":\"testpass\",\"bandwidth\":10}'"
    echo
    echo "📚 Documentation:"
    echo "  - README: https://github.com/je265/oceanproxy"
    echo "  - API Docs: http://$SERVER_IP:8080/docs"
    echo "  - Support: support@oceanproxy.io"
    echo
    echo "🔧 Useful Commands:"
    echo "  - View logs: sudo journalctl -u oceanproxy -f"
    echo "  - Restart service: sudo systemctl restart oceanproxy"
    echo "  - Check status: sudo systemctl status oceanproxy"
    echo "  - Edit config: sudo nano $CONFIG_DIR/oceanproxy.env"
}

# Main installation function
main() {
    log_info "🌊 Starting OceanProxy installation..."
    
    check_root
    detect_os
    install_basic_dependencies
    install_nginx
    install_3proxy
    setup_user_and_directories
    install_go
    build_and_install_oceanproxy
    install_config
    install_systemd_service
    configure_nginx
    configure_firewall
    configure_logrotate
    optimize_system
    start_services
    display_summary
}

# Run installation
main "$@"