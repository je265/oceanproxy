package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/je265/oceanproxy/internal/handlers"
)

// SetupRoutes configures all HTTP routes
func (a *App) SetupRoutes() chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/ping"))

	// CORS middleware
	r.Use(a.corsMiddleware)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/health", a.healthHandler.Health)
		r.Get("/ready", a.healthHandler.Ready)
		r.Get("/metrics", a.metricsHandler)
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication required for all API routes
		r.Use(handlers.NewAuthMiddleware(a.cfg.Auth.BearerToken, a.logger))

		// Plans
		r.Route("/plans", func(r chi.Router) {
			r.Post("/", a.planHandler.CreatePlan)
			r.Get("/", a.planHandler.GetPlans)
			r.Get("/{id}", a.planHandler.GetPlan)
			r.Put("/{id}", a.planHandler.UpdatePlan)
			r.Delete("/{id}", a.planHandler.DeletePlan)
		})

		// Proxies
		r.Route("/proxies", func(r chi.Router) {
			r.Get("/", a.proxyHandler.GetProxies)
			r.Get("/{id}", a.proxyHandler.GetProxy)
			r.Post("/{id}/start", a.proxyHandler.StartProxy)
			r.Post("/{id}/stop", a.proxyHandler.StopProxy)
			r.Post("/{id}/restart", a.proxyHandler.RestartProxy)
			r.Get("/{id}/status", a.proxyHandler.GetProxyStatus)
			r.Get("/{id}/logs", a.proxyHandler.GetProxyLogs)
		})

		// Statistics and monitoring
		r.Get("/stats", a.statsHandler)
		r.Get("/stats/ports", a.portStatsHandler)
		r.Get("/stats/providers", a.providerStatsHandler)

		// Provider specific routes
		r.Route("/providers", func(r chi.Router) {
			r.Get("/", a.getProvidersHandler)
			r.Post("/{provider}/test", a.testProviderHandler)
		})

		// Configuration
		r.Route("/config", func(r chi.Router) {
			r.Get("/regions", a.getRegionsHandler)
			r.Get("/plan-types", a.getPlanTypesHandler)
		})
	})

	// Legacy API endpoints for backward compatibility
	r.Group(func(r chi.Router) {
		r.Use(handlers.NewAuthMiddleware(a.cfg.Auth.BearerToken, a.logger))

		// Proxies.fo legacy endpoint
		r.Post("/plan", a.planHandler.CreateProxiesFoPlan)

		// Nettify legacy endpoint
		r.Post("/nettify/plan", a.planHandler.CreateNettifyPlan)
	})

	// Admin routes (if needed)
	r.Route("/admin", func(r chi.Router) {
		r.Use(handlers.NewAuthMiddleware(a.cfg.Auth.BearerToken, a.logger))

		r.Post("/nginx/reload", a.reloadNginxHandler)
		r.Post("/cleanup", a.cleanupHandler)
		r.Get("/debug/ports", a.debugPortsHandler)
	})

	// Swagger/OpenAPI documentation
	r.Route("/docs", func(r chi.Router) {
		fileServer := http.FileServer(http.Dir("./web/static/swagger/"))
		r.Handle("/*", http.StripPrefix("/docs", fileServer))
	})

	return r
}

// corsMiddleware handles CORS headers
func (a *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		for _, origin := range a.cfg.Server.CORS.AllowOrigins {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods",
			joinStrings(a.cfg.Server.CORS.AllowMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers",
			joinStrings(a.cfg.Server.CORS.AllowHeaders, ", "))

		if a.cfg.Server.CORS.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Additional handler methods
func (a *App) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement metrics endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"metrics_endpoint"}`))
}

func (a *App) statsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement general stats endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"stats_endpoint"}`))
}

func (a *App) portStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement port statistics endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"port_stats_endpoint"}`))
}

func (a *App) providerStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement provider statistics endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"provider_stats_endpoint"}`))
}

func (a *App) getProvidersHandler(w http.ResponseWriter, r *http.Request) {
	// Implement get providers endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"providers":["proxies_fo","nettify"]}`))
}

func (a *App) testProviderHandler(w http.ResponseWriter, r *http.Request) {
	// Implement test provider endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"test_provider_endpoint"}`))
}

func (a *App) getRegionsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement get regions endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"regions":["usa","eu","alpha","beta","asia"]}`))
}

func (a *App) getPlanTypesHandler(w http.ResponseWriter, r *http.Request) {
	// Implement get plan types endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"plan_types":["residential","datacenter","isp","mobile","unlimited"]}`))
}

func (a *App) reloadNginxHandler(w http.ResponseWriter, r *http.Request) {
	// Implement nginx reload endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"nginx_reload_endpoint"}`))
}

func (a *App) cleanupHandler(w http.ResponseWriter, r *http.Request) {
	// Implement cleanup endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"cleanup_endpoint"}`))
}

func (a *App) debugPortsHandler(w http.ResponseWriter, r *http.Request) {
	// Implement debug ports endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"debug_ports_endpoint"}`))
}

// Helper function
func joinStrings(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	if len(slice) == 1 {
		return slice[0]
	}

	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}
