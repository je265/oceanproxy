// internal/service/provider/proxiesfo.go - FIXED
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
    "os"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/pkg/config"
)

type ProxiesFoProvider struct {
	cfg    *config.ProxiesFoConfig
	logger *zap.Logger
	client *http.Client
}

// Temporary debug log path (will be removed later)
const proxiesFoDebugLogPath = "/home/oceanadmin/oceanproxy/proxiesfo_debug.log"
const proxiesFoDebugLogFallbackPath = "/var/log/oceanproxy/proxiesfo_debug.log"

// debugLogf appends masked debug lines to a local file. Best-effort; errors ignored.
func debugLogf(format string, args ...interface{}) {
    // Prefix with timestamp
    line := fmt.Sprintf("[%s] ", time.Now().Format(time.RFC3339)) + fmt.Sprintf(format, args...) + "\n"
    f, err := os.OpenFile(proxiesFoDebugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        // Try fallback location
        f, err = os.OpenFile(proxiesFoDebugLogFallbackPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return
        }
    }
    defer f.Close()
    _, _ = f.WriteString(line)
}

// maskKey returns a masked representation of sensitive values
func maskKey(v string) string {
    if v == "" {
        return "<empty>"
    }
    if len(v) <= 6 {
        return "***"
    }
    return v[:3] + strings.Repeat("*", len(v)-5) + v[len(v)-2:]
}

// sanitizeForm masks sensitive fields and returns an encoded string
func sanitizeForm(v url.Values) string {
    if v == nil {
        return ""
    }
    copyVals := url.Values{}
    for k, vals := range v {
        switch strings.ToLower(k) {
        case "password", "authpassword":
            copyVals[k] = []string{"<masked>"}
        case "username", "authusername":
            masked := "<masked>"
            if len(vals) > 0 {
                u := vals[0]
                if len(u) > 2 {
                    masked = u[:1] + strings.Repeat("*", len(u)-2) + u[len(u)-1:]
                }
            }
            copyVals[k] = []string{masked}
        default:
            copyVals[k] = vals
        }
    }
    return copyVals.Encode()
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
// ProxiesFoResponse represents the API response from Proxies.fo.
// "Data" may be either an object or an array depending on endpoint/inputs.
type ProxiesFoResponse struct {
    Success bool              `json:"Success"`
    Data    ProxiesFoDataAny  `json:"Data"`
    Error   string            `json:"Error"`
}

// ProxiesFoDataAny accepts either a single object or an array of objects
type ProxiesFoDataAny struct {
    Items []ProxiesFoData
}

func (d *ProxiesFoDataAny) UnmarshalJSON(b []byte) error {
    // Try object first
    var obj ProxiesFoData
    if err := json.Unmarshal(b, &obj); err == nil && (obj.ID != "" || obj.AuthUsername != "") {
        d.Items = []ProxiesFoData{obj}
        return nil
    }
    // Try array
    var arr []ProxiesFoData
    if err := json.Unmarshal(b, &arr); err == nil {
        d.Items = arr
        return nil
    }
    // Unknown format; leave empty
    d.Items = nil
    return nil
}

type ProxiesFoData struct {
	ID           string  `json:"ID"`
    User         string  `json:"User"`
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

    // TEMP DEBUG: Begin request context
    debugLogf("CreateAccount start: customer_id=%q plan_type=%q region=%q base_url=%q", req.CustomerID, req.PlanType, req.Region, p.cfg.BaseURL)

	// Map plan types to Proxies.fo reseller IDs
	resellerMap := map[string]string{
		"residential": "7c9ea873-63f9-4013-9147-3807cc6f0553",
		"isp":         "3471aa35-7922-488a-a7a9-b92a5510080e",
		"datacenter":  "b3fd0f3c-693d-4ec5-b49f-c77feaab0b72",
	}

	resellerID, ok := resellerMap[req.PlanType]
	if !ok {
        debugLogf("Unsupported plan type: %q", req.PlanType)
		return nil, fmt.Errorf("unsupported plan type: %s", req.PlanType)
	}

	// Prepare form data
    formData := url.Values{}
    // According to Proxies.fo docs, keys are capitalized
    formData.Set("Reseller", resellerID)
    formData.Set("Username", req.Username)
    formData.Set("Password", req.Password)

	// Set plan-specific parameters
    if req.PlanType == "datacenter" {
		duration := req.Duration
		if duration == 0 {
			duration = 1 // Default to 1 day
		}
        formData.Set("Duration", strconv.Itoa(duration))
        formData.Set("Threads", "500") // Default thread limit
	} else {
		// Residential/ISP plans
        formData.Set("Duration", "180") // 180 days
		bandwidth := req.Bandwidth
		if bandwidth == 0 {
			bandwidth = 1 // Default to 1GB
		}
        // API expects Bandwidth as float; format with no trailing .00 if integer
        formData.Set("Bandwidth", strconv.FormatFloat(float64(bandwidth), 'f', -1, 64))
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/api/plans/new", p.cfg.BaseURL)
    debugLogf("Request URL: %s", apiURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
        debugLogf("Error creating request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-Api-Auth", p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // TEMP DEBUG: Log masked headers and form
    debugLogf("Headers: X-Api-Auth=%s, Content-Type=%s", maskKey(p.cfg.APIKey), httpReq.Header.Get("Content-Type"))
    debugLogf("Form (sanitized): %s", sanitizeForm(formData))

	p.logger.Debug("Sending request to Proxies.fo API",
		zap.String("url", apiURL),
		zap.String("form_data", formData.Encode()),
	)

	resp, err := p.client.Do(httpReq)
	if err != nil {
        debugLogf("HTTP error: %v", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for debugging and parsing
	body, err := io.ReadAll(resp.Body)
	if err != nil {
        debugLogf("Read body error: %v", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

    // TEMP DEBUG: Status and raw body
    debugLogf("Response status: %d", resp.StatusCode)
    debugLogf("Raw body: %s", string(body))

	p.logger.Debug("Raw API response", zap.String("body", string(body)))

	var result ProxiesFoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		p.logger.Error("Failed to decode response",
			zap.String("raw_response", string(body)),
			zap.Error(err),
		)
        debugLogf("JSON unmarshal error: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.logger.Debug("Received response from Proxies.fo API",
		zap.Any("response", result),
	)

	if !result.Success {
        debugLogf("API reported failure: %s", result.Error)
		return nil, fmt.Errorf("Proxies.fo API error: %s", result.Error)
	}

    // Normalize to first item
    if len(result.Data.Items) == 0 {
        return nil, fmt.Errorf("no data returned from Proxies.fo API")
    }
    data := result.Data.Items[0]

	// Determine the correct upstream host based on response
	upstreamHost := data.AuthHostname
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
		ID:       data.ID,
        CustomerID: data.User,
		Username: data.AuthUsername,
		Password: data.AuthPassword,
		Host:     upstreamHost,
		Port:     int(data.AuthPort),
		Region:   req.Region,
	}

	p.logger.Info("Successfully created Proxies.fo account",
		zap.String("account_id", account.ID),
		zap.String("username", account.Username),
		zap.String("host", account.Host),
		zap.Int("port", account.Port),
	)

    // TEMP DEBUG: Success summary (mask sensitive fields)
    debugLogf("Success: id=%q user=%q host=%q port=%d", account.ID, sanitizeForm(url.Values{"username": {account.Username}}), account.Host, account.Port)

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
