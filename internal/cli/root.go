package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (will be set via build flags later)
var Version = "0.0.1-dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ork",
	Short: "Ork - Microservices orchestration made easy",
	Long: `Ork is a modern microservices orchestration tool that makes Docker Compose not suck.

	Run services from anywhere, intelligently manage dependencies, and enjoy beautiful CLI output.`,
	Version: Version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
