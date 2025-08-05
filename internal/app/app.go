package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/handlers"
	"github.com/je265/oceanproxy/internal/pkg/logger"
	"github.com/je265/oceanproxy/internal/repository/json"
	"github.com/je265/oceanproxy/internal/service"
)

// App represents the application
type App struct {
	cfg    *config.Config
	logger *zap.Logger
	server *http.Server
	router chi.Router
}

// New creates a new application instance
func New(cfg *config.Config, log *zap.Logger) (*App, error) {
	app := &App{
		cfg:    cfg,
		logger: log,
	}

	// Initialize repositories
	planRepo := json.NewPlanRepository(cfg.Database.DSN, log)
	instanceRepo := json.NewInstanceRepository(cfg.Database.DSN, log)

	// Initialize services
	providerService := service.NewProviderService(cfg, log)
	proxyService := service.NewProxyService(cfg, log)

	// Load plan type configurations
	planTypes, err := loadPlanTypeConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load plan type configs: %w", err)
	}

	// Load region configurations
	regions, err := loadRegionConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load region configs: %w", err)
	}

	// Initialize port manager
	portManager := service.NewPortManager(log, planTypes)

	// Initialize nginx manager
	nginxManager := service.NewNginxManager(log, cfg, regions, planTypes)

	// Initialize plan service with all dependencies
	planService := service.NewPlanService(
		cfg,
		log,
		planRepo,
		instanceRepo,
		providerService,
		proxyService,
		portManager,
		nginxManager,
		regions,
	)

	// Initialize handlers
	planHandler := handlers.NewPlanHandler(planService, log)
	proxyHandler := handlers.NewProxyHandler(proxyService, log)
	healthHandler := handlers.NewHealthHandler(log)

	// Setup router
	app.setupRouter(planHandler, proxyHandler, healthHandler)

	// Create HTTP server
	app.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      app.router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return app, nil
}

// Start starts the application
func (a *App) Start() error {
	a.logger.Info("Starting HTTP server",
		zap.String("addr", a.server.Addr),
	)

	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down HTTP server")

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}

// setupRouter configures the HTTP router
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

	// Health check
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication middleware for API routes
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
func loadPlanTypeConfigs() (map[string]*domain.PlanTypeConfig, error) {
	// This would typically load from proxy-plans.yaml
	// For now, returning empty map - implement yaml loading
	return make(map[string]*domain.PlanTypeConfig), nil
}

func loadRegionConfigs() (map[string]*domain.Region, error) {
	// This would typically load from regions.yaml
	// For now, returning empty map - implement yaml loading
	return make(map[string]*domain.Region), nil
}
