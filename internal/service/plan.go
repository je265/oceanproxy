package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/repository"
)

type planService struct {
	cfg             *config.Config
	logger          *zap.Logger
	planRepo        repository.PlanRepository
	instanceRepo    repository.InstanceRepository
	providerService ProviderService
	proxyService    ProxyService
}

func NewPlanService(
	cfg *config.Config,
	logger *zap.Logger,
	planRepo repository.PlanRepository,
	instanceRepo repository.InstanceRepository,
	providerService ProviderService,
	proxyService ProxyService,
) PlanService {
	return &planService{
		cfg:             cfg,
		logger:          logger,
		planRepo:        planRepo,
		instanceRepo:    instanceRepo,
		providerService: providerService,
		proxyService:    proxyService,
	}
}

func (s *planService) CreatePlan(ctx context.Context, req *domain.CreatePlanRequest) (*domain.CreatePlanResponse, error) {
	s.logger.Info("Creating new proxy plan",
		zap.String("customer_id", req.CustomerID),
		zap.String("plan_type", req.PlanType),
		zap.String("provider", req.Provider),
	)

	// Find the appropriate plan type configuration
	planTypeKey, err := s.portManager.FindPlanTypeByProviderAndRegion(req.Provider, req.Region, req.PlanType)
	if err != nil {
		return nil, fmt.Errorf("unsupported plan configuration: %w", err)
	}

	planTypeConfig, err := s.portManager.GetPlanTypeConfig(planTypeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan type config: %w", err)
	}

	// Create plan record
	plan := &domain.ProxyPlan{
		ID:          uuid.New(),
		CustomerID:  req.CustomerID,
		PlanType:    req.PlanType,
		Provider:    req.Provider,
		Region:      req.Region,
		PlanTypeKey: planTypeKey,
		Username:    req.Username,
		Password:    req.Password,
		Status:      domain.PlanStatusCreating,
		Bandwidth:   req.Bandwidth,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set expiration
	if req.Duration > 0 {
		plan.ExpiresAt = time.Now().AddDate(0, 0, req.Duration)
	} else {
		plan.ExpiresAt = time.Now().AddDate(0, 0, 180)
	}

	// Save plan to repository
	if err := s.planRepo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Create upstream provider account
	providerAccount, err := s.providerService.CreateAccount(ctx, req.Provider, req)
	if err != nil {
		plan.Status = domain.PlanStatusFailed
		s.planRepo.Update(ctx, plan)
		return nil, fmt.Errorf("failed to create provider account: %w", err)
	}

	// Allocate local port
	localPort, err := s.portManager.AllocatePort(ctx, planTypeKey, plan.ID.String())
	if err != nil {
		plan.Status = domain.PlanStatusFailed
		s.planRepo.Update(ctx, plan)
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// Create proxy instance
	instance := &domain.ProxyInstance{
		ID:          uuid.New(),
		PlanID:      plan.ID,
		PlanTypeKey: planTypeKey,
		LocalPort:   localPort,
		AuthHost:    planTypeConfig.UpstreamHost,
		AuthPort:    planTypeConfig.UpstreamPort,
		Status:      domain.InstanceStatusStarting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		s.portManager.ReleasePort(ctx, planTypeKey, localPort)
		plan.Status = domain.PlanStatusFailed
		s.planRepo.Update(ctx, plan)
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Start 3proxy instance
	if err := s.proxyService.StartInstance(ctx, instance); err != nil {
		s.logger.Error("Failed to start proxy instance", zap.Error(err))
		// Continue - we can retry later
	}

	// Update nginx configuration
	if err := s.nginxManager.UpdateUpstream(ctx, planTypeKey, localPort); err != nil {
		s.logger.Error("Failed to update nginx upstream", zap.Error(err))
		// Continue - nginx can be updated manually if needed
	}

	// Update plan status to active
	plan.Status = domain.PlanStatusActive
	plan.Instances = []*domain.ProxyInstance{instance}
	if err := s.planRepo.Update(ctx, plan); err != nil {
		s.logger.Error("Failed to update plan status", zap.Error(err))
	}

	// Build response with region-specific endpoint
	region := s.regions[planTypeConfig.Region]
	response := &domain.CreatePlanResponse{
		Success:   true,
		PlanID:    plan.ID,
		Username:  plan.Username,
		Password:  plan.Password,
		ExpiresAt: plan.ExpiresAt,
		Proxies: []domain.ProxyEndpoint{
			{
				URL:      region.GetProxyEndpoint(plan.Username, plan.Password),
				Region:   region.Name,
				Username: plan.Username,
				Password: plan.Password,
			},
		},
	}

	s.logger.Info("Successfully created proxy plan",
		zap.String("plan_id", plan.ID.String()),
		zap.String("plan_type_key", planTypeKey),
		zap.Int("local_port", localPort),
		zap.String("endpoint", response.Proxies[0].URL),
	)

	return response, nil
}

func (s *planService) createInstances(ctx context.Context, plan *domain.ProxyPlan, account *ProviderAccount) ([]*domain.ProxyInstance, error) {
	var instances []*domain.ProxyInstance

	// Determine regions based on provider
	regions := s.getRegionsForProvider(plan.Provider)

	for _, region := range regions {
		// Find available port
		port, err := s.findAvailablePort(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to find available port: %w", err)
		}

		instance := &domain.ProxyInstance{
			ID:        uuid.New(),
			PlanID:    plan.ID,
			Region:    region,
			LocalPort: port,
			AuthHost:  account.Host,
			AuthPort:  account.Port,
			LocalHost: fmt.Sprintf("%s.%s", region, s.cfg.Proxy.Domain),
			Status:    domain.InstanceStatusStarting,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.instanceRepo.Create(ctx, instance); err != nil {
			return nil, fmt.Errorf("failed to create instance: %w", err)
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func (s *planService) getRegionsForProvider(provider string) []string {
	switch provider {
	case domain.ProviderProxiesFo:
		return []string{domain.RegionUSA, domain.RegionEU}
	case domain.ProviderNettify:
		return []string{domain.RegionAlpha}
	default:
		return []string{domain.RegionUSA}
	}
}

func (s *planService) findAvailablePort(ctx context.Context) (int, error) {
	// Get all used ports
	instances, err := s.instanceRepo.GetAll(ctx)
	if err != nil {
		return 0, err
	}

	usedPorts := make(map[int]bool)
	for _, instance := range instances {
		usedPorts[instance.LocalPort] = true
	}

	// Find first available port in range
	for port := s.cfg.Proxy.StartPort; port <= s.cfg.Proxy.EndPort; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", s.cfg.Proxy.StartPort, s.cfg.Proxy.EndPort)
}

func (s *planService) buildProxyEndpoints(instances []*domain.ProxyInstance, username, password string) []domain.ProxyEndpoint {
	var endpoints []domain.ProxyEndpoint

	for _, instance := range instances {
		var port int
		switch instance.Region {
		case domain.RegionUSA:
			port = 1337
		case domain.RegionEU:
			port = 1338
		case domain.RegionAlpha:
			port = 9876
		default:
			port = 1337
		}

		endpoint := domain.ProxyEndpoint{
			URL:      fmt.Sprintf("http://%s:%s@%s:%d", username, password, instance.LocalHost, port),
			Region:   instance.Region,
			Username: username,
			Password: password,
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

func (s *planService) GetPlan(ctx context.Context, planID uuid.UUID) (*domain.ProxyPlan, error) {
	return s.planRepo.GetByID(ctx, planID)
}

func (s *planService) GetPlansByCustomer(ctx context.Context, customerID string) ([]*domain.ProxyPlan, error) {
	return s.planRepo.GetByCustomerID(ctx, customerID)
}

func (s *planService) GetAllPlans(ctx context.Context) ([]*domain.ProxyPlan, error) {
	return s.planRepo.GetAll(ctx)
}

func (s *planService) UpdatePlanStatus(ctx context.Context, planID uuid.UUID, status string) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	plan.Status = status
	plan.UpdatedAt = time.Now()

	return s.planRepo.Update(ctx, plan)
}

func (s *planService) DeletePlan(ctx context.Context, planID uuid.UUID) error {
	// Get plan and instances
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	instances, err := s.instanceRepo.GetByPlanID(ctx, planID)
	if err != nil {
		return err
	}

	// Stop all instances
	for _, instance := range instances {
		if err := s.proxyService.StopInstance(ctx, instance.ID); err != nil {
			s.logger.Error("Failed to stop instance during plan deletion",
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err),
			)
		}
	}

	// Delete from repository
	return s.planRepo.Delete(ctx, planID)
}

func (s *planService) CheckExpiredPlans(ctx context.Context) ([]*domain.ProxyPlan, error) {
	return s.planRepo.GetExpired(ctx, time.Now())
}
