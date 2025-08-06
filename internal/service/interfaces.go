// internal/service/interfaces.go - FIXED
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/je265/oceanproxy/internal/domain"
)

// PlanService defines the interface for plan management
type PlanService interface {
	CreatePlan(ctx context.Context, req *domain.CreatePlanRequest) (*domain.CreatePlanResponse, error)
	GetPlan(ctx context.Context, planID uuid.UUID) (*domain.ProxyPlan, error)
	GetPlansByCustomer(ctx context.Context, customerID string) ([]*domain.ProxyPlan, error)
	GetAllPlans(ctx context.Context) ([]*domain.ProxyPlan, error)
	UpdatePlanStatus(ctx context.Context, planID uuid.UUID, status string) error
	DeletePlan(ctx context.Context, planID uuid.UUID) error
	CheckExpiredPlans(ctx context.Context) ([]*domain.ProxyPlan, error)
}

// ProxyService defines the interface for proxy instance management
type ProxyService interface {
	StartInstance(ctx context.Context, instance *domain.ProxyInstance) error
	StopInstance(ctx context.Context, instanceID uuid.UUID) error
	RestartInstance(ctx context.Context, instanceID uuid.UUID) error
	GetInstanceStatus(ctx context.Context, instanceID uuid.UUID) (string, error)
	GetRunningInstances(ctx context.Context) ([]*domain.ProxyInstance, error)
	GetInstance(ctx context.Context, instanceID uuid.UUID) (*domain.ProxyInstance, error)
	GetInstancesByPlan(ctx context.Context, planID uuid.UUID) ([]*domain.ProxyInstance, error)
	HealthCheck(ctx context.Context, instanceID uuid.UUID) error
}

// ProviderService defines the interface for upstream provider integration
type ProviderService interface {
	CreateAccount(ctx context.Context, provider string, req *domain.CreatePlanRequest) (*ProviderAccount, error)
	GetAccountInfo(ctx context.Context, provider, accountID string) (*ProviderAccount, error)
	DeleteAccount(ctx context.Context, provider, accountID string) error
	TestConnection(ctx context.Context, provider string, account *ProviderAccount) error
}

// ProviderAccount represents an account with an upstream provider
type ProviderAccount struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Region   string `json:"region"`
}

// PoolStats represents statistics for a port pool
type PoolStats struct {
	PlanType       string `json:"plan_type"`
	TotalPorts     int    `json:"total_ports"`
	AllocatedPorts int    `json:"allocated_ports"`
	AvailablePorts int    `json:"available_ports"`
}
