package json

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/repository"
)

// jsonPlanRepository implements PlanRepository using JSON file storage
type jsonPlanRepository struct {
	filePath string
	logger   *zap.Logger
	mu       sync.RWMutex
}

// jsonInstanceRepository implements InstanceRepository using JSON file storage
type jsonInstanceRepository struct {
	filePath string
	logger   *zap.Logger
	mu       sync.RWMutex
}

// Storage structures
type planStorage struct {
	Plans map[string]*domain.ProxyPlan `json:"plans"`
}

type instanceStorage struct {
	Instances map[string]*domain.ProxyInstance `json:"instances"`
}

// NewPlanRepository creates a new JSON-based plan repository
func NewPlanRepository(filePath string, logger *zap.Logger) repository.PlanRepository {
	return &jsonPlanRepository{
		filePath: filePath,
		logger:   logger,
	}
}

// NewInstanceRepository creates a new JSON-based instance repository
func NewInstanceRepository(filePath string, logger *zap.Logger) repository.InstanceRepository {
	instanceFilePath := filePath + "_instances"
	return &jsonInstanceRepository{
		filePath: instanceFilePath,
		logger:   logger,
	}
}

// Plan Repository Implementation

func (r *jsonPlanRepository) Create(ctx context.Context, plan *domain.ProxyPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadPlans()
	if err != nil {
		return fmt.Errorf("failed to load plans: %w", err)
	}

	storage.Plans[plan.ID.String()] = plan

	if err := r.savePlans(storage); err != nil {
		return fmt.Errorf("failed to save plans: %w", err)
	}

	r.logger.Info("Plan created", zap.String("plan_id", plan.ID.String()))
	return nil
}

func (r *jsonPlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	plan, exists := storage.Plans[id.String()]
	if !exists {
		return nil, fmt.Errorf("plan not found: %s", id.String())
	}

	return plan, nil
}

func (r *jsonPlanRepository) GetByCustomerID(ctx context.Context, customerID string) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var plans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		if plan.CustomerID == customerID {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func (r *jsonPlanRepository) GetAll(ctx context.Context) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var plans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		plans = append(plans, plan)
	}

	return plans, nil
}

func (r *jsonPlanRepository) Update(ctx context.Context, plan *domain.ProxyPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadPlans()
	if err != nil {
		return fmt.Errorf("failed to load plans: %w", err)
	}

	if _, exists := storage.Plans[plan.ID.String()]; !exists {
		return fmt.Errorf("plan not found: %s", plan.ID.String())
	}

	plan.UpdatedAt = time.Now()
	storage.Plans[plan.ID.String()] = plan

	if err := r.savePlans(storage); err != nil {
		return fmt.Errorf("failed to save plans: %w", err)
	}

	r.logger.Info("Plan updated", zap.String("plan_id", plan.ID.String()))
	return nil
}

func (r *jsonPlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadPlans()
	if err != nil {
		return fmt.Errorf("failed to load plans: %w", err)
	}

	if _, exists := storage.Plans[id.String()]; !exists {
		return fmt.Errorf("plan not found: %s", id.String())
	}

	delete(storage.Plans, id.String())

	if err := r.savePlans(storage); err != nil {
		return fmt.Errorf("failed to save plans: %w", err)
	}

	r.logger.Info("Plan deleted", zap.String("plan_id", id.String()))
	return nil
}

func (r *jsonPlanRepository) GetExpired(ctx context.Context, before time.Time) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var expiredPlans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		if plan.ExpiresAt.Before(before) {
			expiredPlans = append(expiredPlans, plan)
		}
	}

	return expiredPlans, nil
}

