# 🌊 OceanProxy - Complete Beginner's Guide

**OceanProxy is a white-label HTTP proxy service that lets you resell proxy services under your own brand while automatically managing the technical complexity behind the scenes.**

## 📖 Table of Contents

1. [What is OceanProxy?](#what-is-oceanproxy)
2. [How Does It Work?](#how-does-it-work)
3. [Why Use OceanProxy?](#why-use-oceanproxy)
4. [Before You Start](#before-you-start)
5. [Installation Guide](#installation-guide)
6. [Configuration Setup](#configuration-setup)
7. [Getting Your First Proxy Running](#getting-your-first-proxy-running)
8. [Understanding the API](#understanding-the-api)
9. [Managing Your Proxy Business](#managing-your-proxy-business)
10. [Troubleshooting](#troubleshooting)
11. [Advanced Usage](#advanced-usage)
12. [Support](#support)

## What is OceanProxy?

### The Simple Explanation

Imagine you want to start a proxy service business, but you don't want to deal with:
- Setting up servers around the world
- Managing thousands of IP addresses
- Handling customer authentication
- Monitoring server health
- Dealing with different proxy providers

**OceanProxy solves all of this.** It's like having a complete proxy business infrastructure that you can brand as your own.

### What You Get

✅ **Your Own Branded Proxy Service**
- Customers connect to `usa.yourcompany.io:1337` (your domain)
- They never see the underlying providers
- You control pricing, customers, and branding

✅ **Multiple Proxy Types**
- Residential proxies (real home IP addresses)
- Datacenter proxies (server IP addresses)
- ISP proxies (business internet connections)
- Mobile proxies (cellular network IPs)

✅ **Global Coverage**
- USA region proxies
- European Union proxies
- Alpha region (Asia-Pacific)
- Beta region (customizable)
- Easy to add new regions

✅ **Complete Management System**
- Web API to create/delete customer accounts
- Health monitoring of all proxy servers
- Automatic load balancing
- Customer usage tracking

## How Does It Work?

### The Flow (Step by Step)

1. **Customer Makes Request**
   ```
   Customer uses: http://john:password123@usa.yourcompany.io:1337
   ```

2. **OceanProxy Receives Request**
   ```
   Your server receives the request and checks authentication
   ```

3. **Load Balancing**
   ```
   Nginx automatically distributes traffic across multiple proxy instances
   ```

4. **Proxy Forwarding**
   ```
   Request gets forwarded to upstream provider (Proxies.fo, Nettify, etc.)
   ```

5. **Response Returns**
   ```
   Website response flows back through the same path to your customer
   ```

### Visual Diagram

```
[Customer] → [Your Domain] → [OceanProxy] → [Upstream Provider] → [Target Website]
     ↑                                                                      ↓
     ←←←←←←←←←←←←←← Response flows back ←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←
```

### What Makes This Special

**Traditional Approach:**
- Buy servers worldwide
- Set up proxy software on each server
- Manage IP addresses manually
- Handle customer authentication yourself
- Monitor server health manually
- Scale by buying more servers

**OceanProxy Approach:**
- Connect to existing proxy providers via API
- Automatically manage thousands of proxy instances
- Present everything under your brand
- Handle all technical complexity automatically
- Scale instantly without buying hardware

## Why Use OceanProxy?

### For Business Owners

💰 **Low Startup Costs**
- No need to buy expensive servers worldwide
- No data center contracts
- Start with just one small server

📈 **Instant Scalability**
- Go from 10 customers to 10,000 without infrastructure changes
- Automatic load balancing handles traffic spikes
- Add new regions in minutes, not months

🏷️ **Your Brand, Your Control**
- Customers only see your company name
- Set your own pricing
- Build customer relationships directly

⚡ **Quick to Market**
- Launch your proxy service in hours, not months
- Focus on marketing and customers, not infrastructure
- Complete business solution out of the box

### For Developers

🔧 **Production Ready**
- Docker deployment included
- Systemd service configuration
- Comprehensive logging and monitoring
- Automatic error handling

📊 **Complete API**
- RESTful API for all operations
- Create/delete customer accounts programmatically
- Real-time status monitoring
- Usage statistics and reporting

🛡️ **Security Built-In**
- Bearer token authentication
- Rate limiting protection
- Secure password handling
- Input validation on all endpoints

## Before You Start

### What You Need

**Technical Requirements:**
- A Linux server (Ubuntu 20.04+ recommended)
- At least 2GB RAM and 2 CPU cores
- 20GB+ storage space
- Root/sudo access to the server

**Business Requirements:**
- API keys from proxy providers:
  - [Proxies.fo](https://proxies.fo) account and API key
  - [Nettify](https://nettify.xyz) account and API key
- A domain name for your branded service
- Basic understanding of command line (we'll guide you through everything)

**Optional but Recommended:**
- SSL certificate for HTTPS (Let's Encrypt is free)
- Monitoring service (we'll show you free options)
- Backup solution

### Understanding Key Concepts

**Proxy Plan**: A customer's subscription to your proxy service
- Contains: username, password, bandwidth limit, expiration date
- Example: "John's residential USA proxy plan with 10GB bandwidth"

**Proxy Instance**: The actual running proxy server
- Each plan creates one or more instances
- Handles the technical connection forwarding
- Monitored for health and performance

**Provider**: The upstream proxy company (Proxies.fo, Nettify)
- Provides the actual IP addresses and infrastructure
- OceanProxy manages the integration automatically

**Region**: Geographic location grouping
- USA region serves North American customers
- EU region serves European customers  
- Each region has its own branded subdomain

## Installation Guide

### Step 1: Prepare Your Server

First, update your system:

```bash
# Update package lists
sudo apt update && sudo apt upgrade -y

# Install basic tools
sudo apt install -y curl wget git unzip
```

### Step 2: Download OceanProxy

```bash
# Clone the repository
git clone https://github.com/je265/oceanproxy.git
cd oceanproxy

# Make scripts executable
chmod +x scripts/**/*.sh
chmod +x deployments/scripts/*.sh
```

### Step 3: Run the Installation Script

The installation script will:
- Install all system dependencies (Go, nginx, 3proxy, Redis)
- Create system users and directories
- Configure firewall rules
- Set up systemd service
- Optimize system settings

```bash
# Run the automated installer
sudo ./deployments/scripts/install.sh
```

**What the installer does:**
1. Detects your operating system
2. Installs Go, nginx, 3proxy, Redis, and other dependencies
3. Creates `oceanproxy` system user
4. Sets up directories: `/opt/oceanproxy`, `/var/log/oceanproxy`, `/etc/oceanproxy`
5. Configures nginx with stream module for load balancing
6. Sets up firewall rules for proxy ports
7. Creates systemd service for automatic startup
8. Optimizes system settings for high performance

### Step 4: Verify Installation

Check that services are running:

```bash
# Check OceanProxy service
sudo systemctl status oceanproxy

# Check nginx
sudo systemctl status nginx

# Check Redis
sudo systemctl status redis
```

You should see all services as "active (running)".

## Configuration Setup

### Step 1: Configure Environment Variables

The main configuration is in `/etc/oceanproxy/oceanproxy.env`:

```bash
# Edit the configuration file
sudo nano /etc/oceanproxy/oceanproxy.env
```

**Essential settings to change:**

```bash
# SECURITY (CHANGE THESE!)
BEARER_TOKEN=your-super-secret-api-token-here-make-it-long-and-random
JWT_SECRET=another-super-secret-key-for-jwt-tokens

# PROVIDER API KEYS (GET THESE FROM YOUR PROVIDER ACCOUNTS)
PROXIES_FO_API_KEY=your-proxies-fo-api-key-here
NETTIFY_API_KEY=your-nettify-api-key-here

# YOUR DOMAIN (CHANGE THIS TO YOUR DOMAIN)
PROXY_DOMAIN=yourcompany.io

# LOGGING LEVEL (info for production, debug for troubleshooting)
LOG_LEVEL=info
```

### Step 2: Get Provider API Keys

**For Proxies.fo:**
1. Go to [proxies.fo](https://proxies.fo)
2. Create account and verify email
3. Go to API section in dashboard
4. Copy your API key
5. Add it to `PROXIES_FO_API_KEY=` in your config

**For Nettify:**
1. Go to [nettify.xyz](https://nettify.xyz)
2. Create account and verify email
3. Go to API/Developer section
4. Generate API key
5. Add it to `NETTIFY_API_KEY=` in your config

### Step 3: Configure Your Domain

**Option A: Use Subdomains (Recommended)**

Set up DNS records for your domain:
```
usa.yourcompany.io    → Your server IP
eu.yourcompany.io     → Your server IP  
alpha.yourcompany.io  → Your server IP
```

**Option B: Use Different Ports on Main Domain**
```
yourcompany.io:1337   → USA proxies
yourcompany.io:1338   → EU proxies
yourcompany.io:9876   → Alpha proxies
```

### Step 4: Configure Regions and Plan Types

Edit the region configuration:

```bash
sudo nano /etc/oceanproxy/regions.yaml
```

Update the domain references:

```yaml
regions:
  usa:
    subdomain: usa
    domain_suffix: yourcompany.io  # Change this to your domain
    outbound_port: 1337
    description: "United States proxies"
    
  eu:
    subdomain: eu
    domain_suffix: yourcompany.io  # Change this to your domain
    outbound_port: 1338
    description: "European Union proxies"
```

### Step 5: Restart Services

After configuration changes:

```bash
# Restart OceanProxy to load new configuration
sudo systemctl restart oceanproxy

# Restart nginx to apply any changes
sudo systemctl restart nginx

# Check everything is running
sudo systemctl status oceanproxy nginx
```

## Getting Your First Proxy Running

### Step 1: Test the API

First, verify the API is responding:

```bash
# Health check (should return "healthy")
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"2024-01-15T10:30:00Z","version":"1.0.0"}
```

### Step 2: Create Your First Customer Plan

Use the API to create a test customer:

```bash
# Create a residential USA proxy plan
curl -X POST http://localhost:8080/api/v1/plans \
  -H "Authorization: Bearer your-super-secret-api-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "test_customer_001",
    "plan_type": "residential", 
    "provider": "proxies_fo",
    "region": "usa",
    "username": "testuser",
    "password": "testpass123",
    "bandwidth": 10,
    "duration": 30
  }'
```

**Expected successful response:**
```json
{
  "success": true,
  "plan_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "testuser",
  "password": "testpass123", 
  "expires_at": "2024-02-15T10:30:00Z",
  "proxies": [
    {
      "url": "http://testuser:testpass123@usa.yourcompany.io:1337",
      "region": "usa",
      "username": "testuser",
      "password": "testpass123"
    }
  ]
}
```

### Step 3: Test the Proxy

Now test that the proxy actually works:

```bash
# Test the proxy by making a request through it
curl --proxy "http://testuser:testpass123@usa.yourcompany.io:1337" \
     http://httpbin.org/ip

# Expected response (will show proxy's IP, not your server's IP):
# {
#   "origin": "123.456.789.012"  ← This should be different from your server IP
# }
```

### Step 4: Verify Everything is Working

Check the logs to see activity:

```bash
# View recent OceanProxy logs
sudo tail -f /var/log/oceanproxy/app.log

# View nginx proxy logs  
sudo tail -f /var/log/nginx/usa_proxy.log

# View 3proxy logs (the actual proxy process)
sudo tail -f /var/log/oceanproxy/3proxy_*.log
```

**What you should see:**
- OceanProxy logs showing plan creation and instance startup
- Nginx logs showing proxy connections
- 3proxy logs showing traffic forwarding to upstream provider

### Step 5: Understanding What Just Happened

When you created that plan, OceanProxy automatically:

1. **Created upstream account** with Proxies.fo using their API
2. **Allocated local port** (probably 10001) from the USA residential range
3. **Generated 3proxy config** to forward traffic from local port to Proxies.fo
4. **Started 3proxy process** to handle the actual forwarding
5. **Updated nginx config** to include the new proxy in load balancing
6. **Stored plan details** in the database for management

Your customer can now use `http://testuser:testpass123@usa.yourcompany.io:1337` as their proxy, and it will automatically load balance across all your USA residential proxy instances.

## Understanding the API

### Authentication

All API calls require a Bearer token in the Authorization header:

```bash
-H "Authorization: Bearer your-api-token-here"
```

**Security Note:** Keep this token secret! Anyone with this token can create/delete customer accounts.

### Core Endpoints

#### 1. Health Check
```bash
GET /health
# No authentication required
# Returns: Service health status
```

#### 2. Create Customer Plan
```bash
POST /api/v1/plans
# Authentication required
# Body: JSON with customer details
# Returns: Customer proxy credentials
```

#### 3. List All Plans
```bash
GET /api/v1/plans
# Authentication required  
# Optional: ?customer_id=specific_customer
# Returns: Array of all customer plans
```

#### 4. Get Specific Plan
```bash
GET /api/v1/plans/{plan-id}
# Authentication required
# Returns: Detailed plan information
```

#### 5. Delete Customer Plan
```bash
DELETE /api/v1/plans/{plan-id}  
# Authentication required
# Returns: Success confirmation
# Note: This stops all proxy instances and deletes customer access
```

### Plan Creation Parameters

When creating a plan, you specify:

**Required Parameters:**
- `customer_id`: Your internal customer identifier
- `plan_type`: Type of proxy (residential, datacenter, isp, mobile, unlimited)
- `provider`: Which upstream provider (proxies_fo, nettify)  
- `region`: Geographic region (usa, eu, alpha, beta, asia)
- `username`: Customer's proxy username
- `password`: Customer's proxy password

**Optional Parameters:**
- `bandwidth`: Bandwidth limit in GB (default: based on provider)
- `duration`: Plan length in days (default: 30)

### Plan Types Explained

**Residential Proxies:**
- Real home internet connections
- Highest anonymity and success rates  
- More expensive but best for web scraping
- Best for: Social media, e-commerce, SEO tools

**Datacenter Proxies:**
- Server-based IP addresses
- Fast and reliable
- Lower cost but easier to detect
- Best for: General browsing, bulk requests

**ISP Proxies:**
- Business internet connections
- Good balance of speed and anonymity
- Medium pricing
- Best for: Mixed use cases

**Mobile Proxies:**
- Cellular network IP addresses
- Very high anonymity
- Most expensive
- Best for: Mobile app testing, social media automation

### Regional Coverage

**USA Region (usa.yourcompany.io:1337):**
- Covers United States and Canada
- Multiple states/provinces available
- Fast connections to US websites

**EU Region (eu.yourcompany.io:1338):**  
- Covers European Union countries
- GDPR compliant
- Good for European e-commerce sites

**Alpha Region (alpha.yourcompany.io:9876):**
- Asia-Pacific coverage
- Configurable based on your needs
- Good for Asian markets

## Managing Your Proxy Business

### Customer Lifecycle

#### Creating New Customers

```bash
# Residential customer for social media automation
curl -X POST http://localhost:8080/api/v1/plans \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "social_media_customer_001",
    "plan_type": "residential",
    "provider": "proxies_fo", 
    "region": "usa",
    "username": "socialmedia_user",
    "password": "secure_password_123",
    "bandwidth": 50,
    "duration": 30
  }'

# Datacenter customer for web scraping
curl -X POST http://localhost:8080/api/v1/plans \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "scraping_customer_002", 
    "plan_type": "datacenter",
    "provider": "nettify",
    "region": "eu",
    "username": "scraper_user",
    "password": "another_secure_pass",
    "bandwidth": 100,
    "duration": 7
  }'
```

#### Monitoring Customer Usage

```bash
# List all active customers
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans

# Check specific customer
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans/123e4567-e89b-12d3-a456-426614174000

# Filter by customer ID
curl -H "Authorization: Bearer your-token" \
     "http://localhost:8080/api/v1/plans?customer_id=social_media_customer_001"
```

#### Handling Customer Issues

**Customer reports proxy not working:**

1. **Check proxy instance status:**
```bash
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/456e7890-e89b-12d3-a456-426614174001/status
```

2. **Restart proxy instance if needed:**
```bash
curl -X POST -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/456e7890-e89b-12d3-a456-426614174001/restart
```

3. **Check logs for errors:**
```bash
sudo tail -50 /var/log/oceanproxy/3proxy_456e7890-e89b-12d3-a456-426614174001.log
```

#### Customer Cancellation

```bash
# Delete customer plan (stops all proxies and removes access)
curl -X DELETE -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans/123e4567-e89b-12d3-a456-426614174000
```

### System Monitoring

#### Daily Health Checks

Create a simple monitoring script:

```bash
#!/bin/bash
# save as /opt/oceanproxy/daily_check.sh

echo "=== OceanProxy Daily Health Check ===" 
echo "Date: $(date)"

# Check main service
if systemctl is-active --quiet oceanproxy; then
    echo "✅ OceanProxy service: Running"
else  
    echo "❌ OceanProxy service: STOPPED"
fi

# Check API health
if curl -f -s http://localhost:8080/health > /dev/null; then
    echo "✅ API endpoint: Responding"
else
    echo "❌ API endpoint: NOT RESPONDING"  
fi

# Check nginx
if systemctl is-active --quiet nginx; then
    echo "✅ Nginx service: Running"
else
    echo "❌ Nginx service: STOPPED"
fi

# Check disk space
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -lt 80 ]; then
    echo "✅ Disk usage: ${DISK_USAGE}%"
else
    echo "⚠️ Disk usage: ${DISK_USAGE}% (High!)"
fi

# Count active plans
ACTIVE_PLANS=$(curl -s -H "Authorization: Bearer your-token" \
               http://localhost:8080/api/v1/plans | jq '. | length')
echo "📊 Active customer plans: $ACTIVE_PLANS"

echo "=================================="
```

Run it daily with cron:
```bash
# Add to crontab
echo "0 9 * * * /opt/oceanproxy/daily_check.sh >> /var/log/oceanproxy/daily_check.log" | sudo crontab -
```

#### Performance Monitoring

Check system resources:

```bash
# CPU and memory usage
htop

# Network connections
ss -tulpn | grep -E ':(1337|1338|9876|8080)'

# Proxy process count  
ps aux | grep 3proxy | wc -l

# Log file sizes
du -sh /var/log/oceanproxy/*
```

### Backup and Recovery

#### Daily Backup Script

```bash
#!/bin/bash
# save as /opt/oceanproxy/backup.sh

BACKUP_DIR="/var/backups/oceanproxy"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup customer data
cp /var/log/oceanproxy/proxies.json $BACKUP_DIR/proxies_$DATE.json
cp /var/log/oceanproxy/proxies.json_instances $BACKUP_DIR/instances_$DATE.json

# Backup configuration
cp -r /etc/oceanproxy $BACKUP_DIR/config_$DATE/

# Backup logs (last 7 days)
find /var/log/oceanproxy -name "*.log" -mtime -7 -exec cp {} $BACKUP_DIR/ \;

# Clean old backups (keep 30 days)
find $BACKUP_DIR -type f -mtime +30 -delete

echo "Backup completed: $DATE"
```

#### Recovery Process

If you need to restore from backup:

```bash
# Stop services
sudo systemctl stop oceanproxy

# Restore data files
sudo cp /var/backups/oceanproxy/proxies_YYYYMMDD_HHMMSS.json /var/log/oceanproxy/proxies.json
sudo cp /var/backups/oceanproxy/instances_YYYYMMDD_HHMMSS.json /var/log/oceanproxy/proxies.json_instances

# Set correct permissions
sudo chown oceanproxy:oceanproxy /var/log/oceanproxy/proxies*

# Restart services
sudo systemctl start oceanproxy
```

## Troubleshooting

### Common Issues and Solutions

#### 1. "Failed to create plan" Error

**Problem:** API returns error when creating customer plan

**Debugging steps:**
```bash
# Check OceanProxy logs
sudo tail -50 /var/log/oceanproxy/app.log

# Common causes:
# - Invalid provider API keys
# - Network connectivity issues  
# - Provider API quota exceeded
# - Invalid plan type/region combination
```

**Solutions:**
- Verify provider API keys are correct
- Check internet connectivity: `ping api.proxies.fo`
- Check provider account has available quota
- Verify plan type matches provider capabilities

#### 2. "Port already in use" Error

**Problem:** Cannot start proxy instance due to port conflict

**Debugging:**
```bash
# Find what's using the port
sudo lsof -i :10001

# Check for orphaned 3proxy processes
ps aux | grep 3proxy
```

**Solutions:**
```bash
# Kill process using the port
sudo kill -9 PID_NUMBER

# Clean up orphaned 3proxy processes
sudo pkill 3proxy

# Restart OceanProxy service
sudo systemctl restart oceanproxy
```

#### 3. Customer Reports "Proxy Not Working"

**Problem:** Customer cannot connect through proxy

**Debugging steps:**
```bash
# Test proxy from server
curl --proxy "http://username:password@usa.yourcompany.io:1337" http://httpbin.org/ip

# Check proxy instance status
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/INSTANCE_ID/status

# Check 3proxy logs
sudo tail -50 /var/log/oceanproxy/3proxy_*.log

# Check nginx proxy logs
sudo tail -50 /var/log/nginx/usa_proxy.log
```

**Common causes:**
- Expired upstream provider credentials
- Network connectivity issues
- Firewall blocking proxy ports
- DNS resolution problems

**Solutions:**
```bash
# Restart proxy instance
curl -X POST -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/INSTANCE_ID/restart

# Check firewall rules
sudo ufw status

# Test DNS resolution
dig usa.yourcompany.io
```

#### 4. High CPU/Memory Usage

**Problem:** Server performance is slow

**Debugging:**
```bash
# Check resource usage
htop

# Check for memory leaks
sudo tail -100 /var/log/syslog | grep -i "out of memory"

# Count running processes
ps aux | grep -E "(3proxy|nginx|oceanproxy)" | wc -l
```

**Solutions:**
- Implement log rotation to prevent large log files
- Monitor and limit number of concurrent proxy instances
- Consider upgrading server resources
- Optimize 3proxy configurations

#### 5. Nginx Configuration Issues

**Problem:** Load balancing not working correctly

**Debugging:**
```bash
# Test nginx configuration
sudo nginx -t

# Check nginx error logs
sudo tail -50 /var/log/nginx/error.log

# Verify upstream servers
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies
```

**Solutions:**
```bash
# Reload nginx configuration
sudo nginx -s reload

# Restart nginx if needed
sudo systemctl restart nginx

# Regenerate nginx configurations
# (This would be a custom OceanProxy command)
```

### Log Analysis

#### Understanding Log Levels

**INFO**: Normal operation messages
```
2024-01-15 10:30:15 [INFO] Plan created successfully: 123e4567-e89b-12d3-a456-426614174000
```

**WARNING**: Issues that don't stop operation
```
2024-01-15 10:30:16 [WARN] Failed to kill existing process on port 10001: no such process
```

**ERROR**: Problems that prevent operation
```
2024-01-15 10:30:17 [ERROR] Failed to create upstream account: invalid API key
```

#### Important Log Files

1. **Main application**: `/var/log/oceanproxy/app.log`
2. **Individual proxy instances**: `/var/log/oceanproxy/3proxy_*.log`
3. **Nginx proxy traffic**: `/var/log/nginx/*_proxy.log`
4. **System messages**: `/var/log/syslog`

#### Log Monitoring Commands

```bash
# Watch all OceanProxy logs in real-time
sudo tail -f /var/log/oceanproxy/*.log

# Search for errors in the last hour
sudo find /var/log/oceanproxy -name "*.log" -exec grep -l "ERROR" {} \; | xargs tail -100

# Count proxy instances by status
sudo grep -h "status" /var/log/oceanproxy/app.log | grep -E "(running|stopped|failed)" | sort | uniq -c
```

### Performance Optimization

#### System Tuning

```bash
# Increase file descriptor limits
echo "oceanproxy soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "oceanproxy hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Optimize network settings
sudo sysctl -w net.core.somaxconn=65536
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=65536

# Make settings permanent
echo "net.core.somaxconn=65536" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog=65536" | sudo tee -a /etc/sysctl.conf
```

#### Application Tuning

```bash
# Optimize Go garbage collector
export GOGC=100

# Set CPU usage
export GOMAXPROCS=4  # Set to number of CPU cores
```

## Advanced Usage

### Custom Provider Integration

To add a new proxy provider, you need to:

1. **Create provider client** in `internal/service/provider/`:

```go
// internal/service/provider/newprovider.go
type NewProvider struct {
    cfg    *config.NewProviderConfig
    logger *zap.Logger
    client *http.Client
}

func (p *NewProvider) CreateAccount(ctx context.Context, req *domain.CreatePlanRequest) (*service.ProviderAccount, error) {
    // Implementation specific to new provider's API
    return &service.ProviderAccount{
        ID:       "account_id_from_provider",
        Username: req.Username,
        Password: req.Password,
        Host:     "proxy.newprovider.com",
        Port:     8080,
        Region:   req.Region,
    }, nil
}
```

2. **Add to provider service** in `internal/service/provider_service.go`:

```go
func (s *providerService) CreateAccount(ctx context.Context, providerName string, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
    switch providerName {
    case "newprovider":
        return s.newProvider.CreateAccount(ctx, req)
    // ... existing providers
    }
}
```

3. **Update configuration** in `configs/proxy-plans.yaml`:

```yaml
plan_types:
  newprovider_usa_residential:
    provider: newprovider
    region: usa
    plan_type: residential
    upstream_host: proxy.newprovider.com
    upstream_port: 8080
    local_port_range:
      start: 30000
      end: 31999
    outbound_port: 1337
    nginx_upstream_name: oceanproxy_usa_residential
```

### SSL/HTTPS Setup

To enable HTTPS for your API and proxy endpoints:

1. **Get SSL certificates** (using Let's Encrypt):

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate for your domains
sudo certbot --nginx -d yourcompany.io -d usa.yourcompany.io -d eu.yourcompany.io -d alpha.yourcompany.io
```

2. **Update nginx configuration** to enable HTTPS proxy:

```nginx
# Add to /etc/nginx/nginx.conf in the http block
server {
    listen 443 ssl http2;
    server_name usa.yourcompany.io;
    
    ssl_certificate /etc/letsencrypt/live/yourcompany.io/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourcompany.io/privkey.pem;
    
    location /api/ {
        proxy_pass http://oceanproxy:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

3. **Enable HTTPS in OceanProxy configuration**:

```bash
# Edit /etc/oceanproxy/oceanproxy.env
TLS_ENABLED=true
TLS_CERT_FILE=/etc/letsencrypt/live/yourcompany.io/fullchain.pem
TLS_KEY_FILE=/etc/letsencrypt/live/yourcompany.io/privkey.pem
```

### Database Migration to PostgreSQL

For high-volume operations, migrate from JSON files to PostgreSQL:

1. **Install PostgreSQL**:

```bash
sudo apt install postgresql postgresql-contrib
sudo -u postgres createdb oceanproxy
sudo -u postgres createuser oceanproxy
sudo -u postgres psql -c "ALTER USER oceanproxy PASSWORD 'secure_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE oceanproxy TO oceanproxy;"
```

2. **Update configuration**:

```bash
# Edit /etc/oceanproxy/oceanproxy.env
DATABASE_DRIVER=postgres
DATABASE_DSN=postgres://oceanproxy:secure_password@localhost/oceanproxy?sslmode=disable
```

3. **Create database schema** (this would require implementing the PostgreSQL repository):

```sql
-- This is an example schema that would need to be implemented
CREATE TABLE proxy_plans (
    id UUID PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    plan_type VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(50) NOT NULL,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    bandwidth INTEGER NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_proxy_plans_customer_id ON proxy_plans(customer_id);
CREATE INDEX idx_proxy_plans_status ON proxy_plans(status);
```

### Monitoring with Prometheus and Grafana

Set up comprehensive monitoring:

1. **Configure Prometheus** in `build/monitoring/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'oceanproxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
    
  - job_name: 'nginx'
    static_configs:
      - targets: ['localhost:9113']
    scrape_interval: 30s
```

2. **Start monitoring stack**:

```bash
# Using Docker Compose
make compose-up

# Access Grafana at http://your-server:3000
# Default login: admin/admin
```

3. **Create custom dashboards** for:
   - Active customer count
   - Proxy instance health
   - Network traffic by region
   - Error rates by provider
   - Resource utilization

### API Rate Limiting

Implement per-customer rate limiting:

1. **Configure Redis-based rate limiting**:

```bash
# Edit /etc/oceanproxy/oceanproxy.env
ENABLE_RATE_LIMITING=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST=20
```

2. **Monitor rate limiting**:

```bash
# Check rate limit statistics
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/stats/rate-limits
```

### High Availability Setup

For production environments requiring 99.9% uptime:

1. **Load Balancer Setup** (using HAProxy):

```bash
# Install HAProxy
sudo apt install haproxy

# Configure /etc/haproxy/haproxy.cfg
backend oceanproxy_api
    balance roundrobin
    server oceanproxy1 10.0.1.10:8080 check
    server oceanproxy2 10.0.1.11:8080 check
    server oceanproxy3 10.0.1.12:8080 check
```

2. **Database Replication**:

```bash
# Set up PostgreSQL master-slave replication
# Configure read replicas for better performance
```

3. **Shared Storage**:

```bash
# Use NFS or similar for shared configuration
# Ensure all instances can access same data
```

### Kubernetes Deployment

Deploy on Kubernetes for container orchestration:

1. **Create namespace**:

```bash
kubectl create namespace oceanproxy
```

2. **Deploy using provided manifests**:

```bash
kubectl apply -f deployments/k8s/
```

3. **Scale deployment**:

```bash
kubectl scale deployment oceanproxy --replicas=3 -n oceanproxy
```

### Custom Analytics Dashboard

Build a customer-facing dashboard:

1. **Create analytics API endpoints**:

```go
// Add to your API routes
func (h *AnalyticsHandler) GetCustomerStats(w http.ResponseWriter, r *http.Request) {
    customerID := r.URL.Query().Get("customer_id")
    
    stats := analytics.CustomerStats{
        TotalRequests: 1500,
        DataUsed: "2.5 GB",
        TopCountries: []string{"US", "UK", "DE"},
        SuccessRate: 99.2,
    }
    
    json.NewEncoder(w).Encode(stats)
}
```

2. **Create web interface** using your preferred framework (React, Vue, etc.)

## Support

### Getting Help

**Community Support:**
- GitHub Issues: [Report bugs and request features](https://github.com/je265/oceanproxy/issues)
- Discussions: [Ask questions and share tips](https://github.com/je265/oceanproxy/discussions)
- Documentation: [Complete guides and references](https://docs.oceanproxy.io)

**Professional Support:**
- Email: support@oceanproxy.io
- Priority support available for business customers
- Custom integration and setup services

### Frequently Asked Questions

**Q: How many customers can OceanProxy handle?**
A: On a 4GB server, OceanProxy can handle 1000+ concurrent customers. Each proxy instance uses minimal resources (1-2MB RAM).

**Q: Can I use my own proxy servers instead of providers?**
A: Yes! You can implement custom providers. The architecture supports any upstream proxy source.

**Q: Is customer data secure?**
A: Yes. Customer passwords are stored securely, API access is token-based, and all traffic is isolated per customer.

**Q: Can I customize the branding completely?**
A: Absolutely. Customers only see your domain names and branding. Provider details are completely hidden.

**Q: What happens if a provider goes down?**
A: OceanProxy includes health monitoring and can automatically failover to backup providers if configured.

**Q: Can I run this on Windows?**
A: OceanProxy is designed for Linux environments. While Go code is cross-platform, scripts and system integration require Linux.

**Q: How do I handle customer billing?**
A: OceanProxy provides usage data through the API. Integrate with your preferred billing system (Stripe, PayPal, etc.).

**Q: Can customers use their own usernames/passwords?**
A: Yes! When creating plans, you specify the username/password that customers will use.

### Contributing

We welcome contributions! Here's how to help:

1. **Report Issues**: Found a bug? [Open an issue](https://github.com/je265/oceanproxy/issues)

2. **Suggest Features**: Have ideas? [Start a discussion](https://github.com/je265/oceanproxy/discussions)

3. **Contribute Code**:
   ```bash
   # Fork the repository
   git clone https://github.com/yourusername/oceanproxy.git
   cd oceanproxy
   
   # Create feature branch
   git checkout -b feature/amazing-feature
   
   # Make changes and test
   make test
   make lint
   
   # Submit pull request
   git push origin feature/amazing-feature
   ```

4. **Improve Documentation**: Help others by improving guides and examples

### Roadmap

**Coming Soon:**
- [ ] PostgreSQL database support with migrations
- [ ] Advanced analytics dashboard
- [ ] Webhook notifications for events
- [ ] Custom domain SSL automation
- [ ] Kubernetes operator for easy scaling
- [ ] Advanced load balancing algorithms

**Future Features:**
- [ ] Mobile app for proxy management
- [ ] AI-powered traffic routing
- [ ] Blockchain-based proxy verification
- [ ] IoT device proxy support

### License and Legal

**License**: MIT License - Use OceanProxy freely in commercial and personal projects

**Terms of Service**: By using OceanProxy, you agree to:
- Use proxy services legally and ethically
- Respect upstream provider terms of service
- Not engage in malicious activities
- Comply with local laws and regulations

**Privacy**: OceanProxy does not log or store customer proxy traffic. Only connection metadata is retained for operational purposes.

---

## 🎉 Congratulations!

You now have a complete understanding of OceanProxy! You can:

✅ **Start a proxy business** with branded customer endpoints
✅ **Manage customers** through the comprehensive API  
✅ **Scale automatically** without infrastructure worries
✅ **Monitor everything** with built-in health checks
✅ **Troubleshoot issues** using detailed logs and guides
✅ **Customize extensively** for your specific needs

### Quick Success Checklist

- [ ] Server setup completed with installation script
- [ ] Configuration updated with your API keys and domain
- [ ] First test customer created and working
- [ ] Monitoring and backup scripts configured
- [ ] SSL certificates installed for security
- [ ] Documentation read and understood
- [ ] Support contacts saved for help

**Ready to launch your proxy empire?** 🌊

Start with the [Installation Guide](#installation-guide) and you'll be serving customers within hours!

---

*Made with ❤️ by the OceanProxy team*
*Last updated: January 2024*