// internal/repository/interfaces.go - Complete repository interfaces
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/je265/oceanproxy/internal/domain"
)

// PlanRepository defines the interface for plan data persistence
type PlanRepository interface {
	// Create creates a new proxy plan
	Create(ctx context.Context, plan *domain.ProxyPlan) error

	// GetByID retrieves a plan by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProxyPlan, error)

	// GetByCustomerID retrieves all plans for a customer
	GetByCustomerID(ctx context.Context, customerID string) ([]*domain.ProxyPlan, error)

	// GetAll retrieves all plans
	GetAll(ctx context.Context) ([]*domain.ProxyPlan, error)

	// Update updates an existing plan
	Update(ctx context.Context, plan *domain.ProxyPlan) error

	// Delete deletes a plan by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// GetExpired retrieves all plans that have expired before the given time
	GetExpired(ctx context.Context, before time.Time) ([]*domain.ProxyPlan, error)

	// GetByStatus retrieves all plans with a specific status
	GetByStatus(ctx context.Context, status string) ([]*domain.ProxyPlan, error)

	// GetByProvider retrieves all plans for a specific provider
	GetByProvider(ctx context.Context, provider string) ([]*domain.ProxyPlan, error)

	// GetByRegion retrieves all plans for a specific region
	GetByRegion(ctx context.Context, region string) ([]*domain.ProxyPlan, error)

	// Count returns the total number of plans
	Count(ctx context.Context) (int, error)

	// CountByStatus returns the number of plans with a specific status
	CountByStatus(ctx context.Context, status string) (int, error)
}

// InstanceRepository defines the interface for proxy instance data persistence
type InstanceRepository interface {
	// Create creates a new proxy instance
	Create(ctx context.Context, instance *domain.ProxyInstance) error

	// GetByID retrieves an instance by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProxyInstance, error)

	// GetByPlanID retrieves all instances for a plan
	GetByPlanID(ctx context.Context, planID uuid.UUID) ([]*domain.ProxyInstance, error)

	// GetAll retrieves all instances
	GetAll(ctx context.Context) ([]*domain.ProxyInstance, error)

	// Update updates an existing instance
	Update(ctx context.Context, instance *domain.ProxyInstance) error

	// Delete deletes an instance by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByStatus retrieves all instances with a specific status
	GetByStatus(ctx context.Context, status string) ([]*domain.ProxyInstance, error)

	// GetByPort retrieves an instance by its local port
	GetByPort(ctx context.Context, port int) (*domain.ProxyInstance, error)

	// GetByPlanTypeKey retrieves all instances for a specific plan type
	GetByPlanTypeKey(ctx context.Context, planTypeKey string) ([]*domain.ProxyInstance, error)

	// GetRunning retrieves all running instances
	GetRunning(ctx context.Context) ([]*domain.ProxyInstance, error)

	// Count returns the total number of instances
	Count(ctx context.Context) (int, error)

	// CountByStatus returns the number of instances with a specific status
	CountByStatus(ctx context.Context, status string) (int, error)

	// GetPortsInUse returns all ports currently in use
	GetPortsInUse(ctx context.Context) ([]int, error)
}

// UserRepository defines the interface for user data persistence (future use)
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// GetAll retrieves all users
	GetAll(ctx context.Context) ([]*domain.User, error)

	// Count returns the total number of users
	Count(ctx context.Context) (int, error)
}

// StatsRepository defines the interface for statistics and metrics
type StatsRepository interface {
	// RecordRequest records a proxy request
	RecordRequest(ctx context.Context, instanceID uuid.UUID, bytesIn, bytesOut int64) error

	// GetInstanceStats retrieves statistics for a specific instance
	GetInstanceStats(ctx context.Context, instanceID uuid.UUID, from, to time.Time) (*InstanceStats, error)

	// GetPlanStats retrieves statistics for a specific plan
	GetPlanStats(ctx context.Context, planID uuid.UUID, from, to time.Time) (*PlanStats, error)

	// GetOverallStats retrieves overall system statistics
	GetOverallStats(ctx context.Context, from, to time.Time) (*OverallStats, error)
}

// Statistics data structures
type InstanceStats struct {
	InstanceID    uuid.UUID     `json:"instance_id"`
	TotalRequests int64         `json:"total_requests"`
	BytesIn       int64         `json:"bytes_in"`
	BytesOut      int64         `json:"bytes_out"`
	Uptime        time.Duration `json:"uptime"`
	LastActivity  time.Time     `json:"last_activity"`
}

type PlanStats struct {
	PlanID          uuid.UUID `json:"plan_id"`
	TotalRequests   int64     `json:"total_requests"`
	BytesIn         int64     `json:"bytes_in"`
	BytesOut        int64     `json:"bytes_out"`
	ActiveInstances int       `json:"active_instances"`
	TotalInstances  int       `json:"total_instances"`
}

type OverallStats struct {
	TotalPlans       int            `json:"total_plans"`
	ActivePlans      int            `json:"active_plans"`
	TotalInstances   int            `json:"total_instances"`
	RunningInstances int            `json:"running_instances"`
	TotalRequests    int64          `json:"total_requests"`
	BytesIn          int64          `json:"bytes_in"`
	BytesOut         int64          `json:"bytes_out"`
	ProvidersUsed    map[string]int `json:"providers_used"`
	RegionsUsed      map[string]int `json:"regions_used"`
}
