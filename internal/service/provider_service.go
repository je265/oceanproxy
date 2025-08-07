package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/service/provider"
	"github.com/je265/oceanproxy/pkg/config"
)

type providerService struct {
	logger          *zap.Logger
	providerManager *provider.Manager
}

func NewProviderService(cfg *config.Config, logger *zap.Logger) ProviderService {
	// Create provider manager
	manager := provider.NewManager()

	// Register providers
	proxiesFoProvider := provider.NewProxiesFoProvider(&cfg.Providers.ProxiesFo, logger)
	nettifyProvider := provider.NewNettifyProvider(&cfg.Providers.Nettify, logger)

	manager.RegisterProvider(domain.ProviderProxiesFo, proxiesFoProvider)
	manager.RegisterProvider(domain.ProviderNettify, nettifyProvider)

	return &providerService{
		logger:          logger,
		providerManager: manager,
	}
}

func (s *providerService) CreateAccount(ctx context.Context, providerName string, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
	// Use the provider manager to create account
	account, err := s.providerManager.CreateAccount(ctx, providerName, req)
	if err != nil {
		return nil, err
	}

	// Convert provider.ProviderAccount to service.ProviderAccount
	return &ProviderAccount{
		ID:       account.ID,
		Username: account.Username,
		Password: account.Password,
		Host:     account.Host,
		Port:     account.Port,
		Region:   account.Region,
	}, nil
}

func (s *providerService) GetAccountInfo(ctx context.Context, providerName, accountID string) (*ProviderAccount, error) {
	// Use the provider manager to get account info
	account, err := s.providerManager.GetAccountInfo(ctx, providerName, accountID)
	if err != nil {
		return nil, err
	}

	// Convert provider.ProviderAccount to service.ProviderAccount
	return &ProviderAccount{
		ID:       account.ID,
		Username: account.Username,
		Password: account.Password,
		Host:     account.Host,
		Port:     account.Port,
		Region:   account.Region,
	}, nil
}

func (s *providerService) DeleteAccount(ctx context.Context, providerName, accountID string) error {
	return s.providerManager.DeleteAccount(ctx, providerName, accountID)
}

func (s *providerService) TestConnection(ctx context.Context, providerName string, account *ProviderAccount) error {
	// Convert service.ProviderAccount to provider.ProviderAccount
	providerAccount := &provider.ProviderAccount{
		ID:       account.ID,
		Username: account.Username,
		Password: account.Password,
		Host:     account.Host,
		Port:     account.Port,
		Region:   account.Region,
	}

	return s.providerManager.TestConnection(ctx, providerName, providerAccount)
}
