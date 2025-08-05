package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/service/provider"
)

type providerService struct {
	logger    *zap.Logger
	proxiesFo *provider.ProxiesFoProvider
	nettify   *provider.NettifyProvider
}

func NewProviderService(cfg *config.Config, logger *zap.Logger) ProviderService {
	return &providerService{
		logger:    logger,
		proxiesFo: provider.NewProxiesFoProvider(&cfg.Providers.ProxiesFo, logger),
		nettify:   provider.NewNettifyProvider(&cfg.Providers.Nettify, logger),
	}
}

func (s *providerService) CreateAccount(ctx context.Context, providerName string, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
	switch providerName {
	case domain.ProviderProxiesFo:
		return s.proxiesFo.CreateAccount(ctx, req)
	case domain.ProviderNettify:
		return s.nettify.CreateAccount(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

func (s *providerService) GetAccountInfo(ctx context.Context, providerName, accountID string) (*ProviderAccount, error) {
	switch providerName {
	case domain.ProviderProxiesFo:
		return s.proxiesFo.GetAccountInfo(ctx, accountID)
	case domain.ProviderNettify:
		return s.nettify.GetAccountInfo(ctx, accountID)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

func (s *providerService) DeleteAccount(ctx context.Context, providerName, accountID string) error {
	switch providerName {
	case domain.ProviderProxiesFo:
		return s.proxiesFo.DeleteAccount(ctx, accountID)
	case domain.ProviderNettify:
		return s.nettify.DeleteAccount(ctx, accountID)
	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}
}

func (s *providerService) TestConnection(ctx context.Context, providerName string, account *ProviderAccount) error {
	switch providerName {
	case domain.ProviderProxiesFo:
		return s.proxiesFo.TestConnection(ctx, account)
	case domain.ProviderNettify:
		return s.nettify.TestConnection(ctx, account)
	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}
}
