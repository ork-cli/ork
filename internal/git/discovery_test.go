package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SSH URL",
			input:    "git@github.com:org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "Already normalized",
			input:    "github.com/org/repo",
			expected: "github.com/org/repo",
		},
		{
			name:     "SSH URL without .git",
			input:    "git@gitlab.com:company/project",
			expected: "gitlab.com/company/project",
		},
		{
			name:     "HTTPS URL without .git",
			input:    "https://bitbucket.org/team/repo",
			expected: "bitbucket.org/team/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeGitURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create a fake .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Test that it's detected as a git repo
	assert.True(t, isGitRepository(tmpDir))

	// Test that a non-git directory is not detected
	nonGitDir := filepath.Join(tmpDir, "not-a-repo")
	err = os.Mkdir(nonGitDir, 0755)
	require.NoError(t, err)
	assert.False(t, isGitRepository(nonGitDir))

	// Test that a non-existent directory is not detected
	assert.False(t, isGitRepository(filepath.Join(tmpDir, "does-not-exist")))
}

func TestDiscoverRepositories(t *testing.T) {
	// Create temporary workspace
	workspace := t.TempDir()

	// Create a real git repository
	repo1Path := filepath.Join(workspace, "repo1")
	err := os.Mkdir(repo1Path, 0755)
	require.NoError(t, err)

	_, err = git.PlainInit(repo1Path, false)
	require.NoError(t, err)

	// Create a nested git repository (should be found within depth 3)
	nestedPath := filepath.Join(workspace, "projects", "nested", "repo2")
	err = os.MkdirAll(nestedPath, 0755)
	require.NoError(t, err)

	_, err = git.PlainInit(nestedPath, false)
	require.NoError(t, err)

	// Create a directory that's too deep (should not be found with depth 2)
	deepPath := filepath.Join(workspace, "a", "b", "c", "d", "repo3")
	err = os.MkdirAll(deepPath, 0755)
	require.NoError(t, err)

	_, err = git.PlainInit(deepPath, false)
	require.NoError(t, err)

	// Test discovery with depth 3 (should find repo1 and repo2, not repo3)
	repos, err := DiscoverRepositories([]string{workspace}, 3)
	require.NoError(t, err)
	assert.Equal(t, 2, len(repos))

	// Verify repo names
	repoNames := make(map[string]bool)
	for _, repo := range repos {
		repoNames[repo.Name] = true
	}
	assert.True(t, repoNames["repo1"])
	assert.True(t, repoNames["repo2"])
	assert.False(t, repoNames["repo3"])
}

func TestDiscoverRepositories_SkipsNodeModules(t *testing.T) {
	// Create temporary workspace
	workspace := t.TempDir()

	// Create a repo in workspace root
	repo1Path := filepath.Join(workspace, "myproject")
	err := os.Mkdir(repo1Path, 0755)
	require.NoError(t, err)

	_, err = git.PlainInit(repo1Path, false)
	require.NoError(t, err)

	// Create a fake repo inside node_modules (should be skipped)
	nodeModulesPath := filepath.Join(workspace, "myproject", "node_modules", "some-package")
	err = os.MkdirAll(nodeModulesPath, 0755)
	require.NoError(t, err)

	_, err = git.PlainInit(nodeModulesPath, false)
	require.NoError(t, err)

	// Test discovery
	repos, err := DiscoverRepositories([]string{workspace}, 3)
	require.NoError(t, err)

	// Should only find myproject, not the repo in node_modules
	assert.Equal(t, 1, len(repos))
	assert.Equal(t, "myproject", repos[0].Name)
}

func TestDiscoverRepositories_NonExistentWorkspace(t *testing.T) {
	// Test with a non-existent workspace
	repos, err := DiscoverRepositories([]string{"/this/path/does/not/exist"}, 3)
	require.NoError(t, err)
	assert.Equal(t, 0, len(repos), "Should return empty list for non-existent workspace")
}

func TestFindRepository(t *testing.T) {
	repos := []Repository{
		{Name: "frontend", Path: "/home/user/code/frontend", URL: "github.com/org/frontend"},
		{Name: "backend", Path: "/home/user/code/backend", URL: "github.com/org/backend"},
		{Name: "api", Path: "/home/user/code/api", URL: "gitlab.com/team/api"},
	}

	tests := []struct {
		name      string
		searchURL string
		found     bool
		repoName  string
	}{
		{
			name:      "Find exact match",
			searchURL: "github.com/org/frontend",
			found:     true,
			repoName:  "frontend",
		},
		{
			name:      "Find with SSH URL",
			searchURL: "git@github.com:org/backend.git",
			found:     true,
			repoName:  "backend",
		},
		{
			name:      "Find with HTTPS URL",
			searchURL: "https://gitlab.com/team/api.git",
			found:     true,
			repoName:  "api",
		},
		{
			name:      "Not found",
			searchURL: "github.com/org/notfound",
			found:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindRepository(repos, tt.searchURL)
			if tt.found {
				require.NotNil(t, result)
				assert.Equal(t, tt.repoName, result.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}
