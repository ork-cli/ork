package cli

import (
	"context"
	"fmt"

	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/docker"
	"github.com/ork-cli/ork/internal/ui"
	"github.com/ork-cli/ork/pkg/utils"
	"github.com/spf13/cobra"
)

// ============================================================================
// Cobra Command Definition
// ============================================================================

var downCmd = &cobra.Command{
	Use:   "down [service...]",
	Short: "Stop services",
	Long: `Stop one or more services managed by Ork.

	If no services are specified, stops all services for the current project.
	By default, stopped containers are removed to keep your system clean.`,
	Example: `  ork down                     Stop all services in current project
  	ork down redis               Stop specific service
  	ork down redis postgres      Stop multiple services
  	ork down --keep              Stop but keep containers for debugging`,

	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		keepContainers, _ := cmd.Flags().GetBool("keep")

		if err := runDown(args, keepContainers); err != nil {
			handleDownError(err)
			return
		}
	},
}

func init() {
	// Register the 'down' command with the root command
	rootCmd.AddCommand(downCmd)

	// Add flags
	downCmd.Flags().Bool("keep", false, "Keep stopped containers (don't remove)")
}

// ============================================================================
// Main Orchestrator
// ============================================================================

// runDown stops (and optionally removes) Ork-managed containers
func runDown(serviceNames []string, keepContainers bool) error {
	// Load configuration to get the project name
	cfg, err := loadConfigForDown()
	if err != nil {
		return err
	}

	// Create a Docker client
	dockerClient, err := createDockerClientForDown()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			ui.Warning(fmt.Sprintf("Failed to close Docker client: %v", closeErr))
		}
	}()

	// List all containers for this project
	ctx := context.Background()
	containers, err := dockerClient.List(ctx, cfg.Project)
	if err != nil {
		return utils.DockerError(
			"down.list",
			"Failed to list containers",
			"Try running 'ork doctor' to diagnose issues",
			err,
		)
	}

	if len(containers) == 0 {
		ui.Info(fmt.Sprintf("No services running for project: %s", ui.Bold(cfg.Project)))
		return nil
	}

	// Filter containers if specific services requested
	containersToStop := filterContainersByService(containers, serviceNames)

	if len(containersToStop) == 0 {
		if len(serviceNames) > 0 {
			ui.Warning(fmt.Sprintf("No matching services found: %v", serviceNames))
			ui.Hint("Use 'ork ps' to see running services")
		} else {
			ui.Info(fmt.Sprintf("No services running for project: %s", ui.Bold(cfg.Project)))
		}
		return nil
	}

	// Show what we're stopping
	ui.EmptyLine()
	ui.Info(fmt.Sprintf("Stopping %d service(s) for project: %s", len(containersToStop), ui.Bold(cfg.Project)))
	ui.EmptyLine()

	// Stop (and optionally remove) containers
	if err := stopContainers(ctx, dockerClient, containersToStop, keepContainers); err != nil {
		return err
	}

	// Clean up the network if we stopped all services
	if len(serviceNames) == 0 && len(containersToStop) == len(containers) {
		// All services have been stopped, remove the network
		spinner := ui.ShowSpinner("Cleaning up project network...")
		if err := dockerClient.DeleteNetwork(ctx, cfg.Project); err != nil {
			spinner.Warning(fmt.Sprintf("Failed to remove network: %v", err))
		} else {
			spinner.Success(fmt.Sprintf("Removed network: ork-%s-network", cfg.Project))
		}
	}

	ui.EmptyLine()
	ui.SuccessBox(fmt.Sprintf("Successfully stopped %d service(s)", len(containersToStop)))
	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadConfigForDown loads the ork.yml file
func loadConfigForDown() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, utils.ConfigError(
			"down.load",
			"Failed to load configuration",
			"Make sure ork.yml exists in the current directory",
			err,
		)
	}
	return cfg, nil
}

// ============================================================================
// Private Helpers - Docker Operations
// ============================================================================

// createDockerClientForDown creates a Docker client
func createDockerClientForDown() (*docker.Client, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, utils.DockerError(
			"down.docker",
			"Failed to connect to Docker",
			"Make sure Docker is running. Try 'docker ps' or run 'ork doctor'",
			err,
		)
	}
	return client, nil
}

// ============================================================================
// Private Helpers - Filtering
// ============================================================================

// filterContainersByService filters containers by service name
// If no service names provided, returns all containers
func filterContainersByService(containers []docker.ContainerInfo, serviceNames []string) []docker.ContainerInfo {
	// If no service names specified, return all containers
	if len(serviceNames) == 0 {
		return containers
	}

	// Create a set of requested service names for a quick lookup
	serviceSet := make(map[string]bool)
	for _, name := range serviceNames {
		serviceSet[name] = true
	}

	// Filter containers
	filtered := make([]docker.ContainerInfo, 0)
	for _, container := range containers {
		serviceName := container.Labels["ork.service"]
		if serviceSet[serviceName] {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

// ============================================================================
// Private Helpers - Stopping
// ============================================================================

// stopContainers stops (and optionally removes) the given containers
func stopContainers(ctx context.Context, client *docker.Client, containers []docker.ContainerInfo, keepContainers bool) error {
	for _, container := range containers {
		serviceName := container.Labels["ork.service"]

		if keepContainers {
			// Just stop the container
			spinner := ui.ShowSpinner(fmt.Sprintf("Stopping %s", ui.Bold(serviceName)))
			if err := client.Stop(ctx, container.ID); err != nil {
				spinner.Warning(fmt.Sprintf("Failed to stop %s: %v", serviceName, err))
				continue
			}
			spinner.Success(fmt.Sprintf("Stopped %s", ui.Bold(serviceName)))
		} else {
			// Stop and remove the container
			spinner := ui.ShowSpinner(fmt.Sprintf("Stopping %s", ui.Bold(serviceName)))
			if err := client.StopAndRemove(ctx, container.ID); err != nil {
				spinner.Warning(fmt.Sprintf("Failed to stop/remove %s: %v", serviceName, err))
				continue
			}
			spinner.Success(fmt.Sprintf("Stopped and removed %s", ui.Bold(serviceName)))
		}
	}

	return nil
}

// handleDownError formats and displays errors with hints
func handleDownError(err error) {
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
	} else {
		// Fallback for non-Ork errors
		ui.Error(fmt.Sprintf("Error: %v", err))
	}
}
