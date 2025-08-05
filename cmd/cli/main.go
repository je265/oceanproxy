package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/je265/oceanproxy/internal/config"
	"github.com/je265/oceanproxy/internal/domain"
	"github.com/je265/oceanproxy/internal/pkg/logger"
	"github.com/je265/oceanproxy/internal/repository"
	"github.com/je265/oceanproxy/internal/repository/json"
	"github.com/je265/oceanproxy/internal/service"
)

const version = "1.0.0"

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		configFile  = flag.String("config", "configs/config.yaml", "Configuration file path")
		command     = flag.String("command", "", "Command to execute")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("OceanProxy CLI v%s\n", version)
		os.Exit(0)
	}

	if *command == "" {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logLevel := "info"
	if *verbose {
		logLevel = "debug"
	}
	log := logger.New(logLevel, "console")

	// Initialize repositories
	planRepo := json.NewPlanRepository(cfg.Database.DSN, log)
	instanceRepo := json.NewInstanceRepository(cfg.Database.DSN, log)

	// Initialize services
	providerService := service.NewProviderService(cfg, log)
	proxyService := service.NewProxyService(cfg, log, instanceRepo)

	// Execute command
	switch *command {
	case "list-plans":
		listPlans(planRepo)
	case "list-instances":
		listInstances(instanceRepo)
	case "create-plan":
		createPlan(planRepo, providerService, flag.Args())
	case "delete-plan":
		deletePlan(planRepo, flag.Args())
	case "start-instance":
		startInstance(proxyService, flag.Args())
	case "stop-instance":
		stopInstance(proxyService, flag.Args())
	case "status":
		showStatus(planRepo, instanceRepo)
	case "cleanup":
		cleanup(planRepo, instanceRepo, proxyService)
	case "health-check":
		healthCheck(proxyService, flag.Args())
	case "export":
		exportData(planRepo, instanceRepo, flag.Args())
	case "import":
		importData(planRepo, instanceRepo, flag.Args())
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("OceanProxy CLI - Command line interface for OceanProxy management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  oceanproxy-cli [flags] -command <command> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -version          Show version information")
	fmt.Println("  -config string    Configuration file path (default: configs/config.yaml)")
	fmt.Println("  -verbose          Enable verbose output")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list-plans                    List all proxy plans")
	fmt.Println("  list-instances                List all proxy instances")
	fmt.Println("  create-plan <args>            Create a new proxy plan")
	fmt.Println("  delete-plan <plan-id>         Delete a proxy plan")
	fmt.Println("  start-instance <instance-id>  Start a proxy instance")
	fmt.Println("  stop-instance <instance-id>   Stop a proxy instance")
	fmt.Println("  status                        Show system status")
	fmt.Println("  cleanup                       Clean up stopped/failed instances")
	fmt.Println("  health-check [instance-id]    Run health checks")
	fmt.Println("  export <file>                 Export data to file")
	fmt.Println("  import <file>                 Import data from file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  oceanproxy-cli -command list-plans")
	fmt.Println("  oceanproxy-cli -command create-plan customer123 residential proxies_fo usa testuser testpass 10 30")
	fmt.Println("  oceanproxy-cli -command status")
}

func listPlans(planRepo repository.PlanRepository) {
	plans, err := planRepo.GetAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get plans: %v\n", err)
		os.Exit(1)
	}

	if len(plans) == 0 {
		fmt.Println("No plans found")
		return
	}

	fmt.Printf("%-36s %-15s %-12s %-12s %-10s %-10s %s\n",
		"ID", "Customer", "Provider", "Plan Type", "Region", "Status", "Expires")
	fmt.Println(strings.Repeat("-", 120))

	for _, plan := range plans {
		fmt.Printf("%-36s %-15s %-12s %-12s %-10s %-10s %s\n",
			plan.ID.String(),
			truncate(plan.CustomerID, 15),
			plan.Provider,
			plan.PlanType,
			plan.Region,
			plan.Status,
			plan.ExpiresAt.Format("2006-01-02"))
	}
}

func listInstances(instanceRepo repository.InstanceRepository) {
	instances, err := instanceRepo.GetAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get instances: %v\n", err)
		os.Exit(1)
	}

	if len(instances) == 0 {
		fmt.Println("No instances found")
		return
	}

	fmt.Printf("%-36s %-36s %-10s %-25s %-10s %s\n",
		"ID", "Plan ID", "Port", "Plan Type", "Status", "Created")
	fmt.Println(strings.Repeat("-", 130))

	for _, instance := range instances {
		fmt.Printf("%-36s %-36s %-10d %-25s %-10s %s\n",
			instance.ID.String(),
			instance.PlanID.String(),
			instance.LocalPort,
			truncate(instance.PlanTypeKey, 25),
			instance.Status,
			instance.CreatedAt.Format("2006-01-02 15:04"))
	}
}

