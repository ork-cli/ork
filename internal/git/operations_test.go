package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRepo creates a temporary git repository for testing
func createTestRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Initialize git repository
	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	return tmpDir, repo
}

// createTestCommit creates a test commit in the repository
func createTestCommit(t *testing.T, repo *git.Repository, repoPath, filename, content string) {
	t.Helper()

	// Create a test file
	filePath := filepath.Join(repoPath, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	// Get the worktree
	w, err := repo.Worktree()
	require.NoError(t, err)

	// Add the file
	_, err = w.Add(filename)
	require.NoError(t, err)

	// Create commit
	_, err = w.Commit("Test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	require.NoError(t, err)
}

func TestRepoExistsAt(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "existing git repository",
			setup: func(t *testing.T) string {
				repoPath, _ := createTestRepo(t)
				return repoPath
			},
			expected: true,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expected: false,
		},
		{
			name: "directory without .git",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			result := RepoExistsAt(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		expectedBranch string
		expectError    bool
	}{
		{
			name: "repository with commits on main",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectedBranch: "master", // go-git creates "master" by default
			expectError:    false,
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			branch, err := GetCurrentBranch(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBranch, branch)
			}
		})
	}
}

func TestGetCommitHash(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		expectError bool
	}{
		{
			name: "repository with commits",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectError: false,
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			shortHash, fullHash, err := GetCommitHash(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, shortHash, 7, "short hash should be 7 characters")
				assert.Len(t, fullHash, 40, "full hash should be 40 characters")
				assert.Equal(t, fullHash[:7], shortHash, "short hash should match first 7 chars of full hash")
			}
		})
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T) string
		expectChanges   bool
		expectError     bool
		expectedSummary string
	}{
		{
			name: "clean repository",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectChanges:   false,
			expectError:     false,
			expectedSummary: "clean",
		},
		{
			name: "repository with untracked file",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				// Add an untracked file
				untrackedFile := filepath.Join(repoPath, "untracked.txt")
				err := os.WriteFile(untrackedFile, []byte("untracked content"), 0644)
				require.NoError(t, err)
				return repoPath
			},
			expectChanges:   true,
			expectError:     false,
			expectedSummary: "1 untracked",
		},
		{
			name: "repository with modified file",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				// Modify the tracked file
				modifiedFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(modifiedFile, []byte("modified content"), 0644)
				require.NoError(t, err)
				return repoPath
			},
			expectChanges:   true,
			expectError:     false,
			expectedSummary: "1 modified",
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			hasChanges, summary, err := HasUncommittedChanges(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectChanges, hasChanges)
				assert.Equal(t, tt.expectedSummary, summary)
			}
		})
	}
}

func TestGetRepoState(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		expectExists   bool
		expectError    bool
		checkBranch    bool
		expectedBranch string
	}{
		{
			name: "clean repository with commits",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectExists:   true,
			expectError:    false,
			checkBranch:    true,
			expectedBranch: "master",
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectExists: false,
			expectError:  false,
		},
		{
			name: "repository with uncommitted changes",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				// Add an untracked file
				untrackedFile := filepath.Join(repoPath, "untracked.txt")
				err := os.WriteFile(untrackedFile, []byte("untracked content"), 0644)
				require.NoError(t, err)
				return repoPath
			},
			expectExists:   true,
			expectError:    false,
			checkBranch:    true,
			expectedBranch: "master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			state, err := GetRepoState(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state)
				assert.Equal(t, tt.expectExists, state.Exists)

				if tt.checkBranch && tt.expectExists {
					assert.Equal(t, tt.expectedBranch, state.Branch)
					assert.NotEmpty(t, state.CommitHash)
					assert.NotEmpty(t, state.CommitHashFull)
				}
			}
		})
	}
}

func TestIsBranchDirty(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		expectDirty bool
		expectError bool
	}{
		{
			name: "clean repository",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectDirty: false,
			expectError: false,
		},
		{
			name: "dirty repository",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				// Add an untracked file
				untrackedFile := filepath.Join(repoPath, "untracked.txt")
				err := os.WriteFile(untrackedFile, []byte("untracked content"), 0644)
				require.NoError(t, err)
				return repoPath
			},
			expectDirty: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			isDirty, err := IsBranchDirty(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectDirty, isDirty)
			}
		})
	}
}

func TestGetLatestCommitMessage(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T) string
		expectedMessage string
		expectError     bool
	}{
		{
			name: "repository with commit",
			setup: func(t *testing.T) string {
				repoPath, repo := createTestRepo(t)
				createTestCommit(t, repo, repoPath, "test.txt", "content")
				return repoPath
			},
			expectedMessage: "Test commit",
			expectError:     false,
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			message, err := GetLatestCommitMessage(path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessage, message)
			}
		})
	}
}

func TestBuildChangesSummary(t *testing.T) {
	tests := []struct {
		name      string
		modified  int
		added     int
		deleted   int
		untracked int
		expected  string
	}{
		{
			name:      "no changes",
			modified:  0,
			added:     0,
			deleted:   0,
			untracked: 0,
			expected:  "clean",
		},
		{
			name:      "only modified",
			modified:  2,
			added:     0,
			deleted:   0,
			untracked: 0,
			expected:  "2 modified",
		},
		{
			name:      "only added",
			modified:  0,
			added:     1,
			deleted:   0,
			untracked: 0,
			expected:  "1 added",
		},
		{
			name:      "only deleted",
			modified:  0,
			added:     0,
			deleted:   3,
			untracked: 0,
			expected:  "3 deleted",
		},
		{
			name:      "only untracked",
			modified:  0,
			added:     0,
			deleted:   0,
			untracked: 5,
			expected:  "5 untracked",
		},
		{
			name:      "mixed changes",
			modified:  2,
			added:     1,
			deleted:   1,
			untracked: 3,
			expected:  "2 modified, 1 added, 1 deleted, 3 untracked",
		},
		{
			name:      "modified and untracked",
			modified:  1,
			added:     0,
			deleted:   0,
			untracked: 2,
			expected:  "1 modified, 2 untracked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildChangesSummary(tt.modified, tt.added, tt.deleted, tt.untracked)
			assert.Equal(t, tt.expected, result)
		})
	}
}
