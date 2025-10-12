package cli

import (
	"context"
	"fmt"

	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/docker"
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
			fmt.Printf("❌ Error: %v\n", err)
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
			fmt.Printf("❌ Error closing Docker client: %v\n", closeErr)
		}
	}()

	// List all containers for this project
	ctx := context.Background()
	containers, err := dockerClient.List(ctx, cfg.Project)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Printf("No services running for project '%s'\n", cfg.Project)
		return nil
	}

	// Filter containers if specific services requested
	containersToStop := filterContainersByService(containers, serviceNames)

	if len(containersToStop) == 0 {
		if len(serviceNames) > 0 {
			fmt.Printf("No matching services found: %v\n", serviceNames)
			fmt.Printf("💡 Use 'ork ps' to see running services\n")
		} else {
			fmt.Printf("No services running for project '%s'\n", cfg.Project)
		}
		return nil
	}

	// Stop (and optionally remove) containers
	if err := stopContainers(ctx, dockerClient, containersToStop, keepContainers); err != nil {
		return err
	}

	fmt.Printf("✅ Successfully stopped %d service(s)\n", len(containersToStop))

	// Clean up the network if we stopped all services
	if len(serviceNames) == 0 && len(containersToStop) == len(containers) {
		// All services have been stopped, remove the network
		if err := dockerClient.DeleteNetwork(ctx, cfg.Project); err != nil {
			fmt.Printf("⚠️  Warning: failed to remove network: %v\n", err)
		} else {
			fmt.Printf("🌐 Removed network: ork-%s-network\n", cfg.Project)
		}
	}

	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadConfigForDown loads the ork.yml file
func loadConfigForDown() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\n💡 Make sure ork.yml exists in current directory", err)
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
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
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
			fmt.Printf("⏸️  Stopping %s...\n", serviceName)
			if err := client.Stop(ctx, container.ID); err != nil {
				fmt.Printf("⚠️  Warning: failed to stop %s: %v\n", serviceName, err)
				continue
			}
			fmt.Printf("✅ Stopped %s\n", serviceName)
		} else {
			// Stop and remove the container
			fmt.Printf("🛑 Stopping %s...\n", serviceName)
			if err := client.StopAndRemove(ctx, container.ID); err != nil {
				fmt.Printf("⚠️  Warning: failed to stop/remove %s: %v\n", serviceName, err)
				continue
			}
			fmt.Printf("✅ Stopped and removed %s\n", serviceName)
		}
	}

	return nil
}
