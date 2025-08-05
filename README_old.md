# OceanProxy

🌊 **OceanProxy** is a high-performance, white-label HTTP proxy service that provides seamless integration with multiple upstream proxy providers while presenting a unified, branded interface to customers.

## Features

- **Multi-Provider Support**: Integrates with Proxies.fo, Nettify, and extensible to other providers
- **White-Label Architecture**: Present your own branded proxy endpoints to customers
- **Intelligent Load Balancing**: Nginx-based load balancing across multiple proxy instances
- **Regional Proxy Support**: USA, EU, Alpha, Beta, and Asia regions
- **Multiple Plan Types**: Residential, Datacenter, ISP, Mobile, and Unlimited plans
- **Port Pool Management**: Intelligent port allocation with 2000-port ranges per plan type
- **RESTful API**: Full CRUD operations for plan and proxy management
- **Health Monitoring**: Built-in health checks and monitoring endpoints
- **Scalable Architecture**: Designed to handle thousands of concurrent proxy plans
- **Docker Support**: Containerized deployment with Docker Compose

## Architecture

```
Customer Request → nginx (branded domain) → 3proxy instances → Upstream Providers
```

### Flow Example
1. Customer connects to: `http://user:pass@usa.oceanproxy.io:1337`
2. Nginx routes to local 3proxy instance on port 10001
3. 3proxy forwards to original upstream: `pr-us.proxies.fo:13337`
4. Response flows back through the same path

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- nginx with stream module
- 3proxy
- Linux/Unix environment

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/je265/oceanproxy.git
cd oceanproxy
```

2. **Setup environment**
```bash
# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

3. **Install system dependencies**
```bash
sudo ./scripts/system/install_deps.sh
```

4. **Build and run**
```bash
# Development
make setup-dev
make dev

# Production
make build
make run

# Docker
make compose-up
```

## Configuration

### Environment Variables

Key environment variables (see `.env.example` for complete list):

```bash
# Authentication
BEARER_TOKEN=your-secure-bearer-token-here
JWT_SECRET=your-jwt-secret-key-here

# Provider API Keys
PROXIES_FO_API_KEY=your-proxies-fo-api-key
NETTIFY_API_KEY=your-nettify-api-key

# Proxy Configuration
PROXY_DOMAIN=oceanproxy.io
PROXY_START_PORT=10000
PROXY_END_PORT=30000
```

### Plan Type Configuration

Plan types are configured in `configs/proxy-plans.yaml`:

```yaml
plan_types:
  proxies_fo_usa_residential:
    provider: proxies_fo
    region: usa
    plan_type: residential
    upstream_host: pr-us.proxies.fo
    upstream_port: 13337
    local_port_range:
      start: 10000
      end: 11999
```

### Region Configuration

Regions are configured in `configs/regions.yaml`:

```yaml
regions:
  usa:
    subdomain: usa
    domain_suffix: oceanproxy.io
    outbound_port: 1337
    plan_types:
      - proxies_fo_usa_residential
      - proxies_fo_usa_datacenter
```

## API Usage

### Authentication

All API requests require a Bearer token:

```bash
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans
```

### Create a Plan

```bash
curl -X POST http://localhost:8080/api/v1/plans \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "customer_123",
    "plan_type": "residential",
    "provider": "proxies_fo",
    "region": "usa",
    "username": "testuser",
    "password": "testpass",
    "bandwidth": 10,
    "duration": 30
  }'
```

### Response

```json
{
  "success": true,
  "plan_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "testuser", 
  "password": "testpass",
  "expires_at": "2024-02-15T10:30:00Z",
  "proxies": [
    {
      "url": "http://testuser:testpass@usa.oceanproxy.io:1337",
      "region": "usa",
      "username": "testuser",
      "password": "testpass"
    }
  ]
}
```

### Get Plans

```bash
# Get all plans
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans

# Get plans for specific customer
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/plans?customer_id=customer_123
```

### Proxy Management

```bash
# Get proxy instances
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies

# Start proxy instance
curl -X POST -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/{id}/start

# Get proxy status
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/api/v1/proxies/{id}/status
```

