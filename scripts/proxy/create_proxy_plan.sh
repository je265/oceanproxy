#!/bin/bash

# OceanProxy - Create Proxy Plan Script
# Creates 3proxy instance that redirects to ORIGINAL upstream provider

set -euo pipefail

# Validate input parameters
if [ $# -ne 8 ]; then
    echo "Usage: $0 <plan_id> <local_port> <username> <password> <auth_host> <auth_port> <plan_type_key> <region>"
    exit 1
fi

PLAN_ID="$1"
LOCAL_PORT="$2"
USERNAME="$3"
PASSWORD="$4"
AUTH_HOST="$5"        # e.g., pr-us.proxies.fo (ORIGINAL upstream)
AUTH_PORT="$6"        # e.g., 13337 (ORIGINAL upstream port)
PLAN_TYPE_KEY="$7"    # e.g., proxies_fo_usa_residential
REGION="$8"           # e.g., usa

# Configuration
PROXY_CONFIG_DIR="/etc/3proxy"
LOG_DIR="/var/log/oceanproxy"
PROXY_CONFIG="$PROXY_CONFIG_DIR/3proxy_${PLAN_ID}.cfg"

# Logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_DIR/proxy_creation.log"
}

log "Creating proxy plan: $PLAN_ID"
log "Flow: Customer -> us:$LOCAL_PORT -> $AUTH_HOST:$AUTH_PORT"
log "Plan Type: $PLAN_TYPE_KEY, Region: $REGION"

# Validate port range (now supporting up to 30000 for 2000-port ranges)
if [ "$LOCAL_PORT" -lt 10000 ] || [ "$LOCAL_PORT" -gt 30000 ]; then
    log "ERROR: Port $LOCAL_PORT is outside allowed range (10000-30000)"
    exit 1
fi

# Kill any existing process on the port
if lsof -ti:$LOCAL_PORT >/dev/null 2>&1; then
    log "Killing existing process on port $LOCAL_PORT"
    lsof -ti:$LOCAL_PORT | xargs kill -9 || true
    sleep 2
fi

# Create 3proxy configuration that forwards to ORIGINAL upstream
cat > "$PROXY_CONFIG" << EOF
# 3proxy configuration for plan $PLAN_ID
# Plan Type: $PLAN_TYPE_KEY
# Region: $REGION
# 
# IMPORTANT: This forwards to the ORIGINAL upstream provider
# Customer -> OceanProxy:$LOCAL_PORT -> $AUTH_HOST:$AUTH_PORT
# 
# Generated on $(date)

daemon
log $LOG_DIR/3proxy_${PLAN_ID}.log D
logformat "- +_L%t.%. %N.%p %E %U %C:%c %R:%r %O %I %h %T"
rotate 30

# Customer authentication (what customers use)
users $USERNAME:CL:$PASSWORD

# Allow access from anywhere for authenticated users
allow $USERNAME

# HTTP proxy that forwards to ORIGINAL upstream provider
# This is the key: we forward to $AUTH_HOST:$AUTH_PORT (the original provider)
# NOT to our own branded domain
proxy -p$LOCAL_PORT -a -e$AUTH_HOST:$AUTH_PORT

# The -e flag specifies the external proxy to forward to
# Customer traffic flow:
# 1. Customer connects to us with: $USERNAME:$PASSWORD@usa.oceanproxy.io:1337
# 2. nginx routes to: 127.0.0.1:$LOCAL_PORT
# 3. 3proxy forwards to: $AUTH_HOST:$AUTH_PORT with customer credentials
# 4. Response comes back through the same path
EOF

log "Created 3proxy configuration: $PROXY_CONFIG"
log "Upstream target: $AUTH_HOST:$AUTH_PORT (ORIGINAL provider)"

# Start 3proxy instance
log "Starting 3proxy instance for plan $PLAN_ID"
3proxy "$PROXY_CONFIG" &
PROXY_PID=$!

# Wait for startup
sleep 2

# Verify process is running
if ! kill -0 $PROXY_PID 2>/dev/null; then
    log "ERROR: Failed to start 3proxy instance"
    exit 1
fi

log "3proxy instance started with PID: $PROXY_PID"

# Test the proxy connection to original upstream
test_proxy() {
    local test_url="http://httpbin.org/ip"
    local proxy_url="http://$USERNAME:$PASSWORD@127.0.0.1:$LOCAL_PORT"
    
    log "Testing proxy connectivity to original upstream..."
    if timeout 15 curl -s --proxy "$proxy_url" "$test_url" >/dev/null; then
        log "Proxy test successful - traffic flows to $AUTH_HOST:$AUTH_PORT"
        return 0
    else
        log "WARNING: Proxy test failed - check upstream connectivity"
        return 1
    fi
}

# Test the proxy
test_proxy

log "SUCCESS: Proxy plan $PLAN_ID created and ready"
log "Architecture: Customer -> nginx:outbound_port -> 3proxy:$LOCAL_PORT -> $AUTH_HOST:$AUTH_PORT"
log "Customer sees: $REGION.oceanproxy.io (white-labeled)"
log "Traffic goes to: $AUTH_HOST:$AUTH_PORT (original provider)"

echo "SUCCESS: Proxy plan $PLAN_ID created on port $LOCAL_PORT forwarding to $AUTH_HOST:$AUTH_PORT"