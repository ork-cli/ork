package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// ============================================================================
// Constants
// ============================================================================

// Error message constants
const (
	errOpenRepository   = "failed to open repository: %w"
	errGetHead          = "failed to get HEAD: %w"
	errGetWorktree      = "failed to get worktree: %w"
	errGetStatus        = "failed to get status: %w"
	errGetRemotes       = "failed to get remotes: %w"
	errGetCommit        = "failed to get commit: %w"
	errGetLocalCommit   = "failed to get local commit: %w"
	errGetRemoteCommit  = "failed to get remote commit: %w"
	errNoRemotes        = "no remotes found"
	errNoRemoteURLs     = "remote has no URLs"
	errDetachedHead     = "repository is in detached HEAD state"
	errCheckUncommitted = "failed to check for uncommitted changes: %w"
)

// State constants
const (
	stateDetachedHead = "detached HEAD"
	stateNoCommits    = "no commits"
	stateClean        = "clean"
)

// ============================================================================
// Type Definitions
// ============================================================================

// RepoState represents the current state of a git repository
type RepoState struct {
	Exists             bool   // Whether the repository exists at the given path
	Branch             string // Current branch name (e.g., "main")
	CommitHash         string // Short commit hash (e.g., "a1b2c3d")
	CommitHashFull     string // Full commit hash
	HasUncommitted     bool   // Whether there are uncommitted changes
	UncommittedSummary string // Summary of uncommitted changes (e.g., "2 modified, 1 untracked")
}

// ============================================================================
// Internal Helper Functions
// ============================================================================

// openRepo opens a git repository and returns it or an error
func openRepo(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf(errOpenRepository, err)
	}
	return repo, nil
}

// getHead returns the HEAD reference of a repository
func getHead(repo *git.Repository) (*plumbing.Reference, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf(errGetHead, err)
	}
	return head, nil
}

// ============================================================================
// Core Repository State Functions
// ============================================================================

// GetRepoState returns comprehensive state information about a git repository.
// This is the primary function for getting all repository information in a single call.
// Returns a RepoState with Exists=false if the path is not a git repository.
//
// Example:
//
//	state, err := GetRepoState("/path/to/repo")
//	if err != nil {
//	    return err
//	}
//	if !state.Exists {
//	    fmt.Println("Not a git repository")
//	    return
//	}
//	fmt.Printf("Branch: %s, Commit: %s, Status: %s\n",
//	    state.Branch, state.CommitHash, state.UncommittedSummary)
func GetRepoState(path string) (*RepoState, error) {
	state := &RepoState{
		Exists: false,
	}

	// Check if the directory exists
	if !RepoExistsAt(path) {
		return state, nil
	}

	state.Exists = true

	// Get current branch
	branch, err := GetCurrentBranch(path)
	if err != nil {
		// Not a fatal error - might be in a detached HEAD state
		state.Branch = stateDetachedHead
	} else {
		state.Branch = branch
	}

	// Get commit hash
	hash, fullHash, err := GetCommitHash(path)
	if err != nil {
		// Not a fatal error - might be a new repository with no commits
		state.CommitHash = stateNoCommits
		state.CommitHashFull = ""
	} else {
		state.CommitHash = hash
		state.CommitHashFull = fullHash
	}

	// Check for uncommitted changes
	hasChanges, summary, err := HasUncommittedChanges(path)
	if err != nil {
		return state, fmt.Errorf(errCheckUncommitted, err)
	}
	state.HasUncommitted = hasChanges
	state.UncommittedSummary = summary

	return state, nil
}

// RepoExistsAt checks if a git repository exists at the given path
func RepoExistsAt(path string) bool {
	// Expand the home path if needed
	expandedPath := expandHomePath(path)

	// Check if the directory exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return false
	}

	// Check if the .git directory exists
	gitDir := filepath.Join(expandedPath, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// ============================================================================
// Branch and Commit Operations
// ============================================================================

// GetCurrentBranch returns the name of the current branch.
// Returns an error if the repository is in a detached HEAD state or if the branch cannot be determined.
//
// Example:
//
//	branch, err := GetCurrentBranch("/path/to/repo")
//	if err != nil {
//	    fmt.Println("Error:", err)
//	    return
//	}
//	fmt.Println("Current branch:", branch)
func GetCurrentBranch(path string) (string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", err
	}

	head, err := getHead(repo)
	if err != nil {
		return "", err
	}

	// Check if we're in a detached HEAD state
	if !head.Name().IsBranch() {
		return "", fmt.Errorf(errDetachedHead)
	}

	// Get the branch name
	return head.Name().Short(), nil
}

// GetCommitHash returns the current commit hash (both short and full versions)
// Returns (shortHash, fullHash, error)
func GetCommitHash(path string) (string, string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", "", err
	}

	head, err := getHead(repo)
	if err != nil {
		return "", "", err
	}

	// Get the commit hash
	hash := head.Hash()
	shortHash := hash.String()[:7] // First 7 characters
	fullHash := hash.String()

	return shortHash, fullHash, nil
}

// ============================================================================
// Change Detection
// ============================================================================

