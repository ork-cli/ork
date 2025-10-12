package cli

import (
	"context"
	"fmt"

	"github.com/hary-singh/ork/internal/config"
	"github.com/hary-singh/ork/internal/docker"
	"github.com/hary-singh/ork/internal/service"
	"github.com/spf13/cobra"
)

// ============================================================================
// Cobra Command Definition
// ============================================================================

var upCmd = &cobra.Command{
	Use:   "up <service> [service...]",
	Short: "Start services and their dependencies",
	Long: `Start one or more services along with their dependencies.

	Ork automatically resolves and starts all required dependencies in the correct order.
	For example, if 'frontend' depends on 'api', and 'api' depends on 'postgres',
	running 'ork up frontend' will start all three services.`,
	Example: `  ork up frontend              Start frontend (and its dependencies)
  	ork up frontend api          Start multiple services
  	ork up --local frontend      Build and run from local source`,

	Args: cobra.MinimumNArgs(1), // Require at least one service name
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUp(args); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
	},
}

func init() {
	// Register the 'up' command with the root command
	rootCmd.AddCommand(upCmd)

	// Add flags (options) to the command
	upCmd.Flags().Bool("local", false, "Build and run from local source")
	upCmd.Flags().Bool("dev", false, "Use development registry images")
}

// ============================================================================
// Main Orchestrator
// ============================================================================

// runUp orchestrates the service startup process
func runUp(serviceNames []string) error {
	// Load and validate configuration
	cfg, err := loadAndValidateConfig()
	if err != nil {
		return err
	}

	// Verify requested services exist
	if err := validateServiceNames(serviceNames, cfg); err != nil {
		return err
	}

	// Create a Docker client
	dockerClient, err := createDockerClient()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			fmt.Printf("❌ Error closing Docker client: %v\n", closeErr)
		}
	}()

	// Resolve dependencies and get services in the correct start order
	orderedServices, err := service.ResolveDependencies(cfg.Services, serviceNames)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Create a project network for service communication
	ctx := context.Background()
	networkID, err := dockerClient.CreateNetwork(ctx, cfg.Project)
	if err != nil {
		return fmt.Errorf("failed to create project network: %w", err)
	}
	fmt.Printf("🌐 Created network: ork-%s-network\n", cfg.Project)

	// Show startup message
	fmt.Printf("✅ Loaded project: %s (version %s)\n", cfg.Project, cfg.Version)
	fmt.Printf("🚀 Starting services: %v\n", serviceNames)
	if len(orderedServices) > len(serviceNames) {
		fmt.Printf("📦 Including dependencies: %v\n", orderedServices)
	}

	// Create an orchestrator for parallel service management
	orchestrator := service.NewOrchestrator(cfg.Project, dockerClient, networkID)

	// Add all services to the orchestrator
	for _, serviceName := range orderedServices {
		orchestrator.AddService(serviceName, cfg.Services[serviceName])
	}

	// Start services with parallel execution, health checks, and rollback
	if err := orchestrator.StartServicesInOrder(ctx, orderedServices, cfg); err != nil {
		return err
	}

	fmt.Printf("✅ All services started successfully!\n")
	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadAndValidateConfig loads the ork.yml file and validates it
func loadAndValidateConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// ============================================================================
// Private Helpers - Service Validation
// ============================================================================

// validateServiceNames checks if all requested services exist in the config
func validateServiceNames(serviceNames []string, cfg *config.Config) error {
	for _, serviceName := range serviceNames {
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service '%s' not found in ork.yml\n💡 Available services: %s",
				serviceName, getAvailableServicesList(cfg))
		}
	}
	return nil
}

// getAvailableServicesList returns a formatted string of available services
func getAvailableServicesList(cfg *config.Config) string {
	services := ""
	for name := range cfg.Services {
		services += name + " "
	}
	return services
}

// ============================================================================
// Private Helpers - Docker Operations
// ============================================================================

// createDockerClient creates and verifies a Docker client connection
func createDockerClient() (*docker.Client, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return client, nil
}