func (r *jsonPlanRepository) GetByStatus(ctx context.Context, status string) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var plans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		if plan.Status == status {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func (r *jsonPlanRepository) GetByProvider(ctx context.Context, provider string) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var plans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		if plan.Provider == provider {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func (r *jsonPlanRepository) GetByRegion(ctx context.Context, region string) ([]*domain.ProxyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	var plans []*domain.ProxyPlan
	for _, plan := range storage.Plans {
		if plan.Region == region {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func (r *jsonPlanRepository) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return 0, fmt.Errorf("failed to load plans: %w", err)
	}

	return len(storage.Plans), nil
}

func (r *jsonPlanRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadPlans()
	if err != nil {
		return 0, fmt.Errorf("failed to load plans: %w", err)
	}

	count := 0
	for _, plan := range storage.Plans {
		if plan.Status == status {
			count++
		}
	}

	return count, nil
}

// Instance Repository Implementation

func (r *jsonInstanceRepository) Create(ctx context.Context, instance *domain.ProxyInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	storage.Instances[instance.ID.String()] = instance

	if err := r.saveInstances(storage); err != nil {
		return fmt.Errorf("failed to save instances: %w", err)
	}

	r.logger.Info("Instance created", zap.String("instance_id", instance.ID.String()))
	return nil
}

func (r *jsonInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	instance, exists := storage.Instances[id.String()]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", id.String())
	}

	return instance, nil
}

func (r *jsonInstanceRepository) GetByPlanID(ctx context.Context, planID uuid.UUID) ([]*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	var instances []*domain.ProxyInstance
	for _, instance := range storage.Instances {
		if instance.PlanID == planID {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (r *jsonInstanceRepository) GetAll(ctx context.Context) ([]*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	var instances []*domain.ProxyInstance
	for _, instance := range storage.Instances {
		instances = append(instances, instance)
	}

	return instances, nil
}

func (r *jsonInstanceRepository) Update(ctx context.Context, instance *domain.ProxyInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	if _, exists := storage.Instances[instance.ID.String()]; !exists {
		return fmt.Errorf("instance not found: %s", instance.ID.String())
	}

	instance.UpdatedAt = time.Now()
	storage.Instances[instance.ID.String()] = instance

	if err := r.saveInstances(storage); err != nil {
		return fmt.Errorf("failed to save instances: %w", err)
	}

	r.logger.Info("Instance updated", zap.String("instance_id", instance.ID.String()))
	return nil
}

func (r *jsonInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storage, err := r.loadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	if _, exists := storage.Instances[id.String()]; !exists {
		return fmt.Errorf("instance not found: %s", id.String())
	}

	delete(storage.Instances, id.String())

	if err := r.saveInstances(storage); err != nil {
		return fmt.Errorf("failed to save instances: %w", err)
	}

	r.logger.Info("Instance deleted", zap.String("instance_id", id.String()))
	return nil
}

func (r *jsonInstanceRepository) GetByStatus(ctx context.Context, status string) ([]*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	var instances []*domain.ProxyInstance
	for _, instance := range storage.Instances {
		if instance.Status == status {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (r *jsonInstanceRepository) GetByPort(ctx context.Context, port int) (*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	for _, instance := range storage.Instances {
		if instance.LocalPort == port {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("instance not found for port: %d", port)
}

func (r *jsonInstanceRepository) GetByPlanTypeKey(ctx context.Context, planTypeKey string) ([]*domain.ProxyInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	var instances []*domain.ProxyInstance
	for _, instance := range storage.Instances {
		if instance.PlanTypeKey == planTypeKey {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (r *jsonInstanceRepository) GetRunning(ctx context.Context) ([]*domain.ProxyInstance, error) {
	return r.GetByStatus(ctx, domain.InstanceStatusRunning)
}

func (r *jsonInstanceRepository) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return 0, fmt.Errorf("failed to load instances: %w", err)
	}

	return len(storage.Instances), nil
}

func (r *jsonInstanceRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return 0, fmt.Errorf("failed to load instances: %w", err)
	}

	count := 0
	for _, instance := range storage.Instances {
		if instance.Status == status {
			count++
		}
	}

	return count, nil
}

func (r *jsonInstanceRepository) GetPortsInUse(ctx context.Context) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	storage, err := r.loadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}

	var ports []int
	for _, instance := range storage.Instances {
		ports = append(ports, instance.LocalPort)
	}

	return ports, nil
}

// Helper methods for plan repository

func (r *jsonPlanRepository) loadPlans() (*planStorage, error) {
	storage := &planStorage{
		Plans: make(map[string]*domain.ProxyPlan),
	}

	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		return storage, nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return storage, nil
	}

	if err := json.Unmarshal(data, storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return storage, nil
}

func (r *jsonPlanRepository) savePlans(storage *planStorage) error {
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Helper methods for instance repository

func (r *jsonInstanceRepository) loadInstances() (*instanceStorage, error) {
	storage := &instanceStorage{
		Instances: make(map[string]*domain.ProxyInstance),
	}

	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		return storage, nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return storage, nil
	}

	if err := json.Unmarshal(data, storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return storage, nil
}

func (r *jsonInstanceRepository) saveInstances(storage *instanceStorage) error {
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
