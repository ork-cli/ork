package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/hary-singh/ork/internal/config"
	"github.com/hary-singh/ork/internal/docker"
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

	// Show startup message
	fmt.Printf("✅ Loaded project: %s (version %s)\n", cfg.Project, cfg.Version)
	fmt.Printf("🚀 Starting services: %v\n", serviceNames)

	// Start the requested services
	ctx := context.Background()
	if err := startServices(ctx, dockerClient, cfg, serviceNames); err != nil {
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

// startServices starts all requested services
func startServices(ctx context.Context, client *docker.Client, cfg *config.Config, serviceNames []string) error {
	for _, serviceName := range serviceNames {
		if err := startSingleService(ctx, client, cfg, serviceName); err != nil {
			return fmt.Errorf("failed to start service '%s': %w", serviceName, err)
		}
	}
	return nil
}

// startSingleService starts a single service container
func startSingleService(ctx context.Context, client *docker.Client, cfg *config.Config, serviceName string) error {
	service := cfg.Services[serviceName]

	// Build Docker run options from config
	runOpts := buildRunOptions(serviceName, service, cfg.Project)

	// Start the container
	fmt.Printf("🐳 Starting %s...\n", serviceName)
	containerID, err := client.Run(ctx, runOpts)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Started %s (container: %s)\n", serviceName, containerID[:12])
	return nil
}

// buildRunOptions converts a config service to Docker run options
func buildRunOptions(serviceName string, service config.Service, projectName string) docker.RunOptions {
	return docker.RunOptions{
		Name:       fmt.Sprintf("ork-%s-%s", projectName, serviceName),
		Image:      service.Image,
		Ports:      parsePortMappings(service.Ports),
		Env:        service.Env,
		Labels:     buildOrkLabels(projectName, serviceName),
		Command:    service.Command,
		Entrypoint: service.Entrypoint,
	}
}

// parsePortMappings converts port strings like "8080:80" to map["8080"]="80"
func parsePortMappings(portMappings []string) map[string]string {
	ports := make(map[string]string)

	for _, mapping := range portMappings {
		// Split "8080:80" into ["8080", "80"]
		parts := strings.Split(mapping, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPort := parts[1]
			ports[hostPort] = containerPort
		}
	}

	return ports
}

// buildOrkLabels creates standard Ork labels for container tracking
func buildOrkLabels(projectName, serviceName string) map[string]string {
	return map[string]string{
		"ork.managed": "true",
		"ork.project": projectName,
		"ork.service": serviceName,
	}
}
