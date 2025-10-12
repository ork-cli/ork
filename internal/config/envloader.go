package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ============================================================================
// Type Definitions
// ============================================================================

// EnvVars represents a collection of environment variables
type EnvVars map[string]string

// ============================================================================
// Public API
// ============================================================================

// LoadEnvFile loads environment variables from a .env file
// Returns an empty map if the file doesn't exist (not an error)
func LoadEnvFile(filePath string) (EnvVars, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist - return an empty map (not an error)
		return make(EnvVars), nil
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("❌ failed to close .env file: %v\n", err)
		}
	}()

	// Parse the file
	envVars := make(EnvVars)
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Parse the line
		key, value, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("error on line %d: %w", lineNumber, err)
		}

		// Skip empty lines and comments
		if key == "" {
			continue
		}

		// Add to env vars
		envVars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return envVars, nil
}

// LoadProjectEnv loads the project-level .env file from the current directory
// Looks for .env in the directory where ork.yml is located
func LoadProjectEnv() (EnvVars, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load .env from current directory
	envPath := filepath.Join(cwd, ".env")
	return LoadEnvFile(envPath)
}

// LoadServiceEnv loads service-specific .env file
// Looks for .env.<service-name> in the current directory
func LoadServiceEnv(serviceName string) (EnvVars, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load .env.<service-name>
	envPath := filepath.Join(cwd, fmt.Sprintf(".env.%s", serviceName))
	return LoadEnvFile(envPath)
}

// MergeEnvVars merges multiple EnvVars maps with priority
// Later maps override earlier ones
// Example: MergeEnvVars(projectEnv, serviceEnv, configEnv)
// configEnv has highest priority, projectEnv has lowest
func MergeEnvVars(envMaps ...EnvVars) EnvVars {
	result := make(EnvVars)

	// Merge in order - later ones override earlier ones
	for _, envMap := range envMaps {
		for key, value := range envMap {
			result[key] = value
		}
	}

	return result
}

// LoadAllEnvForService loads and merges all environment variables for a service
// Priority (lowest to highest):
//  1. Project .env file
//  2. Service-specific .env.<service> file
//  3. Environment variables from the york.yml config
//
// After merging, all variable references (${VAR} or $VAR) are interpolated
func LoadAllEnvForService(serviceName string, configEnv map[string]string) (EnvVars, error) {
	// Load project-level .env
	projectEnv, err := LoadProjectEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load project .env: %w", err)
	}

	// Load service-specific .env
	serviceEnv, err := LoadServiceEnv(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load service .env: %w", err)
	}

	// Convert config env to EnvVars
	cfgEnv := make(EnvVars)
	for k, v := range configEnv {
		cfgEnv[k] = v
	}

	// Merge with priority: project < service < config
	merged := MergeEnvVars(projectEnv, serviceEnv, cfgEnv)

	// Interpolate variable references
	interpolated, err := InterpolateEnvVars(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to interpolate variables for service %s: %w", serviceName, err)
	}

	return interpolated, nil
}

// InterpolateEnvVars interpolates variable references in environment values
// Supports:
//   - ${VAR_NAME} - standard form
//   - $VAR_NAME - short form (word characters only)
//   - ${VAR_NAME:-default} - with default value
//
// Variables are resolved from:
//  1. The provided EnvVars map (for self-referencing)
//  2. System environment variables (os.Getenv)
//
// Returns an error if circular references are detected
func InterpolateEnvVars(envVars EnvVars) (EnvVars, error) {
	result := make(EnvVars)
	resolving := make(map[string]bool) // Track variables being resolved to detect circular refs

	// Interpolate each value
	for key, value := range envVars {
		interpolated, err := interpolateValue(value, envVars, resolving)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate variable %s: %w", key, err)
		}
		result[key] = interpolated
	}

	return result, nil
}

// ============================================================================
// Private Helpers - Variable Interpolation
// ============================================================================

