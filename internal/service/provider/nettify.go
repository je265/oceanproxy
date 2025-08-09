// internal/service/provider/nettify.go - FIXED
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/pkg/config"
)

type NettifyProvider struct {
	cfg    *config.NettifyConfig
	logger *zap.Logger
	client *http.Client
}

func NewNettifyProvider(cfg *config.NettifyConfig, logger *zap.Logger) *NettifyProvider {
	return &NettifyProvider{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// NettifyCreateResponse represents the API response from Nettify
type NettifyCreateResponse struct {
	PlanID   string `json:"plan_id"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// NettifyPlanDetails represents detailed plan information
type NettifyPlanDetails struct {
	PlanID    string `json:"plan_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	PlanType  string `json:"plan_type"`
	MaxBytes  int64  `json:"max_bytes"`
	UsedBytes int64  `json:"used_bytes"`
	Enabled   bool   `json:"enabled"`
	Active    bool   `json:"active"`
	LastUsed  string `json:"last_used"`
}

func (n *NettifyProvider) CreateAccount(ctx context.Context, req *domain.CreatePlanRequest) (*ProviderAccount, error) {
	n.logger.Info("Creating Nettify account",
		zap.String("customer_id", req.CustomerID),
		zap.String("plan_type", req.PlanType),
	)

    // Use provided username as-is (Nettify accepts custom usernames)
    username := req.Username

	var requestData map[string]interface{}

    if req.PlanType == "unlimited" {
		// Time-based unlimited plan
        hours := req.Duration * 24
        if req.Duration == 0 && hours == 0 {
            hours = 720 // Default example 30 days
        }

		requestData = map[string]interface{}{
			"username":       username,
			"password":       req.Password,
			"plan_type":      req.PlanType,
			"duration_hours": hours,
		}
    } else {
        // Bandwidth-based plan (residential, mobile, datacenter)
        // The API expects bandwidth_mb directly
        bandwidthMB := req.Bandwidth * 1024
        if bandwidthMB == 0 {
            bandwidthMB = 1024 // default to 1GB
        }

        requestData = map[string]interface{}{
            "username":     username,
            "password":     req.Password,
            "plan_type":    req.PlanType,
            "bandwidth_mb": bandwidthMB,
        }
    }

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/plans/create", n.cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+n.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	n.logger.Debug("Sending request to Nettify API",
		zap.String("url", apiURL),
		zap.String("request_data", string(jsonData)),
	)

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)

		if message, exists := errorResp["message"]; exists {
			return nil, fmt.Errorf("Nettify API error (%d): %v", resp.StatusCode, message)
		}
		return nil, fmt.Errorf("Nettify API error: status code %d", resp.StatusCode)
	}

	var result NettifyCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	n.logger.Debug("Received response from Nettify API",
		zap.Any("response", result),
	)

	// Get plan details to retrieve password and other info
	details, err := n.getPlanDetails(ctx, result.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan details: %w", err)
	}

	// Determine upstream configuration based on plan type
	upstreamHost, upstreamPort := n.getUpstreamConfig(req.PlanType)

	account := &ProviderAccount{
		ID:       result.PlanID,
		Username: details.Username,
		Password: details.Password,
		Host:     upstreamHost,
		Port:     upstreamPort,
		Region:   req.Region,
	}

	n.logger.Info("Successfully created Nettify account",
		zap.String("account_id", account.ID),
		zap.String("username", account.Username),
		zap.String("host", account.Host),
		zap.Int("port", account.Port),
	)

	return account, nil
}

func (n *NettifyProvider) getPlanDetails(ctx context.Context, planID string) (*NettifyPlanDetails, error) {
	apiURL := fmt.Sprintf("%s/plans/%s", n.cfg.BaseURL, planID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+n.cfg.APIKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get plan details: status code %d", resp.StatusCode)
	}

	var details NettifyPlanDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode plan details: %w", err)
	}

	return &details, nil
}

func (n *NettifyProvider) getUpstreamConfig(planType string) (string, int) {
	// Map plan types to their upstream configurations
	switch planType {
	case "residential":
		return "proxy.nettify.xyz", 8080
	case "datacenter":
		return "proxy.nettify.xyz", 8765
	case "mobile":
		return "proxy.nettify.xyz", 7654
	case "unlimited":
		return "proxy.nettify.xyz", 6543
	default:
		return "proxy.nettify.xyz", 8080
	}
}

func (n *NettifyProvider) GetAccountInfo(ctx context.Context, accountID string) (*ProviderAccount, error) {
	details, err := n.getPlanDetails(ctx, accountID)
	if err != nil {
		return nil, err
	}

	upstreamHost, upstreamPort := n.getUpstreamConfig(details.PlanType)

	return &ProviderAccount{
		ID:       details.PlanID,
		Username: details.Username,
		Password: details.Password,
		Host:     upstreamHost,
		Port:     upstreamPort,
	}, nil
}

func (n *NettifyProvider) DeleteAccount(ctx context.Context, accountID string) error {
	// Implementation for deleting account
	// This would typically involve an API call to delete/disable the account
	return fmt.Errorf("DeleteAccount not implemented for Nettify")
}

func (n *NettifyProvider) TestConnection(ctx context.Context, account *ProviderAccount) error {
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

	n.logger.Info("Proxy connection test successful",
		zap.String("account_id", account.ID),
		zap.String("host", account.Host),
		zap.Int("port", account.Port),
	)

	return nil
}

// GetAllPlans retrieves all plans from Nettify API
func (n *NettifyProvider) GetAllPlans(ctx context.Context) ([]NettifyPlanDetails, error) {
	apiURL := fmt.Sprintf("%s/plans", n.cfg.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+n.cfg.APIKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get plans: status code %d", resp.StatusCode)
	}

	var plans []NettifyPlanDetails
	if err := json.NewDecoder(resp.Body).Decode(&plans); err != nil {
		return nil, fmt.Errorf("failed to decode plans: %w", err)
	}

	return plans, nil
}
