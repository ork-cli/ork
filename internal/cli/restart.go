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

var restartCmd = &cobra.Command{
	Use:   "restart <service> [service...]",
	Short: "Restart one or more services",
	Long: `
Restart one or more services by stopping and recreating them.

This command always re-reads ork.yml and recreates containers with the latest
configuration, picking up changes to:
  - Environment variables
  - Port mappings
  - Docker image
  - Commands and entrypoints
  - Build configuration (with --force-rebuild)

Only the specified services are restarted - dependencies are not affected.`,
	Example: `
ork restart api                  Restart API service
ork restart api frontend         Restart multiple services
ork restart api --force-rebuild  Rebuild image from source before restarting`,

	Args: cobra.MinimumNArgs(1), // Require at least one service name
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		forceRebuild, _ := cmd.Flags().GetBool("force-rebuild")

		if err := runRestart(args, forceRebuild); err != nil {
			handleRestartError(err)
			return
		}
	},
}

func init() {
	// Register the 'restart' command with the root command
	rootCmd.AddCommand(restartCmd)

	// Add flags
	restartCmd.Flags().Bool("force-rebuild", false, "Force rebuild image even if no changes detected")
}

// ============================================================================
// Main Orchestrator
// ============================================================================

// runRestart orchestrates the service restart process
func runRestart(serviceNames []string, forceRebuild bool) error {
	// Load and validate configuration (fresh read to detect changes)
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

	// Get the network ID for the project
	ctx := context.Background()
	networkID, err := getProjectNetworkID(ctx, dockerClient, cfg.Project)
	if err != nil {
		// If the network doesn't exist, we'll need to create it when restarting
		ui.Warning(fmt.Sprintf("Project network not found, will create during restart: %v", err))
		networkID = ""
	}

	// Show restart summary
	ui.EmptyLine()
	ui.Info(fmt.Sprintf("Project: %s (v%s)", ui.Bold(cfg.Project), cfg.Version))
	ui.Info(fmt.Sprintf("Restarting: %s", ui.Highlight(fmt.Sprintf("%v", serviceNames))))
	ui.EmptyLine()

	// Restart each service
	for _, serviceName := range serviceNames {
		if err := restartService(ctx, cfg, serviceName, dockerClient, networkID, forceRebuild); err != nil {
			return err
		}
	}

	ui.EmptyLine()
	ui.SuccessBox(fmt.Sprintf("Successfully restarted %d service(s)! %s", len(serviceNames), ui.SymbolRocket))
	return nil
}

// ============================================================================
// Private Helpers - Service Restart Logic
// ============================================================================

// restartService restarts a single service with smart config change detection
func restartService(ctx context.Context, cfg *config.Config, serviceName string, client *docker.Client, networkID string, forceRebuild bool) error {
	newServiceCfg := cfg.Services[serviceName]

	// Get the current running container (if any)
	containers, err := client.List(ctx, cfg.Project)
	if err != nil {
		return utils.DockerError(
			"restart.list",
			"Failed to list containers",
			"Try running 'ork doctor' to diagnose issues",
			err,
		)
	}

	var currentContainer *docker.ContainerInfo
	for _, container := range containers {
		if container.Labels["ork.service"] == serviceName {
			currentContainer = &container
			break
		}
	}

	// If the service is not running, just start it
	if currentContainer == nil {
		ui.Info(fmt.Sprintf("%s is not running, starting it...", ui.Bold(serviceName)))
		return startSingleService(ctx, cfg, serviceName, client, networkID)
	}

	// Determine if we need to rebuild the image
	needsRebuild := forceRebuild || newServiceCfg.Build != nil

	// Stop the current container
	spinner := ui.ShowSpinner(fmt.Sprintf("Stopping %s", ui.Bold(serviceName)))
	if err := client.StopAndRemove(ctx, currentContainer.ID); err != nil {
		spinner.Error(fmt.Sprintf("Failed to stop %s", serviceName))
		return utils.DockerError(
			"restart.stop",
			fmt.Sprintf("Failed to stop service %s", serviceName),
			"Check if the container is stuck or Docker is unresponsive",
			err,
		)
	}
	spinner.Success(fmt.Sprintf("Stopped %s", ui.Bold(serviceName)))

	// TODO: Handle rebuild if needsRebuild is true (Phase 5 - build from source)
	if needsRebuild {
		ui.Warning("Build from source not yet implemented, will use image instead")
	}

	// Create and start the new container
	return startSingleService(ctx, cfg, serviceName, client, networkID)
}

// startSingleService starts a single service (helper for restart)
func startSingleService(ctx context.Context, cfg *config.Config, serviceName string, client *docker.Client, networkID string) error {
	// If we don't have a network ID, create the network
	if networkID == "" {
		spinner := ui.ShowSpinner("Creating project network...")
		var err error
		networkID, err = client.CreateNetwork(ctx, cfg.Project)
		if err != nil {
			spinner.Error("Failed to create network")
			return utils.NetworkError(
				"restart.network",
				"Failed to create project network",
				"Check if Docker is running and you have permissions",
				err,
			)
		}
		spinner.Success(fmt.Sprintf("Created network: ork-%s-network", cfg.Project))
	}

	// Create a service instance
	svc := service.New(serviceName, cfg.Project, cfg.Services[serviceName])

	// Start the service
	spinner := ui.ShowSpinner(fmt.Sprintf("Starting %s", ui.Bold(serviceName)))
	if err := svc.Start(ctx, client, networkID); err != nil {
		spinner.Error(fmt.Sprintf("Failed to start %s", serviceName))
		return utils.ServiceError(
			"restart.start",
			fmt.Sprintf("Failed to start service %s", serviceName),
			"Check logs with 'ork logs "+serviceName+"' for details",
			err,
		)
	}

	containerID := svc.GetContainerID()
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	spinner.Success(fmt.Sprintf("Started %s %s", ui.Bold(serviceName), ui.Dim(containerID)))

	return nil
}

// ============================================================================
// Private Helpers - Network Operations
// ============================================================================

// getProjectNetworkID gets the network ID for a project
func getProjectNetworkID(ctx context.Context, client *docker.Client, projectName string) (string, error) {
	return client.GetNetworkID(ctx, projectName)
}

// handleRestartError formats and displays errors with hints
func handleRestartError(err error) {
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
