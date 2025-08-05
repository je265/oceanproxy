package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProxyPlan represents a customer's proxy plan
type ProxyPlan struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CustomerID string    `json:"customer_id" db:"customer_id"`
	PlanType   string    `json:"plan_type" db:"plan_type"`
	Provider   string    `json:"provider" db:"provider"`
	Username   string    `json:"username" db:"username"`
	Password   string    `json:"password" db:"password"`
	Status     string    `json:"status" db:"status"`
	Bandwidth  int       `json:"bandwidth" db:"bandwidth"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`

	// Proxy instances for this plan
	Instances []ProxyInstance `json:"instances,omitempty"`
}

// ProxyInstance represents a single proxy instance
type ProxyInstance struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PlanID    uuid.UUID `json:"plan_id" db:"plan_id"`
	Region    string    `json:"region" db:"region"`
	LocalPort int       `json:"local_port" db:"local_port"`
	AuthHost  string    `json:"auth_host" db:"auth_host"`
	AuthPort  int       `json:"auth_port" db:"auth_port"`
	LocalHost string    `json:"local_host" db:"local_host"`
	Status    string    `json:"status" db:"status"`
	ProcessID int       `json:"process_id,omitempty" db:"process_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ProxyEndpoint represents a customer-facing proxy endpoint
type ProxyEndpoint struct {
	URL      string `json:"url"`
	Region   string `json:"region"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreatePlanRequest represents a request to create a new proxy plan
type CreatePlanRequest struct {
	CustomerID string `json:"customer_id" validate:"required"`
	PlanType   string `json:"plan_type" validate:"required,oneof=residential datacenter isp mobile"`
	Provider   string `json:"provider" validate:"required,oneof=proxies_fo nettify"`
	Username   string `json:"username" validate:"required,min=3,max=50"`
	Password   string `json:"password" validate:"required,min=6,max=100"`
	Bandwidth  int    `json:"bandwidth" validate:"required,min=1,max=1000"`
	Duration   int    `json:"duration,omitempty" validate:"min=1,max=365"` // days
}

// CreatePlanResponse represents the response after creating a plan
type CreatePlanResponse struct {
	Success   bool            `json:"success"`
	PlanID    uuid.UUID       `json:"plan_id"`
	Username  string          `json:"username"`
	Password  string          `json:"password"`
	ExpiresAt time.Time       `json:"expires_at"`
	Proxies   []ProxyEndpoint `json:"proxies"`
}

// Plan status constants
const (
	PlanStatusActive    = "active"
	PlanStatusExpired   = "expired"
	PlanStatusSuspended = "suspended"
	PlanStatusCreating  = "creating"
	PlanStatusFailed    = "failed"
)

// Instance status constants
const (
	InstanceStatusRunning  = "running"
	InstanceStatusStopped  = "stopped"
	InstanceStatusFailed   = "failed"
	InstanceStatusStarting = "starting"
)

// Provider constants
const (
	ProviderProxiesFo = "proxies_fo"
	ProviderNettify   = "nettify"
)

// Plan type constants
const (
	PlanTypeResidential = "residential"
	PlanTypeDatacenter  = "datacenter"
	PlanTypeISP         = "isp"
	PlanTypeMobile      = "mobile"
)

// Region constants
const (
	RegionUSA   = "usa"
	RegionEU    = "eu"
	RegionAlpha = "alpha"
	RegionBeta  = "beta"
)

// Add to existing domain/proxy.go

// CreatePlanRequest represents a request to create a new proxy plan
type CreatePlanRequest struct {
	CustomerID string `json:"customer_id" validate:"required"`
	PlanType   string `json:"plan_type" validate:"required,oneof=residential datacenter isp mobile unlimited"`
	Provider   string `json:"provider" validate:"required,oneof=proxies_fo nettify"`
	Region     string `json:"region" validate:"required,oneof=usa eu alpha beta"`
	Username   string `json:"username" validate:"required,min=3,max=50"`
	Password   string `json:"password" validate:"required,min=6,max=100"`
	Bandwidth  int    `json:"bandwidth" validate:"min=1,max=1000"`         // GB
	Duration   int    `json:"duration,omitempty" validate:"min=1,max=365"` // days
}

// ProxyInstance represents a single proxy instance
type ProxyInstance struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PlanID      uuid.UUID `json:"plan_id" db:"plan_id"`
	PlanTypeKey string    `json:"plan_type_key" db:"plan_type_key"`
	LocalPort   int       `json:"local_port" db:"local_port"`
	AuthHost    string    `json:"auth_host" db:"auth_host"`
	AuthPort    int       `json:"auth_port" db:"auth_port"`
	Status      string    `json:"status" db:"status"`
	ProcessID   int       `json:"process_id,omitempty" db:"process_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ProxyPlan represents a customer's proxy plan
type ProxyPlan struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CustomerID  string    `json:"customer_id" db:"customer_id"`
	PlanType    string    `json:"plan_type" db:"plan_type"`
	Provider    string    `json:"provider" db:"provider"`
	Region      string    `json:"region" db:"region"`
	PlanTypeKey string    `json:"plan_type_key" db:"plan_type_key"`
	Username    string    `json:"username" db:"username"`
	Password    string    `json:"password" db:"password"`
	Status      string    `json:"status" db:"status"`
	Bandwidth   int       `json:"bandwidth" db:"bandwidth"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Associated instances
	Instances []*ProxyInstance `json:"instances,omitempty"`
}