## Port Management

OceanProxy uses intelligent port pool management:

- **USA Residential**: Ports 10000-11999 (2000 ports)
- **USA Datacenter**: Ports 12000-13999 (2000 ports)  
- **USA ISP**: Ports 14000-15999 (2000 ports)
- **EU Residential**: Ports 16000-17999 (2000 ports)
- **EU Datacenter**: Ports 18000-19999 (2000 ports)
- **EU ISP**: Ports 20000-21999 (2000 ports)
- **Alpha Plans**: Ports 22000-29999 (8000 ports)

## Monitoring

### Health Checks

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed readiness check
curl http://localhost:8080/ready
```

### Metrics

Prometheus metrics are available at `/metrics` (when enabled).

### Logging

Structured JSON logging with configurable levels:

- Application logs: `/var/log/oceanproxy/app.log`
- 3proxy logs: `/var/log/oceanproxy/3proxy_{instance_id}.log`
- Nginx logs: `/var/log/nginx/`

## Deployment

### Docker Compose

```bash
# Start all services
docker-compose -f build/docker-compose.yml up -d

# View logs
docker-compose -f build/docker-compose.yml logs -f

# Stop services
docker-compose -f build/docker-compose.yml down
```

### Systemd Service

```bash
# Install system-wide
make install

# Start/stop service
sudo systemctl start oceanproxy
sudo systemctl enable oceanproxy
sudo systemctl status oceanproxy
```

### Kubernetes

Kubernetes manifests are available in the `deployments/k8s/` directory:

```bash
kubectl apply -f deployments/k8s/
```

## Development

### Setup Development Environment

```bash
# Install dev tools and dependencies
make setup-dev

# Run with hot reload
make watch

# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Format code
make fmt
```

### Project Structure

```
oceanproxy/
├── cmd/                    # Application entrypoints
│   ├── server/            # Main server application
│   └── cli/               # CLI tools
├── internal/              # Private application code
│   ├── app/               # Application setup and routing
│   ├── config/            # Configuration management
│   ├── domain/            # Business logic and entities
│   ├── handlers/          # HTTP handlers
│   ├── repository/        # Data persistence layer
│   ├── service/           # Business logic services
│   └── pkg/               # Internal packages
├── configs/               # Configuration files
├── scripts/               # Utility scripts
│   ├── proxy/             # Proxy management scripts
│   └── system/            # System setup scripts
├── build/                 # Build and deployment files
├── deployments/           # Deployment configurations
├── docs/                  # Documentation
└── test/                  # Test files
```

### Adding New Providers

1. **Create provider implementation**:
```go
// internal/service/provider/newprovider.go
type NewProvider struct {
    cfg    *config.NewProviderConfig
    logger *zap.Logger
    client *http.Client
}

func (p *NewProvider) CreateAccount(ctx context.Context, req *domain.CreatePlanRequest) (*service.ProviderAccount, error) {
    // Implementation
}
```

2. **Add to provider service**:
```go
// internal/service/provider_service.go
func (s *providerService) CreateAccount(ctx context.Context, providerName string, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
    switch providerName {
    case "newprovider":
        return s.newProvider.CreateAccount(ctx, req)
    // ...
    }
}
```

3. **Update configuration**:
```yaml
# configs/proxy-plans.yaml
plan_types:
  newprovider_region_plantype:
    provider: newprovider
    region: region
    plan_type: plantype
    upstream_host: upstream.provider.com
    upstream_port: 8080
    local_port_range:
      start: 30000
      end: 31999
```

### Adding New Regions

1. **Update regions configuration**:
```yaml
# configs/regions.yaml
regions:
  newregion:
    subdomain: newregion
    domain_suffix: oceanproxy.io
    outbound_port: 7777
    plan_types:
      - provider_newregion_plantype
