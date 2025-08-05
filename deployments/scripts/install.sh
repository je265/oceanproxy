#!/bin/bash

# OceanProxy Installation Script
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
    
    # Check if supported OS
    case $OS in
        "Ubuntu"|"Debian GNU/Linux")
            PKG_MANAGER="apt"
            ;;
        "CentOS Linux"|"Red Hat Enterprise Linux"|"Rocky Linux")
            PKG_MANAGER="yum"
            ;;
        *)
            log_warning "Unsupported OS: $OS. Installation may not work correctly."
            PKG_MANAGER="apt"
            ;;
    esac
}

# Install system dependencies
install_dependencies() {
    log_info "Installing system dependencies..."
    
    case $PKG_MANAGER in
        "apt")
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
                3proxy \
                nginx \
                nginx-module-stream \
                redis-server \
                postgresql-client \
                logrotate \
                fail2ban
            ;;
        "yum")
            yum update -y
            yum install -y \
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
                nginx \
                redis \
                postgresql \
                logrotate \
                fail2ban
            
            # Install 3proxy from source on CentOS/RHEL
            install_3proxy_from_source
            ;;
    esac
    
    log_success "System dependencies installed"
}

# Install 3proxy from source (for CentOS/RHEL)
install_3proxy_from_source() {
    log_info "Installing 3proxy from source..."
    
    cd /tmp
    wget https://github.com/z3APA3A/3proxy/archive/0.9.4.tar.gz
    tar -xzf 0.9.4.tar.gz
    cd 3proxy-0.9.4
    
    make -f Makefile.Linux
    cp bin/3proxy /usr/local/bin/
    chmod +x /usr/local/bin/3proxy
    
    # Create symlink for compatibility
    ln -sf /usr/local/bin/3proxy /usr/bin/3proxy
    
    cd /
    rm -rf /tmp/3proxy-0.9.4 /tmp/0.9.4.tar.gz
    
    log_success "3proxy installed from source"
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
    chown -R "$APP_USER:$APP_GROUP" "$APP_DIR" "$LOG_DIR" "$DATA_DIR"
    chown -R "$APP_USER:$APP_GROUP" /etc/3proxy
    chmod 755 "$APP_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    chmod 755 /etc/3proxy
    
    log_success "User and directories created"
}

# Install Go (if not present)
install_go() {
    if command -v go &> /dev/null; then
        log_info "Go is already installed: $(go version)"
        return
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
    wget "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    
    log_success "Go installed: $(/usr/local/go/bin/go version)"
}

# Download and install OceanProxy binary
install_oceanproxy() {
    log_info "Installing OceanProxy..."
    
    # For now, we'll assume the binary is built locally
    # In production, you would download from releases
    if [[ -f "./bin/oceanproxy" ]]; then
        cp "./bin/oceanproxy" /usr/local/bin/
        chmod +x /usr/local/bin/oceanproxy
        log_success "OceanProxy binary installed"
    else
        log_error "OceanProxy binary not found. Please run 'make build' first."
        exit 1
    fi
}

