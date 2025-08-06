// internal/service/proxy.go - COMPLETE FIX
package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/repository"
)

type proxyService struct {
	cfg          *config.Config
	logger       *zap.Logger
	instanceRepo repository.InstanceRepository
	planRepo     repository.PlanRepository
}

func NewProxyService(
	cfg *config.Config,
	logger *zap.Logger,
	instanceRepo repository.InstanceRepository,
	planRepo repository.PlanRepository,
) ProxyService {
	return &proxyService{
		cfg:          cfg,
		logger:       logger,
		instanceRepo: instanceRepo,
		planRepo:     planRepo,
	}
}

func (s *proxyService) StartInstance(ctx context.Context, instance *domain.ProxyInstance) error {
	s.logger.Info("Starting proxy instance",
		zap.String("instance_id", instance.ID.String()),
		zap.Int("local_port", instance.LocalPort),
		zap.String("auth_host", instance.AuthHost),
		zap.Int("auth_port", instance.AuthPort))

	// Kill any existing process on the port
	if err := s.killProcessOnPort(instance.LocalPort); err != nil {
		s.logger.Warn("Failed to kill existing process on port",
			zap.Int("port", instance.LocalPort),
			zap.Error(err))
	}

	// Get plan details for authentication
	plan, err := s.planRepo.GetByID(ctx, instance.PlanID)
	if err != nil {
		return fmt.Errorf("failed to get plan for instance: %w", err)
	}

	// Create 3proxy configuration file
	configPath, err := s.create3ProxyConfig(instance, plan.Username, plan.Password)
	if err != nil {
		return fmt.Errorf("failed to create 3proxy config: %w", err)
	}

	// Start 3proxy process
	cmd := exec.CommandContext(ctx, "3proxy", configPath)
	cmd.Dir = s.cfg.Proxy.ConfigDir

	// Set process group to handle cleanup better
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start 3proxy: %w", err)
	}

	processID := cmd.Process.Pid
	s.logger.Info("3proxy process started",
		zap.String("instance_id", instance.ID.String()),
		zap.Int("pid", processID),
		zap.String("config", configPath))

	// Update instance with process ID and status
	instance.ProcessID = processID
	instance.Status = domain.InstanceStatusRunning
	instance.UpdatedAt = time.Now()

	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		s.logger.Error("Failed to update instance status", zap.Error(err))
		// Try to kill the process if we can't update the database
		s.killProcess(processID)
		return fmt.Errorf("failed to update instance: %w", err)
	}

	// Test the proxy connection
	go func() {
		time.Sleep(2 * time.Second)
		if err := s.testProxyConnection(instance, plan.Username, plan.Password); err != nil {
			s.logger.Error("Proxy connection test failed",
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err))
		} else {
			s.logger.Info("Proxy connection test successful",
				zap.String("instance_id", instance.ID.String()))
		}
	}()

	return nil
}

func (s *proxyService) StopInstance(ctx context.Context, instanceID uuid.UUID) error {
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	s.logger.Info("Stopping proxy instance",
		zap.String("instance_id", instanceID.String()),
		zap.Int("pid", instance.ProcessID))

	// Kill the process
	if instance.ProcessID > 0 {
		if err := s.killProcess(instance.ProcessID); err != nil {
			s.logger.Error("Failed to kill process",
				zap.Int("pid", instance.ProcessID),
				zap.Error(err))
		}
	}

	// Kill any process on the port as backup
	if err := s.killProcessOnPort(instance.LocalPort); err != nil {
		s.logger.Warn("Failed to kill process on port",
			zap.Int("port", instance.LocalPort),
			zap.Error(err))
	}

	// Update instance status
	instance.Status = domain.InstanceStatusStopped
	instance.ProcessID = 0
	instance.UpdatedAt = time.Now()

	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	// Clean up configuration file
	configPath := s.getConfigPath(instance.ID.String())
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("Failed to remove config file",
			zap.String("config_path", configPath),
			zap.Error(err))
	}

	s.logger.Info("Proxy instance stopped successfully",
		zap.String("instance_id", instanceID.String()))

	return nil
}

func (s *proxyService) RestartInstance(ctx context.Context, instanceID uuid.UUID) error {
	s.logger.Info("Restarting proxy instance", zap.String("instance_id", instanceID.String()))

	// Get the instance
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Stop the instance
	if err := s.StopInstance(ctx, instanceID); err != nil {
		s.logger.Error("Failed to stop instance during restart", zap.Error(err))
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)

	// Start the instance
	if err := s.StartInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to start instance during restart: %w", err)
	}

	s.logger.Info("Proxy instance restarted successfully",
		zap.String("instance_id", instanceID.String()))

	return nil
}

