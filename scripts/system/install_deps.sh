#!/bin/bash

# OceanProxy - System Dependencies Installation Script
# Installs all required system dependencies for Ubuntu

set -euo pipefail

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "🌊 Installing OceanProxy system dependencies..."

# Update package list
apt-get update

# Install basic dependencies
apt-get install -y \
    curl \
    wget \
    git \
    jq \
    lsof \
    netstat-nat \
    htop \
    vim \
    unzip \
    ca-certificates \
    gnupg \
    lsb-release

# Install 3proxy
echo "Installing 3proxy..."
apt-get install -y 3proxy

# Install nginx with stream module
echo "Installing nginx..."
apt-get install -y nginx nginx-module-stream

# Install Go (if not already installed)
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    GO_VERSION="1.21.5"
    wget -q "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm "go${GO_VERSION}.linux-amd64.tar.gz"
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# Install PostgreSQL (optional)
echo "Installing PostgreSQL..."
apt-get install -y postgresql postgresql-contrib

# Install Redis (optional)
echo "Installing Redis..."
apt-get install -y redis-server

# Create necessary directories
echo "Creating directories..."
mkdir -p /var/log/oceanproxy
mkdir -p /etc/3proxy
mkdir -p /etc/nginx/conf.d
mkdir -p /opt/oceanproxy

# Set permissions
chown -R www-data:www-data /var/log/oceanproxy
chmod 755 /var/log/oceanproxy
chmod 755 /etc/3proxy

# Configure nginx to load stream module
if ! grep -q "load_module.*stream" /etc/nginx/nginx.conf; then
    sed -i '1i load_module modules/ngx_stream_module.so;' /etc/nginx/nginx.conf
fi

# Add stream block to nginx.conf if not present
if ! grep -q "stream {" /etc/nginx/nginx.conf; then
    cat >> /etc/nginx/nginx.conf << 'EOF'

# Stream configuration for proxy load balancing
stream {
    include /etc/nginx/conf.d/*.conf;
}
EOF
fi

# Enable and start services
systemctl enable nginx
systemctl enable redis-server
systemctl enable postgresql

systemctl start nginx
systemctl start redis-server
systemctl start postgresql

# Configure firewall (if ufw is installed)
if command -v ufw &> /dev/null; then
    echo "Configuring firewall..."
    ufw allow 22/tcp
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 8080/tcp
    ufw allow 1337/tcp
    ufw allow 1338/tcp
    ufw allow 9876/tcp
    ufw allow 10000:20000/tcp
fi

echo "✅ System dependencies installed successfully!"
echo ""
echo "Next steps:"
echo "1. Configure your environment variables in .env"
echo "2. Run 'make setup-dev' to setup development environment"
echo "3. Run 'make build' to build the application"
echo "4. Run 'make run' to start the server"