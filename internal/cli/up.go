package cli

import (
	"fmt"

	"github.com/hary-singh/ork/internal/config"
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

	// Show a success message
	fmt.Printf("✅ Loaded project: %s (version %s)\n", cfg.Project, cfg.Version)
	fmt.Printf("🚀 Starting services: %v\n", serviceNames)
	fmt.Println("📦 Resolving dependencies...")
	fmt.Println("(Docker integration coming next)")

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
