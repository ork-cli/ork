package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_ValidConfig tests loading a valid ork.yml file
func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a valid ork.yml file
	configContent := `
version: "1.0"
project: test-project
services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
`
	configPath := filepath.Join(tempDir, "ork.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) // Restore original directory after test

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Load the config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading valid config, got: %v", err)
	}

	// Verify the loaded config
	if cfg.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", cfg.Version)
	}
	if cfg.Project != "test-project" {
		t.Errorf("expected project 'test-project', got '%s'", cfg.Project)
	}
	if len(cfg.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(cfg.Services))
	}
}

// TestLoad_DotOrkYml tests loading .ork.yml (hidden file)
func TestLoad_DotOrkYml(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `
version: "1.0"
project: hidden-config
services:
  api:
    image: node:18
`
	configPath := filepath.Join(tempDir, ".ork.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create .ork.yml file: %v", err)
	}

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading .ork.yml, got: %v", err)
	}

	if cfg.Project != "hidden-config" {
		t.Errorf("expected project 'hidden-config', got '%s'", cfg.Project)
	}
}

// TestLoad_PreferOrkYmlOverDotOrk tests that ork.yml takes precedence over .ork.yml
func TestLoad_PreferOrkYmlOverDotOrk(t *testing.T) {
	tempDir := t.TempDir()

	// Create both files
	orkContent := `
version: "1.0"
project: from-ork-yml
services:
  web:
    image: nginx:alpine
`
	dotOrkContent := `
version: "1.0"
project: from-dot-ork-yml
services:
  api:
    image: node:18
`

	os.WriteFile(filepath.Join(tempDir, "ork.yml"), []byte(orkContent), 0644)
	os.WriteFile(filepath.Join(tempDir, ".ork.yml"), []byte(dotOrkContent), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should load ork.yml, not .ork.yml
	if cfg.Project != "from-ork-yml" {
		t.Errorf("expected ork.yml to be preferred, got project '%s'", cfg.Project)
	}
}

// TestLoad_MissingConfigFile tests error when no config file exists
func TestLoad_MissingConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}

	if !strings.Contains(err.Error(), "no ork.yml or .ork.yml found") {
		t.Errorf("expected 'no ork.yml or .ork.yml found' error, got: %v", err)
	}
}

// TestLoad_InvalidYAML tests error handling for malformed YAML
func TestLoad_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid YAML (tabs and spaces mixed, missing quotes, etc.)
	invalidYAML := `
version: "1.0"
project: test
services:
  web:
    image: nginx:alpine
	  ports:   # This line uses tabs instead of spaces
      - "80:80
`
	configPath := filepath.Join(tempDir, "ork.yml")
	os.WriteFile(configPath, []byte(invalidYAML), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("expected 'failed to parse YAML' error, got: %v", err)
	}
}

// TestLoad_EmptyFile tests error handling for empty config file
func TestLoad_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "ork.yml")
	os.WriteFile(configPath, []byte(""), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	cfg, err := Load()
	// Empty YAML is valid, but should have empty fields
	if err != nil {
		t.Fatalf("expected no error for empty file, got: %v", err)
	}

	if cfg.Version != "" {
		t.Errorf("expected empty version, got '%s'", cfg.Version)
	}
}

// TestLoad_ComplexConfig tests loading a config with all features
func TestLoad_ComplexConfig(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `
version: "1.0"
project: complex-project
services:
  frontend:
    image: nginx:alpine
    ports:
      - "3000:80"
    env:
      API_URL: http://localhost:8080
    depends_on:
      - api
  api:
    git: github.com/org/api
    ports:
      - "8080:8080"
    depends_on:
      - postgres
  postgres:
    build:
      context: ./database
      dockerfile: Dockerfile.postgres
      args:
        PG_VERSION: "15"
    env:
      POSTGRES_PASSWORD: secret
    health:
      endpoint: /health
      interval: 5s
      timeout: 3s
      retries: 3
`
	configPath := filepath.Join(tempDir, "ork.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading complex config, got: %v", err)
	}

	// Verify structure
	if len(cfg.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(cfg.Services))
	}

	// Check frontend
	frontend, exists := cfg.Services["frontend"]
	if !exists {
		t.Fatal("expected 'frontend' service to exist")
	}
	if frontend.Image != "nginx:alpine" {
		t.Errorf("expected frontend image 'nginx:alpine', got '%s'", frontend.Image)
	}
	if len(frontend.DependsOn) != 1 || frontend.DependsOn[0] != "api" {
		t.Errorf("expected frontend to depend on 'api', got %v", frontend.DependsOn)
	}

	// Check api
	api, exists := cfg.Services["api"]
	if !exists {
		t.Fatal("expected 'api' service to exist")
	}
	if api.Git != "github.com/org/api" {
		t.Errorf("expected api git 'github.com/org/api', got '%s'", api.Git)
	}

	// Check postgres build config
	postgres, exists := cfg.Services["postgres"]
	if !exists {
		t.Fatal("expected 'postgres' service to exist")
	}
	if postgres.Build == nil {
		t.Fatal("expected postgres to have build config")
	}
	if postgres.Build.Context != "./database" {
		t.Errorf("expected build context './database', got '%s'", postgres.Build.Context)
	}
	if postgres.Build.Dockerfile != "Dockerfile.postgres" {
		t.Errorf("expected dockerfile 'Dockerfile.postgres', got '%s'", postgres.Build.Dockerfile)
	}

	// Check health config
	if postgres.Health == nil {
		t.Fatal("expected postgres to have health config")
	}
	if postgres.Health.Endpoint != "/health" {
		t.Errorf("expected health endpoint '/health', got '%s'", postgres.Health.Endpoint)
	}
	if postgres.Health.Retries != 3 {
		t.Errorf("expected 3 retries, got %d", postgres.Health.Retries)
	}
}

// TestFindConfigFile_Success tests finding ork.yml
func TestFindConfigFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "ork.yml")
	os.WriteFile(configPath, []byte("version: 1.0"), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	foundPath, err := findConfigFile()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.HasSuffix(foundPath, "ork.yml") {
		t.Errorf("expected path to end with 'ork.yml', got '%s'", foundPath)
	}
}

// TestFindConfigFile_NotFound tests error when no config file exists
func TestFindConfigFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	_, err := findConfigFile()
	if err == nil {
		t.Fatal("expected error when config file not found, got nil")
	}

	if !strings.Contains(err.Error(), "no ork.yml or .ork.yml found") {
		t.Errorf("expected 'no ork.yml or .ork.yml found' error, got: %v", err)
	}
}