func createPlan(planRepo repository.PlanRepository, providerService service.ProviderService, args []string) {
	if len(args) < 7 {
		fmt.Println("Usage: create-plan <customer-id> <plan-type> <provider> <region> <username> <password> <bandwidth> [duration]")
		os.Exit(1)
	}

	bandwidth, err := strconv.Atoi(args[6])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid bandwidth: %v\n", err)
		os.Exit(1)
	}

	duration := 30 // default 30 days
	if len(args) > 7 {
		duration, err = strconv.Atoi(args[7])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid duration: %v\n", err)
			os.Exit(1)
		}
	}

	req := &domain.CreatePlanRequest{
		CustomerID: args[0],
		PlanType:   args[1],
		Provider:   args[2],
		Region:     args[3],
		Username:   args[4],
		Password:   args[5],
		Bandwidth:  bandwidth,
		Duration:   duration,
	}

	// Create plan
	plan := &domain.ProxyPlan{
		ID:         uuid.New(),
		CustomerID: req.CustomerID,
		PlanType:   req.PlanType,
		Provider:   req.Provider,
		Region:     req.Region,
		Username:   req.Username,
		Password:   req.Password,
		Status:     domain.PlanStatusCreating,
		Bandwidth:  req.Bandwidth,
		ExpiresAt:  time.Now().AddDate(0, 0, req.Duration),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := planRepo.Create(context.Background(), plan); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Plan created successfully: %s\n", plan.ID.String())
}

func deletePlan(planRepo repository.PlanRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: delete-plan <plan-id>")
		os.Exit(1)
	}

	planID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid plan ID: %v\n", err)
		os.Exit(1)
	}

	if err := planRepo.Delete(context.Background(), planID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Plan deleted successfully: %s\n", planID.String())
}

func startInstance(proxyService service.ProxyService, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: start-instance <instance-id>")
		os.Exit(1)
	}

	instanceID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid instance ID: %v\n", err)
		os.Exit(1)
	}

	instance, err := proxyService.GetInstance(context.Background(), instanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get instance: %v\n", err)
		os.Exit(1)
	}

	if err := proxyService.StartInstance(context.Background(), instance); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start instance: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Instance started successfully: %s\n", instanceID.String())
}

func stopInstance(proxyService service.ProxyService, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: stop-instance <instance-id>")
		os.Exit(1)
	}

	instanceID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid instance ID: %v\n", err)
		os.Exit(1)
	}

	if err := proxyService.StopInstance(context.Background(), instanceID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop instance: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Instance stopped successfully: %s\n", instanceID.String())
}

func showStatus(planRepo repository.PlanRepository, instanceRepo repository.InstanceRepository) {
	// Get plan statistics
	totalPlans, _ := planRepo.Count(context.Background())
	activePlans, _ := planRepo.CountByStatus(context.Background(), domain.PlanStatusActive)
	expiredPlans, _ := planRepo.CountByStatus(context.Background(), domain.PlanStatusExpired)

	// Get instance statistics
	totalInstances, _ := instanceRepo.Count(context.Background())
	runningInstances, _ := instanceRepo.CountByStatus(context.Background(), domain.InstanceStatusRunning)
	stoppedInstances, _ := instanceRepo.CountByStatus(context.Background(), domain.InstanceStatusStopped)

	fmt.Println("OceanProxy System Status")
	fmt.Println("========================")
	fmt.Printf("Plans:\n")
	fmt.Printf("  Total: %d\n", totalPlans)
	fmt.Printf("  Active: %d\n", activePlans)
	fmt.Printf("  Expired: %d\n", expiredPlans)
	fmt.Printf("\nInstances:\n")
	fmt.Printf("  Total: %d\n", totalInstances)
	fmt.Printf("  Running: %d\n", runningInstances)
	fmt.Printf("  Stopped: %d\n", stoppedInstances)

	// Show recent activity
	plans, err := planRepo.GetAll(context.Background())
	if err == nil && len(plans) > 0 {
		fmt.Printf("\nRecent Plans:\n")
		for i, plan := range plans {
			if i >= 5 { // Show only last 5
				break
			}
			fmt.Printf("  %s - %s (%s)\n",
				plan.CreatedAt.Format("2006-01-02 15:04"),
				truncate(plan.CustomerID, 20),
				plan.Status)
		}
	}
}

