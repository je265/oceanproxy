#!/bin/bash

# OceanProxy Installation Diagnostic Script
# Run this to diagnose installation issues

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

echo "🔍 OceanProxy Installation Diagnostic"
echo "====================================="

# System Information
log_info "System Information:"
echo "  OS: $(lsb_release -d 2>/dev/null | cut -f2 || echo 'Unknown')"
echo "  Kernel: $(uname -r)"
echo "  Architecture: $(uname -m)"
echo "  Uptime: $(uptime -p 2>/dev/null || echo 'Unknown')"
echo

# Network Connectivity
log_info "Network Connectivity Tests:"
test_urls=(
    "https://github.com"
    "https://github.com/3proxy/3proxy"
    "https://sourceforge.net"
    "https://nginx.org"
)

for url in "${test_urls[@]}"; do
    if timeout 10 curl -s --head "$url" >/dev/null 2>&1; then
        echo "  ✅ $url - OK"
    else
        echo "  ❌ $url - FAILED"
    fi
done
echo

# DNS Resolution
log_info "DNS Resolution Tests:"
test_domains=(
    "github.com"
    "sourceforge.net"
    "nginx.org"
    "google.com"
)

for domain in "${test_domains[@]}"; do
    if timeout 5 nslookup "$domain" >/dev/null 2>&1; then
        echo "  ✅ $domain - Resolved"
    else
        echo "  ❌ $domain - Failed"
    fi
done
echo

# Package Manager Status
log_info "Package Manager Status:"
if apt-get update >/dev/null 2>&1; then
    echo "  ✅ apt-get update - OK"
else
    echo "  ❌ apt-get update - FAILED"
fi

if dpkg --audit >/dev/null 2>&1; then
    echo "  ✅ dpkg status - OK"
else
    echo "  ❌ dpkg has issues"
fi
echo

# Available Space
log_info "Disk Space:"
df -h / | tail -1 | while read filesystem size used available percent mountpoint; do
    echo "  Filesystem: $filesystem"
    echo "  Total: $size"
    echo "  Used: $used ($percent)"
    echo "  Available: $available"
done
echo

# Memory Usage
log_info "Memory Usage:"
free -h | grep -E "(Mem|Swap)" | while read line; do
    echo "  $line"
done
echo

# Running Processes Related to Installation
log_info "Related Processes:"
processes=(
    "wget"
    "curl"
    "make"
    "gcc"
    "3proxy"
    "nginx"
)

for process in "${processes[@]}"; do
    if pgrep "$process" >/dev/null 2>&1; then
        echo "  🔄 $process is running (PID: $(pgrep "$process" | head -1))"
    else
        echo "  ⭕ $process not running"
    fi
done
echo

# Check if installation files exist
log_info "Installation File Checks:"
locations=(
    "/usr/local/bin/3proxy"
    "/usr/bin/3proxy"
    "/bin/3proxy"
    "/usr/local/bin/oceanproxy"
    "/etc/oceanproxy/"
    "/var/log/oceanproxy/"
)

for location in "${locations[@]}"; do
    if [[ -e "$location" ]]; then
        echo "  ✅ $location exists"
        if [[ -x "$location" ]]; then
            echo "    └── Executable: Yes"
        fi
    else
        echo "  ❌ $location missing"
    fi
done
echo

# Service Status
log_info "Service Status:"
services=(
    "nginx"
    "oceanproxy"
    "redis-server"
)

for service in "${services[@]}"; do
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        echo "  ✅ $service - Active"
    elif systemctl list-unit-files | grep -q "$service"; then
        echo "  ⭕ $service - Installed but not active"
    else
        echo "  ❌ $service - Not installed"
    fi
done
echo

# Network Port Usage
log_info "Network Ports:"
ports=(
    "8080"
    "1337"
    "1338"
    "9876"
    "80"
    "443"
)

for port in "${ports[@]}"; do
    if netstat -ln 2>/dev/null | grep -q ":$port "; then
        service=$(netstat -lnp 2>/dev/null | grep ":$port " | awk '{print $7}' | head -1)
        echo "  🔄 Port $port in use by: ${service:-unknown}"
    else
        echo "  ⭕ Port $port available"
    fi
done
echo

# Installation Logs
log_info "Recent Installation Activity:"
if [[ -f "/var/log/dpkg.log" ]]; then
    echo "Recent package installations:"
    tail -10 /var/log/dpkg.log | grep -E "(install|configure)" | tail -5
    echo
fi

if [[ -f "/var/log/apt/history.log" ]]; then
    echo "Recent apt history:"
    tail -20 /var/log/apt/history.log | grep -E "(Install|Upgrade)" | tail -3
    echo
fi

# Firewall Status
log_info "Firewall Status:"
if command -v ufw >/dev/null 2>&1; then
    echo "  UFW Status: $(ufw status | head -1)"
else
    echo "  UFW: Not installed"
fi

if command -v iptables >/dev/null 2>&1; then
    rule_count=$(iptables -L | grep -c "Chain\|target")
    echo "  iptables rules: $rule_count"
else
    echo "  iptables: Not available"
fi
echo

# Potential Issues Detection
log_info "Potential Issues:"
issues_found=false

# Check for low disk space
available_space=$(df / | tail -1 | awk '{print $4}')
if [[ $available_space -lt 1048576 ]]; then  # Less than 1GB
    log_warning "Low disk space detected (less than 1GB available)"
    issues_found=true
fi

# Check for DNS issues
if ! timeout 5 nslookup github.com >/dev/null 2>&1; then
    log_warning "DNS resolution issues detected"
    issues_found=true
fi

# Check for network connectivity
if ! timeout 10 curl -s --head https://github.com >/dev/null 2>&1; then
    log_warning "Internet connectivity issues detected"
    issues_found=true
fi

# Check for package manager lock
if [[ -f "/var/lib/dpkg/lock-frontend" ]] && lsof /var/lib/dpkg/lock-frontend >/dev/null 2>&1; then
    log_warning "Package manager appears to be locked"
    issues_found=true
fi

if [[ "$issues_found" == false ]]; then
    log_success "No obvious issues detected"
fi

echo
log_info "Diagnostic completed. If you're still having issues:"
echo "1. Try running the standalone 3proxy installer"
echo "2. Check /var/log/syslog for detailed error messages"
echo "3. Ensure you have a stable internet connection"
echo "4. Try running the installer with 'bash -x' for verbose output"