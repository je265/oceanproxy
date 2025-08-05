oceanproxy/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── api/
│   └── openapi.yaml
├── build/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── nginx/
│       └── nginx.conf
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── cli/
│       └── main.go
├── configs/
│   ├── config.yaml
│   └── config.local.yaml
├── deployments/
│   ├── systemd/
│   │   └── oceanproxy.service
│   └── scripts/
│       ├── install.sh
│       └── setup.sh
├── docs/
│   ├── README.md
│   ├── API.md
│   └── DEPLOYMENT.md
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── routes.go
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── plan.go
│   │   ├── proxy.go
│   │   └── user.go
│   ├── handlers/
│   │   ├── health.go
│   │   ├── plan.go
│   │   ├── proxy.go
│   │   └── middleware.go
│   ├── repository/
│   │   ├── interfaces.go
│   │   ├── json/
│   │   │   └── proxy.go
│   │   └── postgres/
│   │       └── proxy.go
│   ├── service/
│   │   ├── interfaces.go
│   │   ├── plan.go
│   │   ├── proxy.go
│   │   └── provider/
│   │       ├── nettify.go
│   │       └── proxiesfo.go
│   └── pkg/
│       ├── logger/
│       │   └── logger.go
│       ├── validator/
│       │   └── validator.go
│       ├── errors/
│       │   └── errors.go
│       └── utils/
│           ├── crypto.go
│           ├── http.go
│           └── port.go
├── scripts/
│   ├── proxy/
│   │   ├── create_proxy_plan.sh
│   │   ├── cleanup.sh
│   │   └── health_check.sh
│   └── system/
│       ├── install_deps.sh
│       └── setup_nginx.sh
├── test/
│   ├── integration/
│   │   └── api_test.go
│   └── mocks/
│       └── providers.go
├── web/
│   └── static/
│       └── swagger/
├── go.mod
├── go.sum
├── Makefile
├── .env.example
├── .gitignore
└── README.md