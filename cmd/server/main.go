// cmd/server/main.go - FIXED with proper error handling and graceful shutdown
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/app"
	"github.com/je265/oceanproxy/pkg/config"
	"github.com/je265/oceanproxy/pkg/logger"
)

// @title OceanProxy API
// @version 1.0
// @description Complete White-label HTTP Proxy Service API
// @termsOfService https://oceanproxy.io/terms

// @contact.name API Support
// @contact.url https://oceanproxy.io/support
// @contact.email support@oceanproxy.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Build information (injected during build)
	var (
		Version   = "1.0.0"
		BuildTime = "unknown"
		GitCommit = "unknown"
	)

	fmt.Printf("🌊 OceanProxy v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	zapLogger := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			// Ignore sync errors on stdout/stderr
		}
	}()

	zapLogger.Info("Starting OceanProxy",
		zap.String("version", Version),
		zap.String("environment", cfg.Environment),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("bearer_token_set", func() string {
			if cfg.Auth.BearerToken != "" {
				return "yes"
			}
			return "no"
		}()),
	)

	// Create application
	application, err := app.New(cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to create application", zap.Error(err))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      application.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		zapLogger.Info("HTTP server starting",
			zap.String("addr", server.Addr),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	} else {
		zapLogger.Info("Server exited gracefully")
	}
}