// HasUncommittedChanges checks if the repository has uncommitted changes.
// Returns (hasChanges, summary, error) where summary is a human-readable
// description like "2 modified, 1 untracked" or "clean" if no changes.
//
// Example:
//
//	hasChanges, summary, err := HasUncommittedChanges("/path/to/repo")
//	if err != nil {
//	    return err
//	}
//	if hasChanges {
//	    fmt.Printf("Uncommitted changes: %s\n", summary)
//	} else {
//	    fmt.Println("Working tree is clean")
//	}
func HasUncommittedChanges(path string) (bool, string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return false, "", err
	}

	// Get the working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return false, "", fmt.Errorf(errGetWorktree, err)
	}

	// Get the status
	status, err := worktree.Status()
	if err != nil {
		return false, "", fmt.Errorf(errGetStatus, err)
	}

	// Count different types of changes
	var modified, added, deleted, untracked int
	for _, fileStatus := range status {
		switch fileStatus.Staging {
		case git.Added:
			added++
		case git.Modified:
			modified++
		case git.Deleted:
			deleted++
		}

		// Check worktree status for untracked files
		if fileStatus.Worktree == git.Untracked {
			untracked++
		} else if fileStatus.Worktree == git.Modified {
			modified++
		} else if fileStatus.Worktree == git.Deleted {
			deleted++
		}
	}

	// Check if there are any changes
	hasChanges := len(status) > 0

	// Build summary
	summary := buildChangesSummary(modified, added, deleted, untracked)

	return hasChanges, summary, nil
}

// buildChangesSummary creates a human-readable summary of changes
func buildChangesSummary(modified, added, deleted, untracked int) string {
	if modified == 0 && added == 0 && deleted == 0 && untracked == 0 {
		return stateClean
	}

	var parts []string
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleted))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}

	summary := ""
	for i, part := range parts {
		if i > 0 {
			summary += ", "
		}
		summary += part
	}

	return summary
}

// IsBranchDirty checks if the current branch has uncommitted changes
// This is a convenience wrapper around HasUncommittedChanges
func IsBranchDirty(path string) (bool, error) {
	hasChanges, _, err := HasUncommittedChanges(path)
	return hasChanges, err
}

// ============================================================================
// Remote Operations
// ============================================================================

// GetRemoteURL returns the URL of the remote repository.
// Returns the URL of the first remote (typically "origin").
// The URL is normalized to a consistent format (e.g., "github.com/user/repo").
//
// Example:
//
//	url, err := GetRemoteURL("/path/to/repo")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("Remote URL:", url)
func GetRemoteURL(path string) (string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", err
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf(errGetRemotes, err)
	}

	if len(remotes) == 0 {
		return "", fmt.Errorf(errNoRemotes)
	}

	// Get the first remote (usually "origin")
	remote := remotes[0]
	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf(errNoRemoteURLs)
	}

	return normalizeGitURL(remote.Config().URLs[0]), nil
}

// GetLatestCommitMessage returns the message of the latest commit.
//
// Example:
//
//	msg, err := GetLatestCommitMessage("/path/to/repo")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("Latest commit:", msg)
func GetLatestCommitMessage(path string) (string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", err
	}

	head, err := getHead(repo)
	if err != nil {
		return "", err
	}

	// Get the commit object
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf(errGetCommit, err)
	}

	return commit.Message, nil
}

// IsAheadOfRemote checks if the local branch is ahead of the remote branch.
// Returns the number of commits the local branch is ahead.
// Returns 0 if the branches are in sync or if the remote branch doesn't exist.
//
// Note: This is a simplified implementation that returns 1 if there are any differences.
// A full implementation would walk the commit graph to count exact commits ahead.
//
// Example:
//
//	ahead, err := IsAheadOfRemote("/path/to/repo")
//	if err != nil {
//	    return err
//	}
//	if ahead > 0 {
//	    fmt.Printf("Local branch is %d commit(s) ahead of remote\n", ahead)
//	}
func IsAheadOfRemote(path string) (int, error) {
	repo, err := openRepo(path)
	if err != nil {
		return 0, err
	}

	head, err := getHead(repo)
	if err != nil {
		return 0, err
	}

	// Get the current branch name
	branchName := head.Name().Short()

	// Get the remote tracking branch
	remoteBranchName := plumbing.NewRemoteReferenceName("origin", branchName)
	remoteBranch, err := repo.Reference(remoteBranchName, true)
	if err != nil {
		// Remote branch might not exist
		return 0, nil
	}

	// Count commits between local and remote
	localCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return 0, fmt.Errorf(errGetLocalCommit, err)
	}

	remoteCommit, err := repo.CommitObject(remoteBranch.Hash())
	if err != nil {
		return 0, fmt.Errorf(errGetRemoteCommit, err)
	}

	// Simple check: if hashes are different, we might be ahead
	if localCommit.Hash == remoteCommit.Hash {
		return 0, nil
	}

	// For a more accurate count, we'd need to walk the commit graph
	// For now, we'll just return 1 if they're different (simplified)
	return 1, nil
}
