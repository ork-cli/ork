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

Workspace directories can be configured in ~/.ork/config.yml:

  workspaces:
    - ~/code
    - ~/projects
    - ~/workspace

If no configuration exists, ork will scan default directories: ~/code, ~/projects, ~/workspace`,
	RunE: runScan,
}

const (
	bulletFormat        = "  • %s"
	tableRowFormat      = "%s  %s  %s\n"
	detailedTableFormat = "%s  %s  %s  %s  %s\n"
	noReposMessage      = "No git repositories found"
	workspaceConfigMsg  = "Make sure you have repositories in your workspace directories:"
	scanDepth           = 3

	// Column width limits
	maxNameWidth   = 25
	maxPathWidth   = 40
	maxBranchWidth = 35
	maxStatusWidth = 30
	minBranchWidth = 15
)

// detailedColumnWidths holds the column widths for a detailed view
type detailedColumnWidths struct {
	name   int
	path   int
	branch int
	commit int
	status int
}

// detailedStyles holds all the lipgloss styles for a detailed view
type detailedStyles struct {
	header    lipgloss.Style
	separator lipgloss.Style
	name      lipgloss.Style
	path      lipgloss.Style
	branch    lipgloss.Style
	commit    lipgloss.Style
	clean     lipgloss.Style
	dirty     lipgloss.Style
}

var (
	scanDetailed bool
)

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVarP(&scanDetailed, "detailed", "d", false, "Show detailed git state (branch, commit, changes)")
}

// ============================================================================
// Main Command Logic
// ============================================================================

func runScan(_ *cobra.Command, _ []string) error {
	// Load global config
	globalConfig, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
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

	// Display results
	displayResults(repos, elapsed, globalConfig.Workspaces)

	return nil
}

// ============================================================================
// Workspace Management
// ============================================================================

func filterExistingWorkspaces(workspaces []string) []string {
	var existing []string
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
	printWorkspaceList(configuredWorkspaces)
	return nil
}

func printWorkspaceList(workspaces []string) {
	for _, workspace := range workspaces {
		fmt.Println(ui.Dim(fmt.Sprintf(bulletFormat, workspace)))
	}
}

func displayScanningMessage(workspaces []string) {
	ui.Info(fmt.Sprintf("Scanning %d workspace(s)...", len(workspaces)))
	printWorkspaceList(workspaces)
	fmt.Println()
}

// ============================================================================
// Repository Discovery
// ============================================================================

func performDiscovery(workspaces []string) ([]git.Repository, time.Duration, error) {
	start := time.Now()
	repos, err := git.DiscoverRepositories(workspaces, scanDepth)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to discover repositories: %w", err)
	}
	elapsed := time.Since(start)
	return repos, elapsed, nil
}

func displayResults(repos []git.Repository, elapsed time.Duration, workspaces []string) {
	ui.Success(fmt.Sprintf("Found %d repositories in %v", len(repos), elapsed.Round(time.Millisecond)))
	fmt.Println()
	printRepositories(repos, workspaces)
}

// ============================================================================
// Output Formatting - Basic View
// ============================================================================

func printRepositories(repos []git.Repository, workspaces []string) {
	if len(repos) == 0 {
		ui.Warning(noReposMessage)
		fmt.Println()
		fmt.Println(workspaceConfigMsg)
		printWorkspaceList(workspaces)
		return
	}

	// Sort repositories by name
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	// Use the detailed view if a flag is set
	if scanDetailed {
		printDetailedRepositories(repos)
		return
	}

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

// ============================================================================
// Utility Functions
// ============================================================================

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Output Formatting - Detailed View
// ============================================================================

// printDetailedRepositories displays repositories with git state information
func printDetailedRepositories(repos []git.Repository) {
	styles := createDetailedStyles()
	widths := calculateDetailedColumnWidths(repos)
	printDetailedHeader(styles, widths)
	printDetailedRows(repos, styles, widths)
}

// createDetailedStyles creates all lipgloss styles for the detailed view
func createDetailedStyles() detailedStyles {
	return detailedStyles{
		header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		separator: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		name:      lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
		path:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		branch:    lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
		commit:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		clean:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		dirty:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	}
}

// calculateDetailedColumnWidths calculates optimal column widths based on content
func calculateDetailedColumnWidths(repos []git.Repository) detailedColumnWidths {
	widths := detailedColumnWidths{
		name:   len("NAME"),
		path:   len("PATH"),
		branch: minBranchWidth,
		commit: len("COMMIT"),
		status: len("STATUS"),
	}

	// Calculate based on actual data
	for _, repo := range repos {
		widths.name = maxInt(widths.name, len(repo.Name))
		widths.path = maxInt(widths.path, len(repo.Path))

		if state, err := git.GetRepoState(repo.Path); err == nil {
			widths.branch = maxInt(widths.branch, len(state.Branch))
			widths.status = maxInt(widths.status, len(state.UncommittedSummary))
		}
	}

	// Apply max limits
	widths.name = minInt(widths.name, maxNameWidth)
	widths.path = minInt(widths.path, maxPathWidth)
	widths.branch = minInt(widths.branch, maxBranchWidth)
	widths.status = minInt(widths.status, maxStatusWidth)

	return widths
}

// printDetailedHeader prints the header row for a detailed view
func printDetailedHeader(styles detailedStyles, widths detailedColumnWidths) {
	// Print header
	fmt.Printf(detailedTableFormat,
		styles.header.Render(padRight("NAME", widths.name)),
		styles.header.Render(padRight("PATH", widths.path)),
		styles.header.Render(padRight("BRANCH", widths.branch)),
		styles.header.Render(padRight("COMMIT", widths.commit)),
		styles.header.Render(padRight("STATUS", widths.status)))

	// Print separator
	fmt.Printf(detailedTableFormat,
		styles.separator.Render(repeatChar("─", widths.name)),
		styles.separator.Render(repeatChar("─", widths.path)),
		styles.separator.Render(repeatChar("─", widths.branch)),
		styles.separator.Render(repeatChar("─", widths.commit)),
		styles.separator.Render(repeatChar("─", widths.status)))
}

// printDetailedRows prints all repository rows with git state
func printDetailedRows(repos []git.Repository, styles detailedStyles, widths detailedColumnWidths) {
	for _, repo := range repos {
		printDetailedRow(repo, styles, widths)
	}
}

// printDetailedRow prints a single repository row with git state
func printDetailedRow(repo git.Repository, styles detailedStyles, widths detailedColumnWidths) {
	state, err := git.GetRepoState(repo.Path)
	if err != nil {
		printDetailedErrorRow(repo, err.Error(), styles, widths)
		return
	}

	statusStyle := styles.clean
	if state.HasUncommitted {
		statusStyle = styles.dirty
	}

	fmt.Printf(detailedTableFormat,
		styles.name.Render(padRight(truncate(repo.Name, widths.name), widths.name)),
		styles.path.Render(padRight(truncate(repo.Path, widths.path), widths.path)),
		styles.branch.Render(padRight(truncate(state.Branch, widths.branch), widths.branch)),
		styles.commit.Render(padRight(state.CommitHash, widths.commit)),
		statusStyle.Render(state.UncommittedSummary))
}

// printDetailedErrorRow prints an error row for a repository that failed to load
func printDetailedErrorRow(repo git.Repository, errMsg string, styles detailedStyles, widths detailedColumnWidths) {
	fmt.Printf(tableRowFormat,
		styles.name.Render(padRight(truncate(repo.Name, widths.name), widths.name)),
		styles.path.Render(padRight(truncate(repo.Path, widths.path), widths.path)),
		styles.dirty.Render("error: "+errMsg))
}
