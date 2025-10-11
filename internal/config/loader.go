package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ============================================================================
// Public API
// ============================================================================

// Load reads and parses the ork.yml configuration file
// It looks for ork.yml in the current directory, falling back to .ork.yml
func Load() (*Config, error) {
	// Find the config file
	configPath, err := findConfigFile()
	if err != nil {
		return nil, err
	}

	// Read the file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML into our Config struct
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	return &config, nil
}

// ============================================================================
// Private Helpers
// ============================================================================

// findConfigFile searches for ork.yml or .ork.yml in the current directory
func findConfigFile() (string, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try ork.yml first
	configPath := filepath.Join(cwd, "ork.yml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// Fall back to .ork.yml
	configPath = filepath.Join(cwd, ".ork.yml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// No config file found
	return "", fmt.Errorf("no ork.yml or .ork.yml found in %s", cwd)
}
