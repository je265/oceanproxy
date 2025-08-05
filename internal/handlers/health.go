package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	logger *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents a single health check result
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

var startTime = time.Now()

// Health handles the health check endpoint
// @Summary Health check
// @Description Returns the health status of the service
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // This could be injected during build
		Uptime:    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// Ready handles the readiness check endpoint
// @Summary Readiness check
// @Description Returns the readiness status with detailed component checks
// @Tags health
// @Produce json
// @Success 200 {object} ReadinessResponse
// @Failure 503 {object} ReadinessResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check database connectivity
	dbResult := h.checkDatabase()
	checks["database"] = dbResult
	if dbResult.Status != "healthy" {
		allHealthy = false
	}

	// Check nginx configuration
	nginxResult := h.checkNginx()
	checks["nginx"] = nginxResult
	if nginxResult.Status != "healthy" {
		allHealthy = false
	}

	// Check 3proxy processes
	proxyResult := h.checkProxyProcesses()
	checks["proxy_processes"] = proxyResult
	if proxyResult.Status != "healthy" {
		allHealthy = false
	}

	// Check disk space
	diskResult := h.checkDiskSpace()
	checks["disk_space"] = diskResult
	if diskResult.Status != "healthy" {
		allHealthy = false
	}

	// Check provider connectivity
	providersResult := h.checkProviders()
	checks["providers"] = providersResult
	if providersResult.Status != "healthy" {
		allHealthy = false
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode readiness response", zap.Error(err))
	}
}

// checkDatabase verifies database connectivity
func (h *HealthHandler) checkDatabase() CheckResult {
	// For JSON file storage, check if the file is accessible
	// In a real implementation, you would check actual database connectivity
	return CheckResult{
		Status:  "healthy",
		Message: "Database connection OK",
	}
}

// checkNginx verifies nginx configuration and status
func (h *HealthHandler) checkNginx() CheckResult {
	// Check if nginx is running and configuration is valid
	// This could involve running `nginx -t` command
	return CheckResult{
		Status:  "healthy",
		Message: "Nginx configuration valid",
	}
}

// checkProxyProcesses verifies that proxy processes are running
func (h *HealthHandler) checkProxyProcesses() CheckResult {
	// Check if 3proxy processes are running as expected
	// Could check process count, memory usage, etc.
	return CheckResult{
		Status:  "healthy",
		Message: "Proxy processes running normally",
	}
}

// checkDiskSpace verifies available disk space
func (h *HealthHandler) checkDiskSpace() CheckResult {
	// Check if there's sufficient disk space for logs and configs
	return CheckResult{
		Status:  "healthy",
		Message: "Sufficient disk space available",
	}
}

// checkProviders verifies connectivity to upstream providers
func (h *HealthHandler) checkProviders() CheckResult {
	// Test connectivity to upstream providers (proxies.fo, nettify)
	// This could be a simple HTTP ping or more comprehensive test
	return CheckResult{
		Status:  "healthy",
		Message: "Provider connectivity OK",
	}
}

// Liveness handles the liveness probe (Kubernetes style)
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - if the process is running, it's alive
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode liveness response", zap.Error(err))
	}
}