func cleanup(planRepo repository.PlanRepository, instanceRepo repository.InstanceRepository, proxyService service.ProxyService) {
	fmt.Println("Running cleanup...")

	// Find expired plans
	expiredPlans, err := planRepo.GetExpired(context.Background(), time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get expired plans: %v\n", err)
		return
	}

	fmt.Printf("Found %d expired plans\n", len(expiredPlans))

	for _, plan := range expiredPlans {
		// Update plan status
		plan.Status = domain.PlanStatusExpired
		planRepo.Update(context.Background(), plan)

		// Stop associated instances
		instances, err := instanceRepo.GetByPlanID(context.Background(), plan.ID)
		if err != nil {
			continue
		}

		for _, instance := range instances {
			if instance.Status == domain.InstanceStatusRunning {
				proxyService.StopInstance(context.Background(), instance.ID)
				fmt.Printf("Stopped instance %s for expired plan %s\n",
					instance.ID.String(), plan.ID.String())
			}
		}
	}

	// Find failed instances
	failedInstances, err := instanceRepo.GetByStatus(context.Background(), domain.InstanceStatusFailed)
	if err == nil {
		fmt.Printf("Found %d failed instances\n", len(failedInstances))
		for _, instance := range failedInstances {
			// Try to restart failed instances
			if err := proxyService.RestartInstance(context.Background(), instance.ID); err != nil {
				fmt.Printf("Failed to restart instance %s: %v\n", instance.ID.String(), err)
			} else {
				fmt.Printf("Restarted failed instance %s\n", instance.ID.String())
			}
		}
	}

	fmt.Println("Cleanup completed")
}

func healthCheck(proxyService service.ProxyService, args []string) {
	if len(args) > 0 {
		// Check specific instance
		instanceID, err := uuid.Parse(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid instance ID: %v\n", err)
			os.Exit(1)
		}

		if err := proxyService.HealthCheck(context.Background(), instanceID); err != nil {
			fmt.Printf("Health check FAILED for instance %s: %v\n", instanceID.String(), err)
			os.Exit(1)
		} else {
			fmt.Printf("Health check PASSED for instance %s\n", instanceID.String())
		}
	} else {
		// Check all running instances
		instances, err := proxyService.GetRunningInstances(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get running instances: %v\n", err)
			os.Exit(1)
		}

		passed := 0
		failed := 0

		for _, instance := range instances {
			if err := proxyService.HealthCheck(context.Background(), instance.ID); err != nil {
				fmt.Printf("FAIL: %s - %v\n", instance.ID.String(), err)
				failed++
			} else {
				fmt.Printf("PASS: %s\n", instance.ID.String())
				passed++
			}
		}

		fmt.Printf("\nHealth Check Summary: %d passed, %d failed\n", passed, failed)
		if failed > 0 {
			os.Exit(1)
		}
	}
}

func exportData(planRepo repository.PlanRepository, instanceRepo repository.InstanceRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: export <filename>")
		os.Exit(1)
	}

	filename := args[0]

	// Get all data
	plans, err := planRepo.GetAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get plans: %v\n", err)
		os.Exit(1)
	}

	instances, err := instanceRepo.GetAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get instances: %v\n", err)
		os.Exit(1)
	}

	// Create export data structure
	exportData := struct {
		Plans      []*domain.ProxyPlan     `json:"plans"`
		Instances  []*domain.ProxyInstance `json:"instances"`
		ExportedAt time.Time               `json:"exported_at"`
		Version    string                  `json:"version"`
	}{
		Plans:      plans,
		Instances:  instances,
		ExportedAt: time.Now(),
		Version:    version,
	}

	// Write to file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Data exported to %s\n", filename)
	fmt.Printf("Plans: %d, Instances: %d\n", len(plans), len(instances))
}

func importData(planRepo repository.PlanRepository, instanceRepo repository.InstanceRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: import <filename>")
		os.Exit(1)
	}

	filename := args[0]

	// Read file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Parse JSON
	var importData struct {
		Plans      []*domain.ProxyPlan     `json:"plans"`
		Instances  []*domain.ProxyInstance `json:"instances"`
		ExportedAt time.Time               `json:"exported_at"`
		Version    string                  `json:"version"`
	}

	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode data: %v\n", err)
		os.Exit(1)
	}

	// Import plans
	for _, plan := range importData.Plans {
		if err := planRepo.Create(context.Background(), plan); err != nil {
			fmt.Printf("Warning: Failed to import plan %s: %v\n", plan.ID.String(), err)
		}
	}

	// Import instances
	for _, instance := range importData.Instances {
		if err := instanceRepo.Create(context.Background(), instance); err != nil {
			fmt.Printf("Warning: Failed to import instance %s: %v\n", instance.ID.String(), err)
		}
	}

	fmt.Printf("Data imported from %s\n", filename)
	fmt.Printf("Plans: %d, Instances: %d\n", len(importData.Plans), len(importData.Instances))
}

// Helper functions
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
