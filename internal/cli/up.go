package cli

import (
	"context"
	"fmt"

	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/docker"
	"github.com/ork-cli/ork/internal/service"
	"github.com/ork-cli/ork/internal/ui"
	"github.com/ork-cli/ork/pkg/utils"
	"github.com/spf13/cobra"
)

// ============================================================================
// Cobra Command Definition
// ============================================================================

var upCmd = &cobra.Command{
	Use:   "up <service> [service...]",
	Short: "Start services and their dependencies",
	Long: `
Start one or more services along with their dependencies.

Ork automatically resolves and starts all required dependencies in the correct order.
For example, if 'frontend' depends on 'api', and 'api' depends on 'postgres',
running 'ork up frontend' will start all three services.`,
	Example: `
ork up frontend              Start frontend (and its dependencies)
ork up frontend api          Start multiple services
ork up --local frontend      Build and run from local source`,

	Args: cobra.MinimumNArgs(1), // Require at least one service name
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUp(args); err != nil {
			handleUpError(err)
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
			ui.Warning(fmt.Sprintf("Failed to close Docker client: %v", closeErr))
		}
	}()

	// Resolve dependencies and get services in the correct start order
	orderedServices, err := service.ResolveDependencies(cfg.Services, serviceNames)
	if err != nil {
		return utils.ServiceError(
			"up.dependencies",
			"Failed to resolve service dependencies",
			"Check your service dependencies in ork.yml",
			err,
		)
	}

	// Create a project network for service communication
	ctx := context.Background()
	spinner := ui.ShowSpinner("Creating project network...")
	networkID, err := dockerClient.CreateNetwork(ctx, cfg.Project)
	if err != nil {
		spinner.Error("Failed to create network")
		return utils.NetworkError(
			"up.network",
			"Failed to create project network",
			"Check if Docker is running and you have permissions",
			err,
		)
	}
	spinner.Success(fmt.Sprintf("Created network: ork-%s-network", cfg.Project))

	// Show startup summary
	ui.EmptyLine()
	ui.Info(fmt.Sprintf("Project: %s (v%s)", ui.Bold(cfg.Project), cfg.Version))
	ui.Info(fmt.Sprintf("Starting: %s", ui.Highlight(fmt.Sprintf("%v", serviceNames))))
	if len(orderedServices) > len(serviceNames) {
		ui.Info(fmt.Sprintf("Dependencies: %s", ui.Dim(fmt.Sprintf("%v", orderedServices))))
	}
	ui.EmptyLine()

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

	ui.EmptyLine()
	ui.SuccessBox(fmt.Sprintf("All services started successfully! %s", ui.SymbolRocket))
	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadAndValidateConfig loads the ork.yml file and validates it
func loadAndValidateConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, utils.ConfigError(
			"up.load",
			"Failed to load configuration",
			"Make sure ork.yml exists in the current directory",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {
		return nil, utils.ConfigError(
			"up.validate",
			"Invalid configuration",
			"Check your ork.yml for errors",
			err,
		)
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
			availableServices := getAvailableServicesList(cfg)
			suggestions := utils.FindSuggestions(serviceName, availableServices, 3)

			err := utils.ErrServiceNotFound(serviceName, suggestions)
			err.Details = []string{
				fmt.Sprintf("Available services: %s", ui.Dim(fmt.Sprintf("%v", availableServices))),
			}
			return err
		}
	}
	return nil
}

// getAvailableServicesList returns a slice of available service names
func getAvailableServicesList(cfg *config.Config) []string {
	services := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		services = append(services, name)
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
		return nil, utils.DockerError(
			"up.docker",
			"Failed to connect to Docker",
			"Make sure Docker is running. Try 'docker ps' or run 'ork doctor'",
			err,
		)
	}
	return client, nil
}

// handleUpError formats and displays errors with hints
func handleUpError(err error) {
	if orkErr, ok := err.(*utils.OrkError); ok {
		// Display structured error with hints
		ui.Error(orkErr.Message)
		if orkErr.Hint != "" {
			ui.Hint(orkErr.Hint)
		}
		if len(orkErr.Details) > 0 {
			ui.EmptyLine()
			for _, detail := range orkErr.Details {
				ui.List(detail)
			}
		}
		if len(orkErr.Suggestions) > 0 {
			ui.EmptyLine()
			ui.Info("Did you mean:")
			for _, suggestion := range orkErr.Suggestions {
				ui.ListItem(ui.SymbolArrow, ui.Highlight(suggestion))
			}
		}
	} else {
		// Fallback for non-Ork errors
		ui.Error(fmt.Sprintf("Error: %v", err))
	}
}
