// internal/handlers/plan.go
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

type PlanHandler struct {
	planService service.PlanService
	logger      *zap.Logger
}

func NewPlanHandler(planService service.PlanService, logger *zap.Logger) *PlanHandler {
	return &PlanHandler{
		planService: planService,
		logger:      logger,
	}
}

// CreatePlan creates a new proxy plan
// @Summary Create a new proxy plan
// @Description Create a new proxy plan with the specified configuration
// @Tags plans
// @Accept json
// @Produce json
// @Param request body domain.CreatePlanRequest true "Plan creation request"
// @Success 201 {object} domain.CreatePlanResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /plans [post]
func (h *PlanHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
    var req domain.CreatePlanRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
    // Enforce provider-specific credential rules
    if req.Provider == domain.ProviderProxiesFo {
        // Proxies.fo generates credentials; ignore any provided values
        req.Username = ""
        req.Password = ""
    } else if req.Provider == domain.ProviderNettify {
        // Nettify requires custom username/password
        if req.Username == "" || req.Password == "" {
            h.respondWithError(w, http.StatusBadRequest, "username and password are required for nettify provider", nil)
            return
        }
    }
	response, err := h.planService.CreatePlan(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create plan", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create plan", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, response)
}

// GetPlan retrieves a specific proxy plan
// @Summary Get a proxy plan
// @Description Get a proxy plan by ID
// @Tags plans
// @Produce json
// @Param id path string true "Plan ID"
// @Success 200 {object} domain.ProxyPlan
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /plans/{id} [get]
func (h *PlanHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	planIDStr := chi.URLParam(r, "id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid plan ID", err)
		return
	}

	plan, err := h.planService.GetPlan(r.Context(), planID)
	if err != nil {
		h.logger.Error("Failed to get plan", zap.Error(err))
		h.respondWithError(w, http.StatusNotFound, "Plan not found", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, plan)
}

// GetPlans retrieves all proxy plans or plans for a specific customer
// @Summary Get proxy plans
// @Description Get all proxy plans or filter by customer ID
// @Tags plans
// @Produce json
// @Param customer_id query string false "Customer ID to filter by"
// @Success 200 {array} domain.ProxyPlan
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /plans [get]
func (h *PlanHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")

	var plans []*domain.ProxyPlan
	var err error

	if customerID != "" {
		plans, err = h.planService.GetPlansByCustomer(r.Context(), customerID)
	} else {
		plans, err = h.planService.GetAllPlans(r.Context())
	}

	if err != nil {
		h.logger.Error("Failed to get plans", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get plans", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, plans)
}

// DeletePlan deletes a proxy plan
// @Summary Delete a proxy plan
// @Description Delete a proxy plan and all associated instances
// @Tags plans
// @Param id path string true "Plan ID"
// @Success 204
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /plans/{id} [delete]
func (h *PlanHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	planIDStr := chi.URLParam(r, "id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid plan ID", err)
		return
	}

	if err := h.planService.DeletePlan(r.Context(), planID); err != nil {
		h.logger.Error("Failed to delete plan", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete plan", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateProxiesFoPlan creates a plan using Proxies.fo provider (legacy endpoint)
// @Summary Create Proxies.fo plan
// @Description Create a proxy plan using Proxies.fo provider
// @Tags plans
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param reseller formData string true "Plan type (residential, datacenter, isp)"
// @Param bandwidth formData int true "Bandwidth in GB"
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 201 {object} domain.CreatePlanResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /plan [post]
func (h *PlanHandler) CreateProxiesFoPlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Failed to parse form", err)
		return
	}

	customerID := r.FormValue("customer_id")
	if customerID == "" {
		customerID = "legacy_customer_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	req := domain.CreatePlanRequest{
		CustomerID: customerID,
		PlanType:   r.FormValue("reseller"),
		Provider:   domain.ProviderProxiesFo,
		Region:     domain.RegionUSA, // Default to USA for legacy
		Username:   r.FormValue("username"),
		Password:   r.FormValue("password"),
	}

	bandwidth, err := strconv.Atoi(r.FormValue("bandwidth"))
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid bandwidth", err)
		return
	}
	req.Bandwidth = bandwidth

	response, err := h.planService.CreatePlan(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create Proxies.fo plan", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create plan", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, response)
}

// CreateNettifyPlan creates a plan using Nettify provider (legacy endpoint)
// @Summary Create Nettify plan
// @Description Create a proxy plan using Nettify provider
// @Tags plans
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param plan_type formData string true "Plan type (residential, datacenter, mobile)"
// @Param bandwidth formData int true "Bandwidth in GB"
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 201 {object} domain.CreatePlanResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Security BearerAuth
// @Router /nettify/plan [post]
func (h *PlanHandler) CreateNettifyPlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Failed to parse form", err)
		return
	}

	customerID := r.FormValue("customer_id")
	if customerID == "" {
		customerID = "legacy_customer_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	req := domain.CreatePlanRequest{
		CustomerID: customerID,
		PlanType:   r.FormValue("plan_type"),
		Provider:   domain.ProviderNettify,
		Region:     domain.RegionAlpha, // Default to Alpha for Nettify
		Username:   r.FormValue("username"),
		Password:   r.FormValue("password"),
	}

	bandwidth, err := strconv.Atoi(r.FormValue("bandwidth"))
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid bandwidth", err)
		return
	}
	req.Bandwidth = bandwidth

	response, err := h.planService.CreatePlan(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create Nettify plan", zap.Error(err))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create plan", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, response)
}

// GetStats returns statistics about plans
func (h *PlanHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	// This would be implemented to return plan statistics
	// For now, return placeholder data
	stats := map[string]interface{}{
		"total_plans":    0,
		"active_plans":   0,
		"expired_plans":  0,
		"failed_plans":   0,
		"creating_plans": 0,
	}

	h.respondWithJSON(w, http.StatusOK, stats)
}

// Helper methods
func (h *PlanHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *PlanHandler) respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	errorResponse := errors.NewErrorResponse(message, err)
	h.respondWithJSON(w, statusCode, errorResponse)
}
