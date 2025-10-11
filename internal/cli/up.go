package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
		// This is where the actual logic will go
		fmt.Printf("🚀 Starting services: %v\n", args)
		fmt.Println("📦 Resolving dependencies...")
		fmt.Println("(Not implemented yet)")
	},
}

func init() {
	// Register the 'up' command with the root command
	rootCmd.AddCommand(upCmd)

	// Add flags (options) to the command
	upCmd.Flags().Bool("local", false, "Build and run from local source")
	upCmd.Flags().Bool("dev", false, "Use development registry images")
}
