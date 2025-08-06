// internal/service/provider/proxiesfo.go - FIXED
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
)

type ProxiesFoProvider struct {
	cfg    *config.ProxiesFoConfig
	logger *zap.Logger
	client *http.Client
}

func NewProxiesFoProvider(cfg *config.ProxiesFoConfig, logger *zap.Logger) *ProxiesFoProvider {
	return &ProxiesFoProvider{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// ProxiesFoResponse represents the API response from Proxies.fo
type ProxiesFoResponse struct {
	Success bool          `json:"Success"`
	Data    ProxiesFoData `json:"Data"`
	Error   string        `json:"Error"`
}

type ProxiesFoData struct {
	ID           string  `json:"ID"`
	AuthUsername string  `json:"AuthUsername"`
	AuthPassword string  `json:"AuthPassword"`
	AuthHostname string  `json:"AuthHostname"`
	AuthPort     float64 `json:"AuthPort"`
	EndsDate     float64 `json:"EndsDate"`
}

func (p *ProxiesFoProvider) CreateAccount(ctx context.Context, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
	p.logger.Info("Creating Proxies.fo account",
		zap.String("customer_id", req.CustomerID),
		zap.String("plan_type", req.PlanType),
		zap.String("region", req.Region),
	)

	// Map plan types to Proxies.fo reseller IDs
	resellerMap := map[string]string{
		"residential": "7c9ea873-63f9-4013-9147-3807cc6f0553",
		"isp":         "3471aa35-7922-488a-a7a9-b92a5510080e",
		"datacenter":  "b3fd0f3c-693d-4ec5-b49f-c77feaab0b72",
	}

	resellerID, ok := resellerMap[req.PlanType]
	if !ok {
		return nil, fmt.Errorf("unsupported plan type: %s", req.PlanType)
	}

	// Prepare form data
	formData := url.Values{}
	formData.Set("reseller", resellerID)
	formData.Set("username", req.Username)
	formData.Set("password", req.Password)

	// Set plan-specific parameters
	if req.PlanType == "datacenter" {
		duration := req.Duration
		if duration == 0 {
			duration = 1 // Default to 1 day
		}
		formData.Set("duration", strconv.Itoa(duration))
		formData.Set("threads", "500") // Default thread limit
	} else {
		// Residential/ISP plans
		formData.Set("duration", "180") // 180 days
		bandwidth := req.Bandwidth
		if bandwidth == 0 {
			bandwidth = 1 // Default to 1GB
		}
		formData.Set("bandwidth", strconv.Itoa(bandwidth))
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/api/plans/new", p.cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-Api-Auth", p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p.logger.Debug("Sending request to Proxies.fo API",
		zap.String("url", apiURL),
		zap.String("form_data", formData.Encode()),
	)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var result ProxiesFoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.logger.Debug("Received response from Proxies.fo API",
		zap.Any("response", result),
	)

	if !result.Success {
		return nil, fmt.Errorf("Proxies.fo API error: %s", result.Error)
	}

	// Determine the correct upstream host based on response
	upstreamHost := result.Data.AuthHostname
	if upstreamHost == "" {
		// Fallback based on plan type and region
		if req.PlanType == "datacenter" {
			upstreamHost = "dcp.proxies.fo"
		} else {
			// Residential/ISP
			if req.Region == "eu" {
				upstreamHost = "pr-eu.proxies.fo"
			} else {
				upstreamHost = "pr-us.proxies.fo"
			}
		}
	}

	account := &ProviderAccount{
		ID:       result.Data.ID,
		Username: result.Data.AuthUsername,
		Password: result.Data.AuthPassword,
		Host:     upstreamHost,
		Port:     int(result.Data.AuthPort),
		Region:   req.Region,
	}

	p.logger.Info("Successfully created Proxies.fo account",
		zap.String("account_id", account.ID),
		zap.String("username", account.Username),
		zap.String("host", account.Host),
		zap.Int("port", account.Port),
	)

	return account, nil
}

func (p *ProxiesFoProvider) GetAccountInfo(ctx context.Context, accountID string) (*ProviderAccount, error) {
	// Implementation for getting account info
	// This would typically involve another API call to get account details
	return nil, fmt.Errorf("GetAccountInfo not implemented for Proxies.fo")
}

func (p *ProxiesFoProvider) DeleteAccount(ctx context.Context, accountID string) error {
	// Implementation for deleting account
	// This would typically involve an API call to delete/disable the account
	return fmt.Errorf("DeleteAccount not implemented for Proxies.fo")
}

func (p *ProxiesFoProvider) TestConnection(ctx context.Context, account *ProviderAccount) error {
	// Test the proxy connection
	proxyURL := fmt.Sprintf("http://%s:%s@%s:%d",
		account.Username, account.Password, account.Host, account.Port)

	testURL := "http://httpbin.org/ip"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with proxy
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Set proxy
	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	client.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURLParsed),
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("proxy connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("proxy connection test failed with status: %d", resp.StatusCode)
	}

	p.logger.Info("Proxy connection test successful",
		zap.String("account_id", account.ID),
		zap.String("host", account.Host),
		zap.Int("port", account.Port),
	)

	return nil
}
