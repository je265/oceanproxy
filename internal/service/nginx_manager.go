package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
)

// NginxManager manages nginx configuration for proxy load balancing
type NginxManager struct {
	logger      *zap.Logger
	cfg         *config.Config
	regions     map[string]*domain.Region
	planTypes   map[string]*domain.PlanTypeConfig
	configDir   string
	templateDir string
}

// NewNginxManager creates a new nginx manager
func NewNginxManager(
	logger *zap.Logger,
	cfg *config.Config,
	regions map[string]*domain.Region,
	planTypes map[string]*domain.PlanTypeConfig,
) *NginxManager {
	return &NginxManager{
		logger:      logger,
		cfg:         cfg,
		regions:     regions,
		planTypes:   planTypes,
		configDir:   cfg.Proxy.NginxConfDir,
		templateDir: filepath.Join(cfg.Proxy.ScriptDir, "nginx", "templates"),
	}
}

// UpdateUpstream adds a new server to an nginx upstream
func (nm *NginxManager) UpdateUpstream(ctx context.Context, planTypeKey string, localPort int) error {
	planType, exists := nm.planTypes[planTypeKey]
	if !exists {
		return fmt.Errorf("plan type %s not found", planTypeKey)
	}

	region, exists := nm.regions[planType.Region]
	if !exists {
		return fmt.Errorf("region %s not found", planType.Region)
	}

	configFile := filepath.Join(nm.configDir, region.NginxConfigFile)

	// Check if config file exists, create if not
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := nm.createRegionConfig(region); err != nil {
			return fmt.Errorf("failed to create region config: %w", err)
		}
	}

	// Add server to upstream
	if err := nm.addServerToUpstream(configFile, planType.NginxUpstreamName, localPort); err != nil {
		return fmt.Errorf("failed to add server to upstream: %w", err)
	}

	// Test and reload nginx
	if err := nm.testAndReloadNginx(); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	nm.logger.Info("Updated nginx upstream",
		zap.String("plan_type", planTypeKey),
		zap.String("upstream", planType.NginxUpstreamName),
		zap.Int("local_port", localPort),
	)

	return nil
}

// RemoveFromUpstream removes a server from an nginx upstream
func (nm *NginxManager) RemoveFromUpstream(ctx context.Context, planTypeKey string, localPort int) error {
	planType, exists := nm.planTypes[planTypeKey]
	if !exists {
		return fmt.Errorf("plan type %s not found", planTypeKey)
	}

	region, exists := nm.regions[planType.Region]
	if !exists {
		return fmt.Errorf("region %s not found", planType.Region)
	}

	configFile := filepath.Join(nm.configDir, region.NginxConfigFile)

	// Remove server from upstream
	if err := nm.removeServerFromUpstream(configFile, planType.NginxUpstreamName, localPort); err != nil {
		return fmt.Errorf("failed to remove server from upstream: %w", err)
	}

	// Test and reload nginx
	if err := nm.testAndReloadNginx(); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	nm.logger.Info("Removed from nginx upstream",
		zap.String("plan_type", planTypeKey),
		zap.String("upstream", planType.NginxUpstreamName),
		zap.Int("local_port", localPort),
	)

	return nil
}

// createRegionConfig creates nginx configuration for a region
func (nm *NginxManager) createRegionConfig(region *domain.Region) error {
	templateFile := filepath.Join(nm.templateDir, "stream.conf.tmpl")
	configFile := filepath.Join(nm.configDir, region.NginxConfigFile)

	// Read template
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Get plan types for this region
	var upstreams []UpstreamConfig
	for _, planTypeKey := range region.PlanTypes {
		if planType, exists := nm.planTypes[planTypeKey]; exists {
			upstreams = append(upstreams, UpstreamConfig{
				Name:     planType.NginxUpstreamName,
				PlanType: planTypeKey,
			})
		}
	}

	data := RegionTemplateData{
		Region:    region,
		Upstreams: upstreams,
	}

	// Create config file
	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	nm.logger.Info("Created nginx region config",
		zap.String("region", region.Name),
		zap.String("config_file", configFile),
	)

	return nil
}

// addServerToUpstream adds a server to an nginx upstream
func (nm *NginxManager) addServerToUpstream(configFile, upstreamName string, port int) error {
	// Read current config
	content, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	serverLine := fmt.Sprintf("    server 127.0.0.1:%d;", port)

	// Check if server already exists
	if contains(string(content), serverLine) {
		nm.logger.Debug("Server already exists in upstream",
			zap.String("upstream", upstreamName),
			zap.Int("port", port),
		)
		return nil
	}

	// Use sed to add server to upstream
	cmd := exec.Command("sed", "-i",
		fmt.Sprintf("/upstream %s {/a\\    server 127.0.0.1:%d;", upstreamName, port),
		configFile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add server to upstream: %w", err)
	}

	return nil
}

// removeServerFromUpstream removes a server from an nginx upstream
func (nm *NginxManager) removeServerFromUpstream(configFile, upstreamName string, port int) error {
	serverLine := fmt.Sprintf("    server 127.0.0.1:%d;", port)

	// Use sed to remove server from upstream
	cmd := exec.Command("sed", "-i",
		fmt.Sprintf("/%s/d", serverLine),
		configFile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove server from upstream: %w", err)
	}

	return nil
}

// testAndReloadNginx tests nginx configuration and reloads if valid
func (nm *NginxManager) testAndReloadNginx() error {
	// Test nginx configuration
	cmd := exec.Command("nginx", "-t")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nginx configuration test failed: %w", err)
	}

	// Reload nginx
	cmd = exec.Command("systemctl", "reload", "nginx")
	if err := cmd.Run(); err != nil {
		// Try alternative reload method
		cmd = exec.Command("service", "nginx", "reload")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	return nil
}

// RegenerateAllConfigs regenerates all nginx configurations
func (nm *NginxManager) RegenerateAllConfigs(ctx context.Context) error {
	for _, region := range nm.regions {
		if err := nm.createRegionConfig(region); err != nil {
			return fmt.Errorf("failed to create config for region %s: %w", region.Name, err)
		}
	}

	return nm.testAndReloadNginx()
}

// Template data structures
type RegionTemplateData struct {
	Region    *domain.Region
	Upstreams []UpstreamConfig
}

type UpstreamConfig struct {
	Name     string
	PlanType string
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
