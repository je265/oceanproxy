// internal/app/app.go - FIXED application setup with proper authentication
package app

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/handlers"
	"github.com/je265/oceanproxy/internal/repository/json"
	"github.com/je265/oceanproxy/internal/service"
	"github.com/je265/oceanproxy/pkg/config"
)

// App represents the application
type App struct {
	cfg    *config.Config
	logger *zap.Logger
	router chi.Router
}

// New creates a new application instance
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	app := &App{
		cfg:    cfg,
		logger: logger,
	}

	logger.Info("Initializing OceanProxy application",
		zap.String("environment", cfg.Environment),
		zap.String("database_driver", cfg.Database.Driver),
		zap.String("proxy_domain", cfg.Proxy.Domain),
	)

	// Initialize repositories
	planRepo := json.NewPlanRepository(cfg.Database.DSN, logger)
	instanceRepo := json.NewInstanceRepository(cfg.Database.DSN, logger)

	// Load plan type configurations
	planTypes, err := loadPlanTypeConfigs(logger)
	if err != nil {
		logger.Warn("Failed to load plan type configs, using defaults", zap.Error(err))
		planTypes = getDefaultPlanTypes()
	}

	// Load region configurations
	regions, err := loadRegionConfigs(logger)
	if err != nil {
		logger.Warn("Failed to load region configs, using defaults", zap.Error(err))
		regions = getDefaultRegions()
	}

	logger.Info("Loaded configurations",
		zap.Int("plan_types", len(planTypes)),
		zap.Int("regions", len(regions)),
	)

	// Initialize services
	providerService := service.NewProviderService(cfg, logger)
	proxyService := service.NewProxyService(cfg, logger, instanceRepo, planRepo)
	portManager := service.NewPortManager(logger, planTypes)
	nginxManager := service.NewNginxManager(logger, cfg, regions, planTypes)

	planService := service.NewPlanService(
		cfg,
		logger,
		planRepo,
		instanceRepo,
		providerService,
		proxyService,
		portManager,
		nginxManager,
		regions,
	)

	// Initialize handlers
	planHandler := handlers.NewPlanHandler(planService, logger)
	proxyHandler := handlers.NewProxyHandler(proxyService, logger)
	healthHandler := handlers.NewHealthHandler(logger)

	// Setup router
	app.setupRouter(planHandler, proxyHandler, healthHandler)

	logger.Info("Application initialized successfully")

	return app, nil
}

// Router returns the HTTP router
func (a *App) Router() chi.Router {
	return a.router
}

// setupRouter configures the HTTP router with FIXED authentication
func (a *App) setupRouter(
	planHandler *handlers.PlanHandler,
	proxyHandler *handlers.ProxyHandler,
	healthHandler *handlers.HealthHandler,
) {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, origin := range a.cfg.Server.CORS.AllowOrigins {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			for _, method := range a.cfg.Server.CORS.AllowMethods {
				w.Header().Add("Access-Control-Allow-Methods", method)
			}
			for _, header := range a.cfg.Server.CORS.AllowHeaders {
				w.Header().Add("Access-Control-Allow-Headers", header)
			}
			if a.cfg.Server.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health checks (no auth required)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Log the bearer token being used (for debugging)
	a.logger.Info("Setting up authentication",
		zap.String("bearer_token", a.cfg.Auth.BearerToken),
	)

	// API routes with authentication
	r.Route("/api/v1", func(r chi.Router) {
		// FIXED: Use the correct bearer token from config
		r.Use(handlers.NewAuthMiddleware(a.cfg.Auth.BearerToken, a.logger))

		// Plan management
		r.Route("/plans", func(r chi.Router) {
			r.Post("/", planHandler.CreatePlan)
			r.Get("/", planHandler.GetPlans)
			r.Get("/{id}", planHandler.GetPlan)
			r.Delete("/{id}", planHandler.DeletePlan)
		})

		// Proxy management
		r.Route("/proxies", func(r chi.Router) {
			r.Get("/", proxyHandler.GetProxies)
			r.Get("/{id}", proxyHandler.GetProxy)
			r.Post("/{id}/start", proxyHandler.StartProxy)
			r.Post("/{id}/stop", proxyHandler.StopProxy)
			r.Post("/{id}/restart", proxyHandler.RestartProxy)
			r.Get("/{id}/status", proxyHandler.GetProxyStatus)
		})

		// Statistics
		r.Get("/stats", planHandler.GetStats)
	})

	// Legacy endpoints for backward compatibility
	r.Route("/", func(r chi.Router) {
		r.Use(handlers.NewAuthMiddleware(a.cfg.Auth.BearerToken, a.logger))

		// Proxies.fo legacy endpoint
		r.Post("/plan", planHandler.CreateProxiesFoPlan)

		// Nettify legacy endpoint
		r.Post("/nettify/plan", planHandler.CreateNettifyPlan)
	})

	a.router = r
}

// Helper functions to load configurations
func loadPlanTypeConfigs(logger *zap.Logger) (map[string]*domain.PlanTypeConfig, error) {
	// Try multiple paths for plan type configs
	configPaths := []string{
		"/etc/oceanproxy/proxy-plans.yaml",
		"./configs/proxy-plans.yaml",
		"./proxy-plans.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			logger.Info("Loading plan type configuration", zap.String("path", path))
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var config struct {
				PlanTypes map[string]*domain.PlanTypeConfig `yaml:"plan_types"`
			}

			if err := yaml.Unmarshal(data, &config); err != nil {
				logger.Error("Failed to parse plan types config", zap.Error(err))
				continue
			}

			return config.PlanTypes, nil
		}
	}

	return nil, fmt.Errorf("no plan type configuration file found")
}

