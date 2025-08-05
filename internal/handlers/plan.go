package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
func (h *PlanHandler) CreatePlan(c *gin.Context) {
	var req domain.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, errors.NewErrorResponse("Invalid request body", err))
		return
	}

	response, err := h.planService.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create plan", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errors.NewErrorResponse("Failed to create plan", err))
		return
	}

	c.JSON(http.StatusCreated, response)
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
func (h *PlanHandler) GetPlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewErrorResponse("Invalid plan ID", err))
		return
	}

	plan, err := h.planService.GetPlan(c.Request.Context(), planID)
	if err != nil {
		h.logger.Error("Failed to get plan", zap.Error(err))
		c.JSON(http.StatusNotFound, errors.NewErrorResponse("Plan not found", err))
		return
	}

	c.JSON(http.StatusOK, plan)
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
func (h *PlanHandler) GetPlans(c *gin.Context) {
	customerID := c.Query("customer_id")

	var plans []*domain.ProxyPlan
	var err error

	if customerID != "" {
		plans, err = h.planService.GetPlansByCustomer(c.Request.Context(), customerID)
	} else {
		plans, err = h.planService.GetAllPlans(c.Request.Context())
	}

	if err != nil {
		h.logger.Error("Failed to get plans", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errors.NewErrorResponse("Failed to get plans", err))
		return
	}

	c.JSON(http.StatusOK, plans)
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
func (h *PlanHandler) DeletePlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewErrorResponse("Invalid plan ID", err))
		return
	}

	if err := h.planService.DeletePlan(c.Request.Context(), planID); err != nil {
		h.logger.Error("Failed to delete plan", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errors.NewErrorResponse("Failed to delete plan", err))
		return
	}

	c.Status(http.StatusNoContent)
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
func (h *PlanHandler) CreateProxiesFoPlan(c *gin.Context) {
	req := domain.CreatePlanRequest{
		CustomerID: c.PostForm("customer_id"),
		PlanType:   c.PostForm("reseller"),
		Provider:   domain.ProviderProxiesFo,
		Username:   c.PostForm("username"),
		Password:   c.PostForm("password"),
	}

	if req.CustomerID == "" {
		req.CustomerID = "legacy_customer_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	bandwidth, err := strconv.Atoi(c.PostForm("bandwidth"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewErrorResponse("Invalid bandwidth", err))
		return
	}
	req.Bandwidth = bandwidth

	response, err := h.planService.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create Proxies.fo plan", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errors.NewErrorResponse("Failed to create plan", err))
		return
	}

	c.JSON(http.StatusCreated, response)
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
func (h *PlanHandler) CreateNettifyPlan(c *gin.Context) {
	req := domain.CreatePlanRequest{
		CustomerID: c.PostForm("customer_id"),
		PlanType:   c.PostForm("plan_type"),
		Provider:   domain.ProviderNettify,
		Username:   c.PostForm("username"),
		Password:   c.PostForm("password"),
	}

	if req.CustomerID == "" {
		req.CustomerID = "legacy_customer_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	bandwidth, err := strconv.Atoi(c.PostForm("bandwidth"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewErrorResponse("Invalid bandwidth", err))
		return
	}
	req.Bandwidth = bandwidth

	response, err := h.planService.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create Nettify plan", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errors.NewErrorResponse("Failed to create plan", err))
		return
	}

	c.JSON(http.StatusCreated, response)
}
