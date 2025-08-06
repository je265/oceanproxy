// internal/service/port_manager.go - FIXED (remove duplicate PoolStats)
package service

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
)

// PortManager manages port pools for different plan types
type PortManager struct {
	mu        sync.RWMutex
	logger    *zap.Logger
	pools     map[string]*domain.PortPool // plan_type_key -> port_pool
	planTypes map[string]*domain.PlanTypeConfig
}

// NewPortManager creates a new port manager
func NewPortManager(logger *zap.Logger, planTypes map[string]*domain.PlanTypeConfig) *PortManager {
	pm := &PortManager{
		logger:    logger,
		pools:     make(map[string]*domain.PortPool),
		planTypes: planTypes,
	}

	// Initialize port pools for each plan type
	for key, planType := range planTypes {
		pool := domain.NewPortPool(key, planType.LocalPortRange)
		pm.pools[key] = pool

		logger.Info("Initialized port pool",
			zap.String("plan_type", key),
			zap.Int("start_port", planType.LocalPortRange.Start),
			zap.Int("end_port", planType.LocalPortRange.End),
			zap.Int("pool_size", planType.LocalPortRange.Size()),
		)
	}

	return pm
}

// AllocatePort allocates a port for a specific plan type
func (pm *PortManager) AllocatePort(ctx context.Context, planTypeKey, planID string) (int, error) {
	pm.mu.RLock()
	pool, exists := pm.pools[planTypeKey]
	pm.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("plan type %s not found", planTypeKey)
	}

	port, err := pool.AllocatePort(planID)
	if err != nil {
		pm.logger.Error("Failed to allocate port",
			zap.String("plan_type", planTypeKey),
			zap.String("plan_id", planID),
			zap.Error(err),
		)
		return 0, err
	}

	pm.logger.Info("Allocated port",
		zap.String("plan_type", planTypeKey),
		zap.String("plan_id", planID),
		zap.Int("port", port),
	)

	return port, nil
}

// ReleasePort releases a port back to its pool
func (pm *PortManager) ReleasePort(ctx context.Context, planTypeKey string, port int) error {
	pm.mu.RLock()
	pool, exists := pm.pools[planTypeKey]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plan type %s not found", planTypeKey)
	}

	if err := pool.ReleasePort(port); err != nil {
		pm.logger.Error("Failed to release port",
			zap.String("plan_type", planTypeKey),
			zap.Int("port", port),
			zap.Error(err),
		)
		return err
	}

	pm.logger.Info("Released port",
		zap.String("plan_type", planTypeKey),
		zap.Int("port", port),
	)

	return nil
}

// GetPlanTypeConfig returns the configuration for a plan type
func (pm *PortManager) GetPlanTypeConfig(planTypeKey string) (*domain.PlanTypeConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	config, exists := pm.planTypes[planTypeKey]
	if !exists {
		return nil, fmt.Errorf("plan type %s not found", planTypeKey)
	}

	return config, nil
}

// GetAvailablePlanTypes returns all available plan types
func (pm *PortManager) GetAvailablePlanTypes() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var planTypes []string
	for key := range pm.planTypes {
		planTypes = append(planTypes, key)
	}

	return planTypes
}

// GetPoolStats returns statistics for all port pools
func (pm *PortManager) GetPoolStats() map[string]PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for key, pool := range pm.pools {
		stats[key] = PoolStats{
			PlanType:       key,
			TotalPorts:     pm.planTypes[key].LocalPortRange.Size(),
			AllocatedPorts: pool.GetAllocatedCount(),
			AvailablePorts: pool.GetAvailableCount(),
		}
	}

	return stats
}

// FindPlanTypeByProviderAndRegion finds plan types matching provider and region
func (pm *PortManager) FindPlanTypeByProviderAndRegion(provider, region, planType string) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := fmt.Sprintf("%s_%s_%s", provider, region, planType)
	if _, exists := pm.planTypes[key]; exists {
		return key, nil
	}

	return "", fmt.Errorf("plan type not found: provider=%s, region=%s, type=%s", provider, region, planType)
}
