package cli

import (
	"context"
	"fmt"

	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/docker"
	"github.com/ork-cli/ork/internal/ui"
	"github.com/spf13/cobra"
)

// ============================================================================
// Cobra Command Definition
// ============================================================================

var logsCmd = &cobra.Command{
	Use:   "logs <service>",
	Short: "View logs from a service",
	Long: `
View and stream logs from a running service container.

By default, shows all available logs. Use --tail to limit output,
and --follow to stream logs continuously (like tail -f).`,
	Example: `
ork logs api                 Show all logs for api service
ork logs api --follow        Stream logs continuously
ork logs api --tail 100      Show last 100 lines
ork logs api --timestamps    Show timestamps in output`,

	Args: cobra.ExactArgs(1), // Require exactly one service name
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]

		// Get flags
		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetString("tail")
		timestamps, _ := cmd.Flags().GetBool("timestamps")

		if err := runLogs(serviceName, follow, tail, timestamps); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
	},
}

func init() {
	// Register the 'logs' command with the root command
	rootCmd.AddCommand(logsCmd)

	// Add flags
	logsCmd.Flags().BoolP("follow", "f", false, "Stream logs continuously (like tail -f)")
	logsCmd.Flags().StringP("tail", "n", "all", "Number of lines to show from the end")
	logsCmd.Flags().BoolP("timestamps", "t", false, "Show timestamps in log output")
}

// ============================================================================
// Main Orchestrator
// ============================================================================

// runLogs retrieves and displays logs for a specific service
func runLogs(serviceName string, follow bool, tail string, timestamps bool) error {
	// Load configuration to get the project name
	cfg, err := loadConfigForLogs()
	if err != nil {
		return err
	}

	// Create a Docker client
	dockerClient, err := createDockerClientForLogs()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			fmt.Printf("❌ Error closing Docker client: %v\n", closeErr)
		}
	}()

	// Find the container for this service
	ctx := context.Background()
	containerID, err := findContainerForService(ctx, dockerClient, cfg.Project, serviceName)
	if err != nil {
		return err
	}

	// Print a beautiful service header
	header := ui.FormatServiceHeader(serviceName, containerID, follow)
	fmt.Println(header)
	ui.EmptyLine()

	// Create a formatter that applies log level coloring
	logFormatter := func(line string) string {
		return ui.FormatLogLine(line, timestamps)
	}

	// Build log options with formatter
	logOpts := docker.LogsOptions{
		Follow:     follow,
		Tail:       tail,
		Timestamps: timestamps,
		Formatter:  logFormatter,
	}

	// Stream logs
	if err := dockerClient.Logs(ctx, containerID, logOpts); err != nil {
		return fmt.Errorf("failed to retrieve logs: %w", err)
	}

	// Show streaming footer if following
	if follow {
		fmt.Println(ui.FormatStreamingFooter())
	}

	return nil
}

// ============================================================================
// Private Helpers - Configuration
// ============================================================================

// loadConfigForLogs loads the ork.yml file
func loadConfigForLogs() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\n💡 Make sure ork.yml exists in current directory", err)
	}
	return cfg, nil
}

// ============================================================================
// Private Helpers - Docker Operations
// ============================================================================

// createDockerClientForLogs creates a Docker client
func createDockerClientForLogs() (*docker.Client, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return client, nil
}

// ============================================================================
// Private Helpers - Service Discovery
// ============================================================================

// findContainerForService finds the container ID for a given service name
func findContainerForService(ctx context.Context, client *docker.Client, projectName, serviceName string) (string, error) {
	// List all containers for this project
	containers, err := client.List(ctx, projectName)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Find the container matching the service name
	for _, container := range containers {
		if container.Labels["ork.service"] == serviceName {
			return container.ID, nil
		}
	}

	// Service not found
	return "", fmt.Errorf("service '%s' not found\n💡 Use 'ork ps' to see running services", serviceName)
}
