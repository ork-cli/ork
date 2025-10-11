package config

import (
	"strings"
	"testing"
)

// TestValidate_Success tests that a valid config passes validation
func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		Project: "test-project",
		Services: map[string]Service{
			"web": {
				Image: "nginx:alpine",
				Ports: []string{"80:80"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestValidate_MissingVersion tests that missing version fails validation
func TestValidate_MissingVersion(t *testing.T) {
	cfg := &Config{
		Project: "test-project",
		Services: map[string]Service{
			"web": {Image: "nginx:alpine"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}

	if !strings.Contains(err.Error(), "version is required") {
		t.Errorf("expected 'version is required' error, got: %v", err)
	}
}

// TestValidate_MissingProject tests that missing project name fails validation
func TestValidate_MissingProject(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		Services: map[string]Service{
			"web": {Image: "nginx:alpine"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing project, got nil")
	}

	if !strings.Contains(err.Error(), "project name is required") {
		t.Errorf("expected 'project name is required' error, got: %v", err)
	}
}

// TestValidate_NoServices tests that config with no services fails validation
func TestValidate_NoServices(t *testing.T) {
	cfg := &Config{
		Version:  "1.0",
		Project:  "test-project",
		Services: map[string]Service{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no services, got nil")
	}

	if !strings.Contains(err.Error(), "at least one service must be defined") {
		t.Errorf("expected 'at least one service' error, got: %v", err)
	}
}

// TestValidateServiceSource_NoSource tests that service with no source fails
func TestValidateServiceSource_NoSource(t *testing.T) {
	service := Service{
		Ports: []string{"80:80"},
	}

	err := validateServiceSource(service)
	if err == nil {
		t.Fatal("expected error for no source, got nil")
	}

	if !strings.Contains(err.Error(), "must specify one of: git, image, or build") {
		t.Errorf("expected 'must specify one of' error, got: %v", err)
	}
}

// TestValidateServiceSource_MultipleSourcesGitAndImage tests git + image fails
func TestValidateServiceSource_MultipleSourcesGitAndImage(t *testing.T) {
	service := Service{
		Git:   "github.com/org/repo",
		Image: "nginx:alpine",
	}

	err := validateServiceSource(service)
	if err == nil {
		t.Fatal("expected error for multiple sources, got nil")
	}

	if !strings.Contains(err.Error(), "can only specify one of") {
		t.Errorf("expected 'can only specify one of' error, got: %v", err)
	}
}

// TestValidateServiceSource_MultipleSourcesGitAndBuild tests git + build fails
func TestValidateServiceSource_MultipleSourcesGitAndBuild(t *testing.T) {
	service := Service{
		Git: "github.com/org/repo",
		Build: &Build{
			Context: "./app",
		},
	}

	err := validateServiceSource(service)
	if err == nil {
		t.Fatal("expected error for multiple sources, got nil")
	}

	if !strings.Contains(err.Error(), "can only specify one of") {
		t.Errorf("expected 'can only specify one of' error, got: %v", err)
	}
}

// TestValidateServiceSource_AllThreeSources tests git + image + build fails
func TestValidateServiceSource_AllThreeSources(t *testing.T) {
	service := Service{
		Git:   "github.com/org/repo",
		Image: "nginx:alpine",
		Build: &Build{
			Context: "./app",
		},
	}

	err := validateServiceSource(service)
	if err == nil {
		t.Fatal("expected error for three sources, got nil")
	}

	if !strings.Contains(err.Error(), "found 3") {
		t.Errorf("expected error mentioning 3 sources, got: %v", err)
	}
}

// TestValidateServiceSource_ValidGit tests that git-only source is valid
func TestValidateServiceSource_ValidGit(t *testing.T) {
	service := Service{
		Git: "github.com/org/repo",
	}

	err := validateServiceSource(service)
	if err != nil {
		t.Errorf("expected no error for valid git source, got: %v", err)
	}
}

// TestValidateServiceSource_ValidImage tests that image-only source is valid
func TestValidateServiceSource_ValidImage(t *testing.T) {
	service := Service{
		Image: "nginx:alpine",
	}

	err := validateServiceSource(service)
	if err != nil {
		t.Errorf("expected no error for valid image source, got: %v", err)
	}
}

// TestValidateServiceSource_ValidBuild tests that build-only source is valid
func TestValidateServiceSource_ValidBuild(t *testing.T) {
	service := Service{
		Build: &Build{
			Context: "./app",
		},
	}

	err := validateServiceSource(service)
	if err != nil {
		t.Errorf("expected no error for valid build source, got: %v", err)
	}
}

// TestCountSources tests the countSources helper function
func TestCountSources(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected int
	}{
		{
			name:     "no sources",
			service:  Service{},
			expected: 0,
		},
		{
			name:     "git only",
			service:  Service{Git: "github.com/org/repo"},
			expected: 1,
		},
		{
			name:     "image only",
			service:  Service{Image: "nginx:alpine"},
			expected: 1,
		},
		{
			name:     "build only",
			service:  Service{Build: &Build{Context: "./app"}},
			expected: 1,
		},
		{
			name: "git and image",
			service: Service{
				Git:   "github.com/org/repo",
				Image: "nginx:alpine",
			},
			expected: 2,
		},
		{
			name: "all three",
			service: Service{
				Git:   "github.com/org/repo",
				Image: "nginx:alpine",
				Build: &Build{Context: "./app"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countSources(tt.service)
			if count != tt.expected {
				t.Errorf("expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

// TestValidateBuildConfig_MissingContext tests build without context fails
func TestValidateBuildConfig_MissingContext(t *testing.T) {
	service := Service{
		Build: &Build{
			Dockerfile: "Dockerfile",
		},
	}

	err := validateBuildConfig(service)
	if err == nil {
		t.Fatal("expected error for missing build context, got nil")
	}

	if !strings.Contains(err.Error(), "build.context is required") {
		t.Errorf("expected 'build.context is required' error, got: %v", err)
	}
}

// TestValidateBuildConfig_ValidBuild tests valid build config passes
func TestValidateBuildConfig_ValidBuild(t *testing.T) {
	service := Service{
		Build: &Build{
			Context:    "./app",
			Dockerfile: "Dockerfile",
		},
	}

	err := validateBuildConfig(service)
	if err != nil {
		t.Errorf("expected no error for valid build config, got: %v", err)
	}
}

// TestValidateBuildConfig_NoBuildSection tests service without build passes
func TestValidateBuildConfig_NoBuildSection(t *testing.T) {
	service := Service{
		Image: "nginx:alpine",
	}

	err := validateBuildConfig(service)
	if err != nil {
		t.Errorf("expected no error when build is nil, got: %v", err)
	}
}

// TestValidateDependencies_UnknownService tests reference to unknown service fails
func TestValidateDependencies_UnknownService(t *testing.T) {
	allServices := map[string]Service{
		"api": {Image: "node:18"},
	}

	err := validateDependencies("frontend", []string{"api", "postgres"}, allServices)
	if err == nil {
		t.Fatal("expected error for unknown dependency, got nil")
	}

	if !strings.Contains(err.Error(), "unknown service 'postgres'") {
		t.Errorf("expected 'unknown service' error, got: %v", err)
	}
}

// TestValidateDependencies_SelfDependency tests self-dependency fails
func TestValidateDependencies_SelfDependency(t *testing.T) {
	allServices := map[string]Service{
		"api": {Image: "node:18"},
	}

	err := validateDependencies("api", []string{"api"}, allServices)
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}

	if !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Errorf("expected 'cannot depend on itself' error, got: %v", err)
	}
}

// TestValidateDependencies_ValidDependencies tests valid dependencies pass
func TestValidateDependencies_ValidDependencies(t *testing.T) {
	allServices := map[string]Service{
		"frontend": {Image: "nginx:alpine"},
		"api":      {Image: "node:18"},
		"postgres": {Image: "postgres:15"},
	}

	err := validateDependencies("frontend", []string{"api", "postgres"}, allServices)
	if err != nil {
		t.Errorf("expected no error for valid dependencies, got: %v", err)
	}
}

// TestValidateDependencies_NoDependencies tests empty dependencies pass
func TestValidateDependencies_NoDependencies(t *testing.T) {
	allServices := map[string]Service{
		"api": {Image: "node:18"},
	}

	err := validateDependencies("api", []string{}, allServices)
	if err != nil {
		t.Errorf("expected no error for no dependencies, got: %v", err)
	}
}

// TestValidatePorts_InvalidFormat tests port without colon fails
func TestValidatePorts_InvalidFormat(t *testing.T) {
	ports := []string{"8080"}

	err := validatePorts(ports)
	if err == nil {
		t.Fatal("expected error for invalid port format, got nil")
	}

	if !strings.Contains(err.Error(), "invalid port format '8080'") {
		t.Errorf("expected 'invalid port format' error, got: %v", err)
	}
}

// TestValidatePorts_ValidFormats tests various valid port formats
func TestValidatePorts_ValidFormats(t *testing.T) {
	tests := []struct {
		name  string
		ports []string
	}{
		{
			name:  "simple mapping",
			ports: []string{"8080:8080"},
		},
		{
			name:  "different ports",
			ports: []string{"3000:80"},
		},
		{
			name:  "multiple ports",
			ports: []string{"8080:8080", "3000:3000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePorts(tt.ports)
			if err != nil {
				t.Errorf("expected no error for valid ports %v, got: %v", tt.ports, err)
			}
		})
	}
}

// TestValidatePorts_EmptyList tests empty port list passes
func TestValidatePorts_EmptyList(t *testing.T) {
	err := validatePorts([]string{})
	if err != nil {
		t.Errorf("expected no error for empty ports, got: %v", err)
	}
}