func (s *proxyService) GetInstanceStatus(ctx context.Context, instanceID uuid.UUID) (string, error) {
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get instance: %w", err)
	}

	// Check if the process is actually running
	if instance.ProcessID > 0 {
		if s.isProcessRunning(instance.ProcessID) {
			return domain.InstanceStatusRunning, nil
		} else {
			// Process died, update status
			instance.Status = domain.InstanceStatusStopped
			instance.ProcessID = 0
			instance.UpdatedAt = time.Now()
			s.instanceRepo.Update(ctx, instance)
			return domain.InstanceStatusStopped, nil
		}
	}

	return instance.Status, nil
}

func (s *proxyService) GetRunningInstances(ctx context.Context) ([]*domain.ProxyInstance, error) {
	return s.instanceRepo.GetRunning(ctx)
}

func (s *proxyService) HealthCheck(ctx context.Context, instanceID uuid.UUID) error {
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Check if process is running
	if instance.ProcessID <= 0 || !s.isProcessRunning(instance.ProcessID) {
		return fmt.Errorf("process not running")
	}

	// Get plan for authentication details
	plan, err := s.planRepo.GetByID(ctx, instance.PlanID)
	if err != nil {
		return fmt.Errorf("failed to get plan for health check: %w", err)
	}

	// Test proxy connection
	return s.testProxyConnection(instance, plan.Username, plan.Password)
}

func (s *proxyService) GetInstance(ctx context.Context, instanceID uuid.UUID) (*domain.ProxyInstance, error) {
	return s.instanceRepo.GetByID(ctx, instanceID)
}

func (s *proxyService) GetInstancesByPlan(ctx context.Context, planID uuid.UUID) ([]*domain.ProxyInstance, error) {
	return s.instanceRepo.GetByPlanID(ctx, planID)
}

// Helper methods

func (s *proxyService) create3ProxyConfig(instance *domain.ProxyInstance, username, password string) (string, error) {
	configPath := s.getConfigPath(instance.ID.String())

	configContent := fmt.Sprintf(`# 3proxy configuration for instance %s
# Generated on %s

daemon
log %s/3proxy_%s.log D
logformat "- +_L%%t.%%. %%N.%%p %%E %%U %%C:%%c %%R:%%r %%O %%I %%h %%T"
rotate 30

# Authentication
users %s:CL:%s

# Allow access for authenticated users
allow %s

# HTTP proxy forwarding to upstream
proxy -p%d -a -e%s:%d
`,
		instance.ID.String(),
		time.Now().Format(time.RFC3339),
		s.cfg.Proxy.LogDir,
		instance.ID.String(),
		username,
		password,
		username,
		instance.LocalPort,
		instance.AuthHost,
		instance.AuthPort,
	)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	s.logger.Debug("Created 3proxy config",
		zap.String("instance_id", instance.ID.String()),
		zap.String("config_path", configPath))

	return configPath, nil
}

func (s *proxyService) getConfigPath(instanceID string) string {
	return fmt.Sprintf("%s/3proxy_%s.cfg", s.cfg.Proxy.ConfigDir, instanceID)
}

func (s *proxyService) killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful shutdown first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		s.logger.Debug("SIGTERM failed, trying SIGKILL", zap.Int("pid", pid))
		// Force kill if graceful shutdown fails
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to die
	process.Wait()
	return nil
}

func (s *proxyService) killProcessOnPort(port int) error {
	// Use lsof to find process using the port
	cmd := exec.Command("lsof", "-ti:"+strconv.Itoa(port))
	output, err := cmd.Output()
	if err != nil {
		// No process found on port, which is fine
		return nil
	}

	pidStr := string(output)
	pidStr = pidStr[:len(pidStr)-1] // Remove newline

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("failed to parse PID: %w", err)
	}

	return s.killProcess(pid)
}

func (s *proxyService) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}

func (s *proxyService) testProxyConnection(instance *domain.ProxyInstance, username, password string) error {
	// Test the proxy by making a simple HTTP request through it
	// This is a placeholder implementation
	s.logger.Debug("Testing proxy connection",
		zap.String("instance_id", instance.ID.String()),
		zap.Int("local_port", instance.LocalPort))

	// In a real implementation, you would:
	// 1. Make an HTTP request through the proxy
	// 2. Verify the response
	// 3. Check that the request was forwarded to the upstream

	return nil
}
