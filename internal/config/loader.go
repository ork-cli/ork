package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// defaultWorkspaces returns the default workspace directories if none are configured
func defaultWorkspaces() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	return []string{
		filepath.Join(home, "code"),
		filepath.Join(home, "projects"),
		filepath.Join(home, "workspace"),
	}
}

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

// LoadGlobal reads and parses the global ~/.ork/config.yml file
// Returns default configuration if the file doesn't exist
func LoadGlobal() (*GlobalConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(home, ".ork", "config.yml")

	// If file doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &GlobalConfig{
			Workspaces: defaultWorkspaces(),
		}, nil
	}

	// Read the file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	// If no workspaces configured, use defaults
	if len(config.Workspaces) == 0 {
		config.Workspaces = defaultWorkspaces()
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
