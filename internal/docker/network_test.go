package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Helper Function Tests - Naming
// ============================================================================

func TestBuildNetworkName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		want        string
	}{
		{
			name:        "simple project name",
			projectName: "webapp",
			want:        "ork-webapp-network",
		},
		{
			name:        "project with hyphens",
			projectName: "my-app",
			want:        "ork-my-app-network",
		},
		{
			name:        "project with underscores",
			projectName: "my_app",
			want:        "ork-my_app-network",
		},
		{
			name:        "single character project",
			projectName: "x",
			want:        "ork-x-network",
		},
		{
			name:        "long project name",
			projectName: "very-long-project-name-for-testing",
			want:        "ork-very-long-project-name-for-testing-network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNetworkName(tt.projectName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// Helper Function Tests - Labels
// ============================================================================

func TestBuildNetworkLabels(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		wantLabels  map[string]string
	}{
		{
			name:        "creates standard ork labels",
			projectName: "webapp",
			wantLabels: map[string]string{
				"ork.managed": "true",
				"ork.project": "webapp",
			},
		},
		{
			name:        "handles project with special chars",
			projectName: "my-app_v2",
			wantLabels: map[string]string{
				"ork.managed": "true",
				"ork.project": "my-app_v2",
			},
		},
		{
			name:        "empty project name",
			projectName: "",
			wantLabels: map[string]string{
				"ork.managed": "true",
				"ork.project": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNetworkLabels(tt.projectName)
			assert.Equal(t, tt.wantLabels, got)
		})
	}
}

func TestBuildNetworkLabels_HasRequiredKeys(t *testing.T) {
	// Test that all required keys are present
	labels := buildNetworkLabels("testproject")

	assert.Contains(t, labels, "ork.managed", "should have ork.managed label")
	assert.Contains(t, labels, "ork.project", "should have ork.project label")
	assert.Equal(t, "true", labels["ork.managed"], "ork.managed should be 'true'")
	assert.Equal(t, "testproject", labels["ork.project"], "ork.project should match project name")
}

func TestBuildNetworkLabels_ConsistentOutput(t *testing.T) {
	// Test that calling the function multiple times with same input gives same output
	projectName := "webapp"

	labels1 := buildNetworkLabels(projectName)
	labels2 := buildNetworkLabels(projectName)

	assert.Equal(t, labels1, labels2, "should produce consistent output")
}
