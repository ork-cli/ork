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

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running services",
	Long: `List all services managed by Ork for the current project.

	Shows container status, ports, and other information for all services
	defined in your ork.yml configuration file.`,
	Example: `  ork ps                       List all services in current project
  	ork ps --all                 Include stopped containers`,

	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		showAll, _ := cmd.Flags().GetBool("all")

		if err := runPS(showAll); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
	},
}

func init() {
	// Register the 'ps' command with the root command
	rootCmd.AddCommand(psCmd)

	// Add flags
	psCmd.Flags().BoolP("all", "a", false, "Show all containers (including stopped)")
}

// ============================================================================
// Main Orchestrator
// ============================================================================

// runPS lists all Ork-managed containers for the current project
func runPS(showAll bool) error {
	// Load configuration to get the project name
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create a Docker client
	dockerClient, err := createDockerClientForPS()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			fmt.Printf("❌ Error closing Docker client: %v\n", closeErr)
		}
	}()

	// List containers
	ctx := context.Background()
	containers, err := dockerClient.List(ctx, cfg.Project)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter out stopped containers if --all not specified
	if !showAll {
		containers = filterRunningContainers(containers)
	}

	// Display results
	displayContainers(containers, cfg.Project)

	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadConfig loads the ork.yml file (validation not required for ps)
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\n💡 Make sure ork.yml exists in current directory", err)
	}
	return cfg, nil
}

// ============================================================================
// Private Helpers - Docker Operations
// ============================================================================

// createDockerClientForPS creates a Docker client
func createDockerClientForPS() (*docker.Client, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return client, nil
}

// ============================================================================
// Private Helpers - Filtering
// ============================================================================

// filterRunningContainers filters out stopped containers
func filterRunningContainers(containers []docker.ContainerInfo) []docker.ContainerInfo {
	running := make([]docker.ContainerInfo, 0, len(containers))

	for _, container := range containers {
		// Docker status starts with "Up" for running containers
		if strings.HasPrefix(container.Status, "Up") {
			running = append(running, container)
		}
	}

	return running
}

// ============================================================================
// Private Helpers - Display
// ============================================================================

// displayContainers prints containers in a nice table format
func displayContainers(containers []docker.ContainerInfo, projectName string) {
	if len(containers) == 0 {
		fmt.Printf("No containers found for project '%s'\n", projectName)
		fmt.Printf("💡 Start services with: ork up <service>\n")
		return
	}

	// Print header
	fmt.Printf("Services for project: %s\n\n", projectName)
	fmt.Printf("%-20s %-15s %-30s %-20s\n", "SERVICE", "STATUS", "PORTS", "CONTAINER ID")
	fmt.Printf("%s\n", strings.Repeat("-", 85))

	// Print each container
	for _, c := range containers {
		serviceName := extractServiceName(c.Labels)
		status := formatStatus(c.Status)
		ports := formatPortsList(c.Ports)

		fmt.Printf("%-20s %-15s %-30s %-20s\n",
			serviceName,
			status,
			ports,
			c.ID,
		)
	}

	fmt.Printf("\n")
}

// extractServiceName gets the service name from labels
func extractServiceName(labels map[string]string) string {
	if serviceName, exists := labels["ork.service"]; exists {
		return serviceName
	}
	return "unknown"
}

// formatStatus formats the container status with color indicators
func formatStatus(status string) string {
	if strings.HasPrefix(status, "Up") {
		return "🟢 " + status
	}
	return "🔴 " + status
}

// formatPortsList formats the port list for display
func formatPortsList(ports []string) string {
	if len(ports) == 0 {
		return "-"
	}

	// If there are many ports, just show the first few
	if len(ports) > 2 {
		return strings.Join(ports[:2], ", ") + "..."
	}

	return strings.Join(ports, ", ")
}