func loadRegionConfigs(logger *zap.Logger) (map[string]*domain.Region, error) {
	// Try multiple paths for region configs
	configPaths := []string{
		"/etc/oceanproxy/regions.yaml",
		"./configs/regions.yaml",
		"./regions.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			logger.Info("Loading region configuration", zap.String("path", path))
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var config struct {
				Regions map[string]*domain.Region `yaml:"regions"`
			}

			if err := yaml.Unmarshal(data, &config); err != nil {
				logger.Error("Failed to parse regions config", zap.Error(err))
				continue
			}

			return config.Regions, nil
		}
	}

	return nil, fmt.Errorf("no region configuration file found")
}

// Default configurations
func getDefaultPlanTypes() map[string]*domain.PlanTypeConfig {
	return map[string]*domain.PlanTypeConfig{
		"proxies_fo_usa_residential": {
			Name:         "proxies_fo_usa_residential",
			Provider:     "proxies_fo",
			Region:       "usa",
			PlanType:     "residential",
			UpstreamHost: "pr-us.proxies.fo",
			UpstreamPort: 13337,
			LocalPortRange: domain.PortRange{
				Start: 10000,
				End:   11999,
			},
			OutboundPort:      1337,
			NginxUpstreamName: "oceanproxy_usa_residential",
		},
		"proxies_fo_usa_datacenter": {
			Name:         "proxies_fo_usa_datacenter",
			Provider:     "proxies_fo",
			Region:       "usa",
			PlanType:     "datacenter",
			UpstreamHost: "dcp.proxies.fo",
			UpstreamPort: 13338,
			LocalPortRange: domain.PortRange{
				Start: 12000,
				End:   13999,
			},
			OutboundPort:      1337,
			NginxUpstreamName: "oceanproxy_usa_datacenter",
		},
		"nettify_alpha_residential": {
			Name:         "nettify_alpha_residential",
			Provider:     "nettify",
			Region:       "alpha",
			PlanType:     "residential",
			UpstreamHost: "proxy.nettify.xyz",
			UpstreamPort: 8080,
			LocalPortRange: domain.PortRange{
				Start: 22000,
				End:   23999,
			},
			OutboundPort:      9876,
			NginxUpstreamName: "oceanproxy_alpha_residential",
		},
	}
}

func getDefaultRegions() map[string]*domain.Region {
	return map[string]*domain.Region{
		"usa": {
			Name:         "usa",
			Subdomain:    "usa",
			DomainSuffix: "oceanproxy.io",
			OutboundPort: 1337,
			Description:  "United States proxies",
			PlanTypes: []string{
				"proxies_fo_usa_residential",
				"proxies_fo_usa_datacenter",
			},
			NginxConfigFile: "oceanproxy_usa.conf",
		},
		"eu": {
			Name:         "eu",
			Subdomain:    "eu",
			DomainSuffix: "oceanproxy.io",
			OutboundPort: 1338,
			Description:  "European Union proxies",
			PlanTypes: []string{
				"proxies_fo_eu_residential",
				"proxies_fo_eu_datacenter",
			},
			NginxConfigFile: "oceanproxy_eu.conf",
		},
		"alpha": {
			Name:         "alpha",
			Subdomain:    "alpha",
			DomainSuffix: "oceanproxy.io",
			OutboundPort: 9876,
			Description:  "Alpha region proxies",
			PlanTypes: []string{
				"nettify_alpha_residential",
				"nettify_alpha_mobile",
			},
			NginxConfigFile: "oceanproxy_alpha.conf",
		},
	}
}