```

2. **Update nginx configuration** to handle the new outbound port.

## API Documentation

Complete API documentation is available at `/docs` when the server is running, or view the OpenAPI specification at `api/openapi.yaml`.

### Key Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check  
- `POST /api/v1/plans` - Create proxy plan
- `GET /api/v1/plans` - List plans
- `GET /api/v1/plans/{id}` - Get specific plan
- `DELETE /api/v1/plans/{id}` - Delete plan
- `GET /api/v1/proxies` - List proxy instances
- `POST /api/v1/proxies/{id}/start` - Start proxy instance
- `POST /api/v1/proxies/{id}/stop` - Stop proxy instance
- `GET /api/v1/proxies/{id}/status` - Get proxy status

### Legacy Endpoints

For backward compatibility:
- `POST /plan` - Create Proxies.fo plan (legacy)
- `POST /nettify/plan` - Create Nettify plan (legacy)

## Security

### Authentication

- Bearer token authentication for all API endpoints
- Configurable JWT support for future enhancements
- Rate limiting on API endpoints

### Network Security

- All proxy traffic is isolated per customer
- No cross-customer traffic contamination
- Upstream provider credentials are never exposed to customers
- SSL/TLS support for API endpoints

### Best Practices

1. **Use strong bearer tokens**: Generate cryptographically secure tokens
2. **Rotate credentials regularly**: Update provider API keys periodically
3. **Monitor access logs**: Track API usage and detect anomalies
4. **Firewall configuration**: Limit access to management ports
5. **Regular updates**: Keep dependencies and base images updated

## Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# Find process using port
lsof -i :PORT_NUMBER

# Kill process if needed
kill -9 PID
```

#### 3proxy Not Starting
```bash
# Check configuration
3proxy -t /etc/3proxy/3proxy_INSTANCE_ID.cfg

# Check logs
tail -f /var/log/oceanproxy/3proxy_INSTANCE_ID.log
```

#### Nginx Configuration Issues
```bash
# Test configuration
nginx -t

# Reload configuration
systemctl reload nginx
```

#### Provider API Issues
```bash
# Test provider connectivity
curl -X POST https://api.provider.com/test \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Debugging

#### Enable Debug Logging
```bash
export LOG_LEVEL=debug
```

#### View Application Logs
```bash
# Follow application logs
tail -f /var/log/oceanproxy/app.log

# View with docker-compose
docker-compose logs -f oceanproxy
```

#### Check System Status
```bash
# Check service status
make status

# Check port usage
netstat -tlnp | grep oceanproxy

# Check process list
ps aux | grep oceanproxy
```

## Performance Tuning

### System Configuration

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Optimize network stack
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65536" >> /etc/sysctl.conf
sysctl -p
```

### Application Tuning

```bash
# Set Go runtime parameters
export GOMAXPROCS=4
export GOGC=100

# Optimize for high concurrency
export GODEBUG=madvdontneed=1
```

### Nginx Optimization

```nginx
# Increase worker connections
worker_connections 4096;

# Optimize buffer sizes
proxy_buffering on;
proxy_buffer_size 128k;
proxy_buffers 4 256k;
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make test`)
6. Format code (`make fmt`)
7. Lint code (`make lint`)
8. Commit your changes (`git commit -am 'Add amazing feature'`)
9. Push to the branch (`git push origin feature/amazing-feature`)
10. Open a Pull Request

### Code Style

- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Write tests for new functionality
- Keep functions small and focused

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: Check the `docs/` directory
- **Issues**: Open an issue on GitHub
- **Email**: support@oceanproxy.io
- **Discord**: Join our community server

## Roadmap

- [ ] PostgreSQL database support
- [ ] Kubernetes operator
- [ ] Advanced analytics dashboard
- [ ] Custom domain support
- [ ] API rate limiting per customer
- [ ] Webhook notifications
- [ ] Advanced load balancing algorithms
- [ ] Multi-datacenter deployment
- [ ] Prometheus metrics export
- [ ] Grafana dashboard templates

## Changelog

### v1.0.0 (Current)
- Initial release
- Multi-provider support (Proxies.fo, Nettify)
- RESTful API
- Docker support
- Port pool management
- Health monitoring

---

**Made with ❤️ by the OceanProxy team**