// Regular expressions for variable references
var (
	// Matches ${VAR_NAME} or ${VAR_NAME:-default}
	varRefWithBraces = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?}`)
	// Matches $VAR_NAME (word characters only, no braces)
	varRefShort = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
)

// interpolateValue interpolates all variable references in a single value
func interpolateValue(value string, envVars EnvVars, resolving map[string]bool) (string, error) {
	var interpolationError error

	// First, handle ${VAR} and ${VAR:-default} (with braces)
	result := varRefWithBraces.ReplaceAllStringFunc(value, func(match string) string {
		// If we already have an error, don't process more replacements
		if interpolationError != nil {
			return match
		}

		submatches := varRefWithBraces.FindStringSubmatch(match)
		varName := submatches[1]
		defaultValue := ""
		if len(submatches) > 3 {
			defaultValue = submatches[3]
		}

		// Resolve the variable
		resolved, err := resolveVariable(varName, envVars, resolving, defaultValue)
		if err != nil {
			interpolationError = err
			return match
		}
		return resolved
	})

	// Check for errors from braces replacement
	if interpolationError != nil {
		return "", interpolationError
	}

	// Then, handle $VAR (short form, no braces)
	result = varRefShort.ReplaceAllStringFunc(result, func(match string) string {
		// If we already have an error, don't process more replacements
		if interpolationError != nil {
			return match
		}

		submatches := varRefShort.FindStringSubmatch(match)
		varName := submatches[1]

		// Resolve the variable
		resolved, err := resolveVariable(varName, envVars, resolving, "")
		if err != nil {
			interpolationError = err
			return match
		}
		return resolved
	})

	// Check for errors from short form replacement
	if interpolationError != nil {
		return "", interpolationError
	}

	return result, nil
}

// resolveVariable resolves a single variable reference
// Looks up in envVars first, then os.Getenv, then uses defaultValue
func resolveVariable(varName string, envVars EnvVars, resolving map[string]bool, defaultValue string) (string, error) {
	// Check for circular reference
	if resolving[varName] {
		return "", fmt.Errorf("circular reference detected: %s", varName)
	}

	// Try to get from envVars first
	if val, exists := envVars[varName]; exists {
		// Mark as resolving to detect circular references
		resolving[varName] = true
		defer delete(resolving, varName)

		// Recursively interpolate the value (in case it also contains variables)
		interpolated, err := interpolateValue(val, envVars, resolving)
		if err != nil {
			return "", err
		}
		return interpolated, nil
	}

	// Try system environment variable
	if val := os.Getenv(varName); val != "" {
		return val, nil
	}

	// Use default value if provided
	if defaultValue != "" {
		return defaultValue, nil
	}

	// Return an empty string if not found and no default
	return "", nil
}

// ============================================================================
// Private Helpers - Line Parsing
// ============================================================================

// parseLine parses a single line from a .env file
// Returns (key, value, error)
// Returns ("", "", nil) for blank lines and comments
func parseLine(line string) (string, string, error) {
	// Trim whitespace
	line = strings.TrimSpace(line)

	// Skip empty lines
	if line == "" {
		return "", "", nil
	}

	// Skip comments
	if strings.HasPrefix(line, "#") {
		return "", "", nil
	}

	// Find the = separator
	equalIndex := strings.Index(line, "=")
	if equalIndex == -1 {
		// No = sign - skip this line (could warn, but we'll be lenient)
		return "", "", nil
	}

	// Extract key and value
	key := strings.TrimSpace(line[:equalIndex])
	value := strings.TrimSpace(line[equalIndex+1:])

	// Validate key
	if key == "" {
		return "", "", fmt.Errorf("empty key in .env file")
	}

	// Remove quotes from value if present
	value = unquoteValue(value)

	return key, value, nil
}

// unquoteValue removes surrounding quotes from a value
// Supports both single and double quotes
func unquoteValue(value string) string {
	// Check for double quotes
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}

	// Check for single quotes
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}
