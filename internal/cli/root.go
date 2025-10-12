package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (set via build flags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersionInfo sets the version information from build flags
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = buildVersionString()
}

// buildVersionString creates a detailed version string
func buildVersionString() string {
	result := version
	if commit != "none" && commit != "" {
		result += fmt.Sprintf(" (commit: %s)", commit)
	}
	if date != "unknown" && date != "" {
		result += fmt.Sprintf(" (built: %s)", date)
	}
	return result
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ork",
	Short: "Ork - Microservices orchestration made easy",
	Long: `Ork is a modern microservices orchestration tool that makes Docker Compose not suck.

	Run services from anywhere, intelligently manage dependencies, and enjoy beautiful CLI output.`,
	Version: version,
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
