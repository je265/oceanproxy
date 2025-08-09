// internal/service/provider/interfaces.go - NEW FILE
package provider

import (
	"context"

	"github.com/je265/oceanproxy/internal/domain"
)

// Provider represents a generic proxy provider
type Provider interface {
	CreateAccount(ctx context.Context, req *domain.CreatePlanRequest) (*ProviderAccount, error)
	GetAccountInfo(ctx context.Context, accountID string) (*ProviderAccount, error)
	DeleteAccount(ctx context.Context, accountID string) error
	TestConnection(ctx context.Context, account *ProviderAccount) error
}

// ProviderAccount represents an account with an upstream provider
type ProviderAccount struct {
	ID       string `json:"id"`
    CustomerID string `json:"customer_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Region   string `json:"region"`
}

// Manager handles multiple providers
type Manager struct {
	providers map[string]Provider
}

// NewManager creates a new provider manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registers a provider with the manager
func (m *Manager) RegisterProvider(name string, provider Provider) {
	m.providers[name] = provider
}

// CreateAccount creates an account with the specified provider
func (m *Manager) CreateAccount(ctx context.Context, providerName string, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
	provider, exists := m.providers[providerName]
	if !exists {
		return nil, ErrProviderNotFound{Provider: providerName}
	}

	return provider.CreateAccount(ctx, req)
}

// GetAccountInfo gets account information from the specified provider
func (m *Manager) GetAccountInfo(ctx context.Context, providerName, accountID string) (*ProviderAccount, error) {
	provider, exists := m.providers[providerName]
	if !exists {
		return nil, ErrProviderNotFound{Provider: providerName}
	}

	return provider.GetAccountInfo(ctx, accountID)
}

// DeleteAccount deletes an account from the specified provider
func (m *Manager) DeleteAccount(ctx context.Context, providerName, accountID string) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return ErrProviderNotFound{Provider: providerName}
	}

	return provider.DeleteAccount(ctx, accountID)
}

// TestConnection tests connectivity to the specified provider
func (m *Manager) TestConnection(ctx context.Context, providerName string, account *ProviderAccount) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return ErrProviderNotFound{Provider: providerName}
	}

	return provider.TestConnection(ctx, account)
}

// Custom error types
type ErrProviderNotFound struct {
	Provider string
}

func (e ErrProviderNotFound) Error() string {
	return "provider not found: " + e.Provider
}
