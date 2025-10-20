package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// ============================================================================
// Type Definitions
// ============================================================================

// Repository represents a discovered git repository
type Repository struct {
	Name string // Repository name (e.g., "frontend", "api")
	Path string // Absolute path to the repository
	URL  string // Git remote URL (e.g., "github.com/org/repo")
}

// ============================================================================
// Public Discovery API
// ============================================================================

// DiscoverRepositories scans workspace directories and finds git repositories.
// It searches up to maxDepth levels deep (default: 3 if maxDepth <= 0).
// Automatically skips hidden directories (except .ork), node_modules, vendor, dist, and build.
//
// Parameters:
//   - workspaceDirs: List of directories to scan (supports ~ for home directory)
//   - maxDepth: Maximum directory depth to search (0 or negative uses default of 3)
//
// Returns:
//   - Deduplicated list of discovered repositories
//   - Error if scanning fails
//
// Example:
//
//	workspaces := []string{"~/code", "~/projects"}
//	repos, err := DiscoverRepositories(workspaces, 3)
//	if err != nil {
//	    return err
//	}
//	for _, repo := range repos {
//	    fmt.Printf("%s: %s\n", repo.Name, repo.Path)
//	}
func DiscoverRepositories(workspaceDirs []string, maxDepth int) ([]Repository, error) {
	if maxDepth <= 0 {
		maxDepth = 3 // Default depth
	}

	var repos []Repository
	seen := make(map[string]bool) // Track repos we've already found

	for _, workspace := range workspaceDirs {
		expandedPath := expandHomePath(workspace)
		if !directoryExists(expandedPath) {
			continue
		}

		found, err := scanDirectory(expandedPath, 0, maxDepth)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace %s: %w", workspace, err)
		}

		repos = deduplicateRepos(repos, found, seen)
	}

	return repos, nil
}

// ============================================================================
// Internal Helper Functions - Path Operations
// ============================================================================

// expandHomePath expands ~ to the home directory
func expandHomePath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}

// directoryExists checks if a directory exists
func directoryExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// deduplicateRepos adds new repos to the list, skipping duplicates
func deduplicateRepos(existing, found []Repository, seen map[string]bool) []Repository {
	for _, repo := range found {
		if !seen[repo.Path] {
			existing = append(existing, repo)
			seen[repo.Path] = true
		}
	}
	return existing
}

// ============================================================================
// Internal Helper Functions - Directory Scanning
// ============================================================================

// scanDirectory recursively searches for git repositories up to maxDepth
func scanDirectory(dir string, currentDepth, maxDepth int) ([]Repository, error) {
	if currentDepth > maxDepth {
		return []Repository{}, nil
	}

	if isGitRepository(dir) {
		return handleGitRepository(dir)
	}

	return scanSubdirectories(dir, currentDepth, maxDepth)
}

// handleGitRepository creates a repository entry for a git directory
func handleGitRepository(dir string) ([]Repository, error) {
	repo, err := createRepository(dir)
	if err != nil {
		return []Repository{}, nil
	}
	return []Repository{repo}, nil
}

// scanSubdirectories recursively scans subdirectories for git repositories
func scanSubdirectories(dir string, currentDepth, maxDepth int) ([]Repository, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Repository{}, nil // Permission denied or other errors
	}

	var repos []Repository
	for _, entry := range entries {
		if shouldSkipDirectory(entry) {
			continue
		}

		subdirPath := filepath.Join(dir, entry.Name())
		found, err := scanDirectory(subdirPath, currentDepth+1, maxDepth)
		if err != nil {
			continue
		}

		repos = append(repos, found...)
	}

	return repos, nil
}

// shouldSkipDirectory determines if a directory should be skipped during scanning
func shouldSkipDirectory(entry os.DirEntry) bool {
	if !entry.IsDir() {
		return true
	}

	name := entry.Name()

	// Skip hidden directories (except .ork)
	if strings.HasPrefix(name, ".") && name != ".ork" {
		return true
	}

	// Skip common non-code directories
	skipDirs := []string{"node_modules", "vendor", "dist", "build"}
	for _, skipDir := range skipDirs {
		if name == skipDir {
			return true
		}
	}

	return false
}

// ============================================================================
// Internal Helper Functions - Git Operations
// ============================================================================

// isGitRepository checks if a directory is a git repository
func isGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// createRepository creates a Repository struct from a git repository path
func createRepository(path string) (Repository, error) {
	// Open the repository
	repo, err := git.PlainOpen(path)
	if err != nil {
		return Repository{}, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get the repository name (last component of the path)
	name := filepath.Base(path)

	// Try to get the remote URL
	url := ""
	remotes, err := repo.Remotes()
	if err == nil && len(remotes) > 0 {
		// Get the first remote (usually "origin")
		remote := remotes[0]
		if len(remote.Config().URLs) > 0 {
			url = normalizeGitURL(remote.Config().URLs[0])
		}
	}

	return Repository{
		Name: name,
		Path: path,
		URL:  url,
	}, nil
}

// normalizeGitURL converts a git URL to a normalized form (e.g., github.com/org/repo)
// Handles both SSH and HTTPS URLs
func normalizeGitURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URLs (git@github.com:org/repo)
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		return url
	}

	// Handle HTTPS URLs (https://github.com/org/repo)
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
		return url
	}

	// Handle HTTP URLs (http://github.com/org/repo)
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
		return url
	}

	return url
}

// ============================================================================
// Public Repository Lookup
// ============================================================================

// FindRepository searches for a repository by git URL in the discovered repos.
// The URL is normalized before comparison, so it works with both SSH and HTTPS URLs.
//
// Example:
//
//	repos, _ := DiscoverRepositories(workspaces, 3)
//	repo := FindRepository(repos, "github.com/user/project")
//	if repo != nil {
//	    fmt.Println("Found at:", repo.Path)
//	}
func FindRepository(repos []Repository, gitURL string) *Repository {
	normalized := normalizeGitURL(gitURL)

	for i := range repos {
		if repos[i].URL == normalized {
			return &repos[i]
		}
	}

	return nil
}
