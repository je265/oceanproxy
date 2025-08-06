package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/app"
	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/pkg/logger"
)

// @title OceanProxy API
// @version 1.0
// @description Whitelabel HTTP Proxy Service API
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
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	zapLogger := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	defer zapLogger.Sync()

	// Create application
	application, err := app.New(cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to create application", zap.Error(err))
	}

	// Start server
	go func() {
		zapLogger.Info("Starting OceanProxy API server",
			zap.Int("port", cfg.Server.Port),
			zap.String("env", cfg.Environment),
		)
		if err := application.Start(); err != nil {
			zapLogger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}
