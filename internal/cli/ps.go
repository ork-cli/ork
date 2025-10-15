package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/docker"
	"github.com/ork-cli/ork/internal/ui"
	"github.com/ork-cli/ork/pkg/utils"
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
			handlePSError(err)
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
		return utils.DockerError(
			"ps.list",
			"Failed to list containers",
			"Try running 'ork doctor' to diagnose issues",
			err,
		)
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
		return nil, utils.ConfigError(
			"ps.load",
			"Failed to load configuration",
			"Make sure ork.yml exists in current directory",
			err,
		)
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
		return nil, utils.DockerError(
			"ps.docker",
			"Failed to connect to Docker",
			"Make sure Docker is running with 'docker ps' or run 'ork doctor'",
			err,
		)
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

// displayContainers prints containers in a beautiful table format
func displayContainers(containers []docker.ContainerInfo, projectName string) {
	// Convert containers to table rows
	var rows []ui.ServiceRow
	for _, c := range containers {
		serviceName := extractServiceName(c.Labels)
		status := normalizeStatus(c.Status)
		uptime := extractUptime(c.Status)

		rows = append(rows, ui.ServiceRow{
			Service:     serviceName,
			Status:      status,
			Ports:       c.Ports,
			ContainerID: c.ID,
			Uptime:      uptime,
		})
	}

	// Render beautiful table
	table := ui.ServiceTable(projectName, rows)
	fmt.Print(table)
}

// extractServiceName gets the service name from labels
func extractServiceName(labels map[string]string) string {
	if serviceName, exists := labels["ork.service"]; exists {
		return serviceName
	}
	return "unknown"
}

// normalizeStatus converts Docker status to our normalized format
func normalizeStatus(status string) string {
	if strings.HasPrefix(status, "Up") {
		return "running"
	} else if strings.HasPrefix(status, "Exited") {
		return "stopped"
	} else if strings.Contains(strings.ToLower(status), "restarting") {
		return "starting"
	}
	return "stopped"
}

// extractUptime extracts uptime from Docker status string
func extractUptime(status string) string {
	// Docker status format: "Up 2 hours" or "Up 5 minutes" or "Exited (0) 2 hours ago"
	if strings.HasPrefix(status, "Up ") {
		// Extract the time portion
		uptime := strings.TrimPrefix(status, "Up ")
		// Clean up any parenthetical info
		if idx := strings.Index(uptime, "("); idx != -1 {
			uptime = strings.TrimSpace(uptime[:idx])
		}
		return uptime
	}
	return ""
}

// handlePSError formats and displays errors with hints
func handlePSError(err error) {
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
