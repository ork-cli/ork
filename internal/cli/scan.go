package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ork-cli/ork/internal/config"
	"github.com/ork-cli/ork/internal/git"
	"github.com/ork-cli/ork/internal/ui"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan workspace directories for git repositories",
	Long: `
Scan configured workspace directories to discover git repositories.

The scan results are cached for 24 hours to improve performance. Use --refresh to force a new scan.

Workspace directories can be configured in ~/.ork/config.yml:

  workspaces:
    - ~/code
    - ~/projects
    - ~/workspace

If no configuration exists, ork will scan default directories: ~/code, ~/projects, ~/workspace`,
	RunE: runScan,
}

const (
	bulletFormat   = "  • %s"
	tableRowFormat = "%s  %s  %s\n"
)

var (
	scanRefresh bool
)

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVar(&scanRefresh, "refresh", false, "Force a fresh scan, ignoring cache")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Load global config
	globalConfig, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Try to load from cache if not refreshing
	if !scanRefresh {
		if repos := tryLoadCache(globalConfig.Workspaces); repos != nil {
			return nil // Cache was loaded and displayed
		}
	}

	// Invalidate cache if refreshing
	if scanRefresh {
		if err := git.InvalidateCache(); err != nil {
			return fmt.Errorf("failed to invalidate cache: %w", err)
		}
	}

	// Filter and validate workspaces
	existingWorkspaces := filterExistingWorkspaces(globalConfig.Workspaces)
	if len(existingWorkspaces) == 0 {
		return handleNoWorkspaces(globalConfig.Workspaces)
	}

	// Display scanning message
	displayScanningMessage(existingWorkspaces)

	// Perform discovery
	repos, elapsed, err := performDiscovery(globalConfig.Workspaces)
	if err != nil {
		return err
	}

	// Save to cache (non-fatal if it fails)
	saveCacheIfPossible(repos)

	// Display results
	displayResults(repos, elapsed, globalConfig.Workspaces)

	return nil
}

func tryLoadCache(workspaces []string) []git.Repository {
	cached, err := git.LoadCache()
	if err == nil && cached != nil {
		ui.Success("Loaded repositories from cache")
		printRepositories(cached, workspaces)
		fmt.Println()
		fmt.Println(ui.Dim("Use 'ork scan --refresh' to force a fresh scan"))
		return cached
	}
	return nil
}

func filterExistingWorkspaces(workspaces []string) []string {
	existing := []string{}
	for _, workspace := range workspaces {
		if workspaceExists(workspace) {
			existing = append(existing, workspace)
		}
	}
	return existing
}

func workspaceExists(workspace string) bool {
	// Expand ~ to the home directory
	expandedPath := workspace
	if strings.HasPrefix(workspace, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			expandedPath = filepath.Join(home, workspace[2:])
		}
	}

	// Check if the directory exists
	_, err := os.Stat(expandedPath)
	return err == nil
}

func handleNoWorkspaces(configuredWorkspaces []string) error {
	ui.Warning("No workspace directories found")
	fmt.Println()
	fmt.Println("Configure workspaces in ~/.ork/config.yml or ensure these directories exist:")
	for _, workspace := range configuredWorkspaces {
		fmt.Println(ui.Dim(fmt.Sprintf(bulletFormat, workspace)))
	}
	return nil
}

func displayScanningMessage(workspaces []string) {
	ui.Info(fmt.Sprintf("Scanning %d workspace(s)...", len(workspaces)))
	for _, workspace := range workspaces {
		fmt.Println(ui.Dim(fmt.Sprintf(bulletFormat, workspace)))
	}
	fmt.Println()
}

func performDiscovery(workspaces []string) ([]git.Repository, time.Duration, error) {
	start := time.Now()
	repos, err := git.DiscoverRepositories(workspaces, 3)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to discover repositories: %w", err)
	}
	elapsed := time.Since(start)
	return repos, elapsed, nil
}

func saveCacheIfPossible(repos []git.Repository) {
	if err := git.SaveCache(repos); err != nil {
		ui.Warning(fmt.Sprintf("Warning: Failed to save cache: %v", err))
	}
}

func displayResults(repos []git.Repository, elapsed time.Duration, workspaces []string) {
	ui.Success(fmt.Sprintf("Found %d repositories in %v", len(repos), elapsed.Round(time.Millisecond)))
	fmt.Println()
	printRepositories(repos, workspaces)
}

func printRepositories(repos []git.Repository, workspaces []string) {
	if len(repos) == 0 {
		ui.Warning("No git repositories found")
		fmt.Println()
		fmt.Println("Make sure you have repositories in your workspace directories:")
		for _, workspace := range workspaces {
			fmt.Println(ui.Dim(fmt.Sprintf(bulletFormat, workspace)))
		}
		return
	}

	// Sort repositories by name
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	// Create header style
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	// Calculate column widths
	nameWidth := len("NAME")
	pathWidth := len("PATH")
	urlWidth := len("GIT URL")

	for _, repo := range repos {
		if len(repo.Name) > nameWidth {
			nameWidth = len(repo.Name)
		}
		if len(repo.Path) > pathWidth {
			pathWidth = len(repo.Path)
		}
		if len(repo.URL) > urlWidth {
			urlWidth = len(repo.URL)
		}
	}

	// Limit max widths
	if nameWidth > 30 {
		nameWidth = 30
	}
	if pathWidth > 60 {
		pathWidth = 60
	}
	if urlWidth > 60 {
		urlWidth = 60
	}

	// Print header (pad first, then style)
	fmt.Printf(tableRowFormat,
		headerStyle.Render(padRight("NAME", nameWidth)),
		headerStyle.Render(padRight("PATH", pathWidth)),
		headerStyle.Render(padRight("GIT URL", urlWidth)))

	// Print separator
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	fmt.Printf(tableRowFormat,
		separatorStyle.Render(repeatChar("─", nameWidth)),
		separatorStyle.Render(repeatChar("─", pathWidth)),
		separatorStyle.Render(repeatChar("─", urlWidth)))

	// Print repositories
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	for _, repo := range repos {
		name := truncate(repo.Name, nameWidth)
		path := truncate(repo.Path, pathWidth)
		url := truncate(repo.URL, urlWidth)

		// Pad first, then style - this keeps alignment correct
		fmt.Printf(tableRowFormat,
			nameStyle.Render(padRight(name, nameWidth)),
			pathStyle.Render(padRight(path, pathWidth)),
			urlStyle.Render(padRight(url, urlWidth)))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + repeatChar(" ", width-len(s))
}

func repeatChar(char string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result
}
