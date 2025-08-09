// internal/service/plan.go - FIXED (remove unused variables)
package service

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/je265/oceanproxy/internal/domain"
    "github.com/je265/oceanproxy/internal/repository"
    "github.com/je265/oceanproxy/pkg/config"
)

type planService struct {
	cfg             *config.Config
	logger          *zap.Logger
	planRepo        repository.PlanRepository
	instanceRepo    repository.InstanceRepository
	providerService ProviderService
	proxyService    ProxyService
	portManager     *PortManager
	nginxManager    *NginxManager
	regions         map[string]*domain.Region
}

func NewPlanService(
	cfg *config.Config,
	logger *zap.Logger,
	planRepo repository.PlanRepository,
	instanceRepo repository.InstanceRepository,
	providerService ProviderService,
	proxyService ProxyService,
	portManager *PortManager,
	nginxManager *NginxManager,
	regions map[string]*domain.Region,
) PlanService {
	return &planService{
		cfg:             cfg,
		logger:          logger,
		planRepo:        planRepo,
		instanceRepo:    instanceRepo,
		providerService: providerService,
		proxyService:    proxyService,
		portManager:     portManager,
		nginxManager:    nginxManager,
		regions:         regions,
	}
}

func (s *planService) CreatePlan(ctx context.Context, req *domain.CreatePlanRequest) (*domain.CreatePlanResponse, error) {
	s.logger.Info("Creating new proxy plan",
		zap.String("customer_id", req.CustomerID),
		zap.String("plan_type", req.PlanType),
		zap.String("provider", req.Provider),
		zap.String("region", req.Region),
	)

	// Find the appropriate plan type configuration
	planTypeKey, err := s.portManager.FindPlanTypeByProviderAndRegion(req.Provider, req.Region, req.PlanType)
	if err != nil {
		return nil, fmt.Errorf("unsupported plan configuration: %w", err)
	}

	// Get plan type config for upstream details
	_, err = s.portManager.GetPlanTypeConfig(planTypeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan type config: %w", err)
	}

    // Create plan record (username/password may be overridden by provider)
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
		plan.ExpiresAt = time.Now().AddDate(0, 0, 30) // Default to 30 days
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

    // Use provider-generated credentials and customer association if provided
    if providerAccount != nil {
        if providerAccount.Username != "" {
            plan.Username = providerAccount.Username
        }
        if providerAccount.Password != "" {
            plan.Password = providerAccount.Password
        }
        if providerAccount.CustomerID != "" {
            plan.CustomerID = providerAccount.CustomerID
        }
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
		AuthHost:    providerAccount.Host,
		AuthPort:    providerAccount.Port,
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

    // Build response with customer-facing endpoint mapping rules
    host, port, displayRegion, err := s.resolveEndpointHostPort(req.Provider, req.PlanType, req.Region)
    if err != nil {
        return nil, err
    }

    endpointURL := fmt.Sprintf("http://%s:%s@%s:%d", plan.Username, plan.Password, host, port)

    response := &domain.CreatePlanResponse{
		Success:   true,
		PlanID:    plan.ID,
		Username:  plan.Username,
		Password:  plan.Password,
		ExpiresAt: plan.ExpiresAt,
		Proxies: []domain.ProxyEndpoint{
			{
                URL:      endpointURL,
                Region:   displayRegion,
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

// resolveEndpointHostPort determines the customer-facing host, port, and region label
// based on provider, plan type, and requested region.
func (s *planService) resolveEndpointHostPort(provider, planType, reqRegion string) (string, int, string, error) {
    switch provider {
    case domain.ProviderProxiesFo:
        switch planType {
        case domain.PlanTypeResidential:
            // usa -> usa.oceanproxy.io, eu -> eu.oceanproxy.io
            region := s.regions[reqRegion]
            if region == nil {
                return "", 0, "", fmt.Errorf("region %s not found", reqRegion)
            }
            return region.GetFullDomain(), region.OutboundPort, region.Name, nil
        case domain.PlanTypeDatacenter:
            // datacenter.oceanproxy.io with port from requested region
            region := s.regions[reqRegion]
            if region == nil {
                return "", 0, "", fmt.Errorf("region %s not found", reqRegion)
            }
            return "datacenter.oceanproxy.io", region.OutboundPort, "datacenter", nil
        case domain.PlanTypeISP:
            // isp.oceanproxy.io with port from requested region
            region := s.regions[reqRegion]
            if region == nil {
                return "", 0, "", fmt.Errorf("region %s not found", reqRegion)
            }
            return "isp.oceanproxy.io", region.OutboundPort, "isp", nil
        default:
            // fallback to requested region
            region := s.regions[reqRegion]
            if region == nil {
                return "", 0, "", fmt.Errorf("region %s not found", reqRegion)
            }
            return region.GetFullDomain(), region.OutboundPort, region.Name, nil
        }
    case domain.ProviderNettify:
        switch planType {
        case domain.PlanTypeResidential:
            // alpha.oceanproxy.io (use alpha port)
            alpha := s.regions[domain.RegionAlpha]
            if alpha == nil {
                return "", 0, "", fmt.Errorf("region %s not found", domain.RegionAlpha)
            }
            return "alpha.oceanproxy.io", alpha.OutboundPort, "alpha", nil
        case domain.PlanTypeDatacenter:
            // beta.oceanproxy.io (use beta port)
            beta := s.regions[domain.RegionBeta]
            if beta == nil {
                return "", 0, "", fmt.Errorf("region %s not found", domain.RegionBeta)
            }
            return "beta.oceanproxy.io", beta.OutboundPort, "beta", nil
        case domain.PlanTypeMobile:
            // mobile.oceanproxy.io (use alpha port as base if mobile not defined)
            // Try a region named "mobile" if present; otherwise fall back to alpha's port
            if mobile := s.regions["mobile"]; mobile != nil {
                return "mobile.oceanproxy.io", mobile.OutboundPort, "mobile", nil
            }
            alpha := s.regions[domain.RegionAlpha]
            if alpha == nil {
                return "", 0, "", fmt.Errorf("region %s not found", domain.RegionAlpha)
            }
            return "mobile.oceanproxy.io", alpha.OutboundPort, "mobile", nil
        case domain.PlanTypeUnlimited:
            // unlim.oceanproxy.io (use alpha port as base if unlim not defined)
            if unlim := s.regions["unlim"]; unlim != nil {
                return "unlim.oceanproxy.io", unlim.OutboundPort, "unlim", nil
            }
            alpha := s.regions[domain.RegionAlpha]
            if alpha == nil {
                return "", 0, "", fmt.Errorf("region %s not found", domain.RegionAlpha)
            }
            return "unlim.oceanproxy.io", alpha.OutboundPort, "unlim", nil
        default:
            alpha := s.regions[domain.RegionAlpha]
            if alpha == nil {
                return "", 0, "", fmt.Errorf("region %s not found", domain.RegionAlpha)
            }
            return alpha.GetFullDomain(), alpha.OutboundPort, alpha.Name, nil
        }
    }

    // Unknown provider; default to requested region
    region := s.regions[reqRegion]
    if region == nil {
        return "", 0, "", fmt.Errorf("region %s not found", reqRegion)
    }
    return region.GetFullDomain(), region.OutboundPort, region.Name, nil
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
	updatedPlan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	updatedPlan.Status = status
	updatedPlan.UpdatedAt = time.Now()

	return s.planRepo.Update(ctx, updatedPlan)
}

func (s *planService) DeletePlan(ctx context.Context, planID uuid.UUID) error {
	// Get plan and instances
	planToDelete, err := s.planRepo.GetByID(ctx, planID)
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

		// Release port
		if err := s.portManager.ReleasePort(ctx, instance.PlanTypeKey, instance.LocalPort); err != nil {
			s.logger.Error("Failed to release port during plan deletion",
				zap.String("instance_id", instance.ID.String()),
				zap.Int("port", instance.LocalPort),
				zap.Error(err),
			)
		}

		// Remove from nginx upstream
		if err := s.nginxManager.RemoveFromUpstream(ctx, instance.PlanTypeKey, instance.LocalPort); err != nil {
			s.logger.Error("Failed to remove from nginx upstream during plan deletion",
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err),
			)
		}

		// Delete instance
		if err := s.instanceRepo.Delete(ctx, instance.ID); err != nil {
			s.logger.Error("Failed to delete instance during plan deletion",
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("Plan deletion completed",
		zap.String("plan_id", planToDelete.ID.String()),
		zap.String("customer_id", planToDelete.CustomerID),
	)

	// Delete plan from repository
	return s.planRepo.Delete(ctx, planID)
}

func (s *planService) CheckExpiredPlans(ctx context.Context) ([]*domain.ProxyPlan, error) {
	return s.planRepo.GetExpired(ctx, time.Now())
}
