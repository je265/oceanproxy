package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/pkg/errors"
	"github.com/je265/oceanproxy/internal/service"
)

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	proxyService service.ProxyService
	logger       *zap.Logger
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(proxyService service.ProxyService, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		logger:       logger,
	}
}

// GetProxies retrieves all proxy instances
// @Summary Get all proxy instances
// @Description Get all proxy instances with optional filtering
// @Tags proxies
// @Produce json
// @Param status query string false "Filter by status"
// @Param plan_id query string false "Filter by plan ID"
// @Success 200 {array} domain.ProxyInstance
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies [get]
func (h *ProxyHandler) GetProxies(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	planIDStr := r.URL.Query().Get("plan_id")

	var instances []*domain.ProxyInstance
	var err error

	if planIDStr != "" {
		planID, parseErr := uuid.Parse(planIDStr)
		if parseErr != nil {
			h.respondWithError(w, http.StatusBadRequest, "Invalid plan ID", parseErr)
			return
		}
		instances, err = h.proxyService.GetInstancesByPlan(r.Context(), planID)
	} else {
		instances, err = h.proxyService.GetRunningInstances(r.Context())
	}

	if err != nil {
		h.logger.Error("Failed to get proxy instances", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get proxy instances", err)
		return
	}

	// Filter by status if provided
	if status != "" {
		filtered := make([]*domain.ProxyInstance, 0)
		for _, instance := range instances {
			if instance.Status == status {
				filtered = append(filtered, instance)
			}
		}
		instances = filtered
	}

	h.respondWithJSON(w, http.StatusOK, instances)
}

// GetProxy retrieves a specific proxy instance
// @Summary Get proxy instance
// @Description Get a proxy instance by ID
// @Tags proxies
// @Produce json
// @Param id path string true "Proxy Instance ID"
// @Success 200 {object} domain.ProxyInstance
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id} [get]
func (h *ProxyHandler) GetProxy(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	instance, err := h.proxyService.GetInstance(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get proxy instance", zap.Error(err))
		h.respondWithError(w, http.StatusNotFound, "Proxy instance not found", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, instance)
}

// StartProxy starts a proxy instance
// @Summary Start proxy instance
// @Description Start a proxy instance
// @Tags proxies
// @Param id path string true "Proxy Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id}/start [post]
func (h *ProxyHandler) StartProxy(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	// Get the instance first
	instance, err := h.proxyService.GetInstance(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get proxy instance", zap.Error(err))
		h.respondWithError(w, http.StatusNotFound, "Proxy instance not found", err)
		return
	}

	// Start the instance
	if err := h.proxyService.StartInstance(r.Context(), instance); err != nil {
		h.logger.Error("Failed to start proxy instance",
			zap.String("instance_id", instanceID.String()),
			zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to start proxy instance", err)
		return
	}

	h.logger.Info("Proxy instance started successfully",
		zap.String("instance_id", instanceID.String()),
		zap.Int("local_port", instance.LocalPort))

	response := map[string]interface{}{
		"success":     true,
		"message":     "Proxy instance started successfully",
		"instance_id": instanceID,
		"status":      "starting",
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// StopProxy stops a proxy instance
// @Summary Stop proxy instance
// @Description Stop a proxy instance
// @Tags proxies
// @Param id path string true "Proxy Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id}/stop [post]
func (h *ProxyHandler) StopProxy(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	if err := h.proxyService.StopInstance(r.Context(), instanceID); err != nil {
		h.logger.Error("Failed to stop proxy instance",
			zap.String("instance_id", instanceID.String()),
			zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to stop proxy instance", err)
		return
	}

	h.logger.Info("Proxy instance stopped successfully",
		zap.String("instance_id", instanceID.String()))

	response := map[string]interface{}{
		"success":     true,
		"message":     "Proxy instance stopped successfully",
		"instance_id": instanceID,
		"status":      "stopped",
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// RestartProxy restarts a proxy instance
// @Summary Restart proxy instance
// @Description Restart a proxy instance
// @Tags proxies
// @Param id path string true "Proxy Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id}/restart [post]
func (h *ProxyHandler) RestartProxy(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	if err := h.proxyService.RestartInstance(r.Context(), instanceID); err != nil {
		h.logger.Error("Failed to restart proxy instance",
			zap.String("instance_id", instanceID.String()),
			zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to restart proxy instance", err)
		return
	}

	h.logger.Info("Proxy instance restarted successfully",
		zap.String("instance_id", instanceID.String()))

	response := map[string]interface{}{
		"success":     true,
		"message":     "Proxy instance restarted successfully",
		"instance_id": instanceID,
		"status":      "restarting",
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// GetProxyStatus gets the status of a proxy instance
// @Summary Get proxy instance status
// @Description Get the current status of a proxy instance
// @Tags proxies
// @Produce json
// @Param id path string true "Proxy Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id}/status [get]
func (h *ProxyHandler) GetProxyStatus(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	status, err := h.proxyService.GetInstanceStatus(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get proxy instance status",
			zap.String("instance_id", instanceID.String()),
			zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get proxy instance status", err)
		return
	}

	// Perform health check
	healthErr := h.proxyService.HealthCheck(r.Context(), instanceID)
	isHealthy := healthErr == nil

	response := map[string]interface{}{
		"instance_id": instanceID,
		"status":      status,
		"healthy":     isHealthy,
		"timestamp":   time.Now(),
	}

	if !isHealthy {
		response["health_error"] = healthErr.Error()
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// GetProxyLogs gets the logs for a proxy instance
// @Summary Get proxy instance logs
// @Description Get the logs for a proxy instance
// @Tags proxies
// @Produce json
// @Param id path string true "Proxy Instance ID"
// @Param lines query int false "Number of log lines to return" default(100)
// @Param follow query bool false "Follow log output" default(false)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /proxies/{id}/logs [get]
func (h *ProxyHandler) GetProxyLogs(w http.ResponseWriter, r *http.Request) {
	instanceIDStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}

	// Parse query parameters
	linesStr := r.URL.Query().Get("lines")
	lines := 100 // default
	if linesStr != "" {
		if parsedLines, err := strconv.Atoi(linesStr); err == nil && parsedLines > 0 {
			lines = parsedLines
		}
	}

	followStr := r.URL.Query().Get("follow")
	follow := followStr == "true"

	// Get the instance to validate it exists
	instance, err := h.proxyService.GetInstance(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get proxy instance for logs", zap.Error(err))
		h.respondWithError(w, http.StatusNotFound, "Proxy instance not found", err)
		return
	}

	// For now, return mock logs. In a real implementation, you would:
	// 1. Read from the actual log file at /var/log/oceanproxy/3proxy_{plan_id}.log
	// 2. Handle following logs with streaming response
	// 3. Parse and format log entries

	mockLogs := []string{
		"2024-01-15 10:30:15 [INFO] Proxy instance started on port " + strconv.Itoa(instance.LocalPort),
		"2024-01-15 10:30:16 [INFO] Connected to upstream " + instance.AuthHost + ":" + strconv.Itoa(instance.AuthPort),
		"2024-01-15 10:35:22 [INFO] Client connection from 192.168.1.100",
		"2024-01-15 10:35:23 [INFO] Forwarding request to upstream",
		"2024-01-15 10:35:24 [INFO] Response forwarded to client",
	}

	// Limit to requested number of lines
	if len(mockLogs) > lines {
		mockLogs = mockLogs[len(mockLogs)-lines:]
	}

	response := map[string]interface{}{
		"instance_id": instanceID,
		"lines":       lines,
		"follow":      follow,
		"logs":        mockLogs,
		"timestamp":   time.Now(),
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// Helper methods
func (h *ProxyHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *ProxyHandler) respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	errorResponse := errors.NewErrorResponse(message, err)
	h.respondWithJSON(w, statusCode, errorResponse)
}