# Install configuration files
install_config() {
    log_info "Installing configuration files..."
    
    # Copy main configuration
    if [[ -f "./configs/config.yaml" ]]; then
        cp "./configs/config.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_GROUP" "$CONFIG_DIR/config.yaml"
    fi
    
    # Copy region and plan configurations
    if [[ -f "./configs/regions.yaml" ]]; then
        cp "./configs/regions.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_GROUP" "$CONFIG_DIR/regions.yaml"
    fi
    
    if [[ -f "./configs/proxy-plans.yaml" ]]; then
        cp "./configs/proxy-plans.yaml" "$CONFIG_DIR/"
        chown "$APP_USER:$APP_GROUP" "$CONFIG_DIR/proxy-plans.yaml"
    fi
    
    # Create environment file template
    cat > "$CONFIG_DIR/oceanproxy.env" << 'EOF'
# OceanProxy Environment Configuration
ENVIRONMENT=production
BEARER_TOKEN=change-this-secure-token
JWT_SECRET=change-this-jwt-secret
PROXIES_FO_API_KEY=your-proxies-fo-api-key
NETTIFY_API_KEY=your-nettify-api-key
LOG_LEVEL=info
LOG_FORMAT=json
EOF
    
    chown "$APP_USER:$APP_GROUP" "$CONFIG_DIR/oceanproxy.env"
    chmod 600 "$CONFIG_DIR/oceanproxy.env"
    
    log_success "Configuration files installed"
    log_warning "Please edit $CONFIG_DIR/oceanproxy.env with your API keys"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    if [[ -f "./deployments/systemd/oceanproxy.service" ]]; then
        cp "./deployments/systemd/oceanproxy.service" /etc/systemd/system/
        systemctl daemon-reload
        systemctl enable oceanproxy
        log_success "Systemd service installed and enabled"
    else
        log_error "Systemd service file not found"
        exit 1
    fi
}

# Configure nginx
configure_nginx() {
    log_info "Configuring nginx..."
    
    # Enable stream module
    if ! grep -q "load_module.*stream" /etc/nginx/nginx.conf; then
        sed -i '1i load_module modules/ngx_stream_module.so;' /etc/nginx/nginx.conf
    fi
    
    # Add stream block if not present
    if ! grep -q "stream {" /etc/nginx/nginx.conf; then
        cat >> /etc/nginx/nginx.conf << 'EOF'

# Stream configuration for proxy load balancing
stream {
    include /etc/nginx/conf.d/*.conf;
}
EOF
    fi
    
    # Copy nginx configuration if available
    if [[ -f "./build/nginx/nginx.conf" ]]; then
        cp "./build/nginx/nginx.conf" /etc/nginx/nginx.conf.oceanproxy
        log_info "Custom nginx configuration available at /etc/nginx/nginx.conf.oceanproxy"
    fi
    
    # Test nginx configuration
    if nginx -t; then
        systemctl enable nginx
        systemctl restart nginx
        log_success "Nginx configured and restarted"
    else
        log_error "Nginx configuration test failed"
        exit 1
    fi
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        # Configure UFW
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
        log_warning "No supported firewall found. Please configure manually."
    fi
}

# Configure logrotate
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
    create 644 $APP_USER $APP_GROUP
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
    create 644 $APP_USER $APP_GROUP
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
    systemctl enable redis-server || systemctl enable redis
    systemctl start redis-server || systemctl start redis
    
    # Start OceanProxy
    systemctl start oceanproxy
    
    # Check service status
    if systemctl is-active --quiet oceanproxy; then
        log_success "OceanProxy service started successfully"
    else
        log_error "Failed to start OceanProxy service"
        systemctl status oceanproxy
        exit 1
    fi
}

# Display installation summary
display_summary() {
    log_success "🌊 OceanProxy installation completed successfully!"
    echo
    echo "Configuration:"
    echo "  - Application directory: $APP_DIR"
    echo "  - Configuration directory: $CONFIG_DIR"
    echo "  - Log directory: $LOG_DIR"
    echo "  - Data directory: $DATA_DIR"
    echo
    echo "Services:"
    echo "  - OceanProxy: systemctl status oceanproxy"
    echo "  - Nginx: systemctl status nginx"
    echo "  - Redis: systemctl status redis-server"
    echo
    echo "Next steps:"
    echo "  1. Edit $CONFIG_DIR/oceanproxy.env with your API keys"
    echo "  2. systemctl restart oceanproxy"
    echo "  3. Test the API: curl http://localhost:8080/health"
    echo
    echo "Documentation: https://github.com/je265/oceanproxy"
    echo "Support: support@oceanproxy.io"
}

# Main installation function
main() {
    log_info "🌊 Starting OceanProxy installation..."
    
    check_root
    detect_os
    install_dependencies
    setup_user_and_directories
    install_go
    install_oceanproxy
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