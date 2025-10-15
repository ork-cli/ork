package utils

import (
	"fmt"
	"strings"
)

// ============================================================================
// Error Types - Categorized for better handling
// ============================================================================

// ErrorKind represents the category of error
type ErrorKind string

const (
	ErrorConfig     ErrorKind = "config"     // Configuration errors
	ErrorDocker     ErrorKind = "docker"     // Docker-related errors
	ErrorNetwork    ErrorKind = "network"    // Network/port errors
	ErrorGit        ErrorKind = "git"        // Git operations
	ErrorService    ErrorKind = "service"    // Service management
	ErrorFile       ErrorKind = "file"       // File system errors
	ErrorValidation ErrorKind = "validation" // Validation failures
	ErrorInternal   ErrorKind = "internal"   // Unexpected internal errors
)

// ============================================================================
// OrkError - Structured error with context and hints
// ============================================================================

// OrkError is a rich error type that provides context, hints, and suggestions
type OrkError struct {
	// Op is the operation that failed (e.g., "docker.start", "config.load")
	Op string

	// Kind categorizes the error
	Kind ErrorKind

	// Err is the underlying error
	Err error

	// Message is a user-friendly description of what went wrong
	Message string

	// Hint provides a suggestion for how to fix the error
	Hint string

	// Details provides additional context (optional)
	Details []string

	// Suggestions provide "did you mean?" style suggestions (optional)
	Suggestions []string
}

// Error implements the error interface
func (e *OrkError) Error() string {
	var parts []string

	// Add operation context if available
	if e.Op != "" {
		parts = append(parts, fmt.Sprintf("operation: %s", e.Op))
	}

	// Add the main message
	if e.Message != "" {
		parts = append(parts, e.Message)
	} else if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying error (for errors.Is and errors.As)
func (e *OrkError) Unwrap() error {
	return e.Err
}

// ============================================================================
// Error Constructors - Convenience functions for common error types
// ============================================================================

// ConfigError creates a configuration-related error
func ConfigError(op, message, hint string, err error) *OrkError {
	return &OrkError{
		Op:      op,
		Kind:    ErrorConfig,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// DockerError creates a Docker-related error
func DockerError(op, message, hint string, err error) *OrkError {
	return &OrkError{
		Op:      op,
		Kind:    ErrorDocker,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// NetworkError creates a network/port-related error
func NetworkError(op, message, hint string, err error) *OrkError {
	return &OrkError{
		Op:      op,
		Kind:    ErrorNetwork,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// ServiceError creates a service management error
func ServiceError(op, message, hint string, err error) *OrkError {
	return &OrkError{
		Op:      op,
		Kind:    ErrorService,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// ValidationError creates a validation error with suggestions
func ValidationError(op, message string, suggestions []string) *OrkError {
	return &OrkError{
		Op:          op,
		Kind:        ErrorValidation,
		Message:     message,
		Suggestions: suggestions,
		Hint:        "Check your configuration for errors",
	}
}

// FileError creates a file system error
func FileError(op, message, hint string, err error) *OrkError {
	return &OrkError{
		Op:      op,
		Kind:    ErrorFile,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// ============================================================================
// Common Error Scenarios - Pre-defined errors for frequent cases
// ============================================================================

// ErrDockerNotRunning creates an error for when the Docker daemon is not running
func ErrDockerNotRunning(err error) *OrkError {
	return &OrkError{
		Op:      "docker.connect",
		Kind:    ErrorDocker,
		Err:     err,
		Message: "Docker daemon is not running",
		Hint:    "Start Docker Desktop or run 'sudo systemctl start docker'",
		Details: []string{
			"Check if Docker is installed: docker --version",
			"Verify Docker daemon status: docker ps",
			"Run diagnostics: ork doctor",
		},
	}
}

// ErrConfigNotFound creates an error for when ork.yml is missing
func ErrConfigNotFound(path string) *OrkError {
	return &OrkError{
		Op:      "config.load",
		Kind:    ErrorConfig,
		Message: fmt.Sprintf("Configuration file not found: %s", path),
		Hint:    "Run 'ork init' to create a new ork.yml",
		Details: []string{
			"Make sure you're in the right directory",
			"Check if the file is named ork.yml or .ork.yml",
		},
	}
}

// ErrServiceNotFound creates an error for unknown service names
func ErrServiceNotFound(serviceName string, availableServices []string) *OrkError {
	return &OrkError{
		Op:          "service.resolve",
		Kind:        ErrorService,
		Message:     fmt.Sprintf("Service '%s' not found in configuration", serviceName),
		Hint:        "Check service names in your ork.yml",
		Suggestions: availableServices,
	}
}

// ErrPortInUse creates an error for port conflicts
func ErrPortInUse(port, service, conflictingProcess string) *OrkError {
	details := []string{
		fmt.Sprintf("Port %s is required by service '%s'", port, service),
	}
	if conflictingProcess != "" {
		details = append(details, fmt.Sprintf("Currently used by: %s", conflictingProcess))
	}

	return &OrkError{
		Op:      "network.allocate",
		Kind:    ErrorNetwork,
		Message: fmt.Sprintf("Port %s is already in use", port),
		Hint:    "Stop the conflicting service or change the port in your ork.yml",
		Details: details,
	}
}

// ErrCircularDependency creates an error for circular service dependencies
func ErrCircularDependency(cycle []string) *OrkError {
	return &OrkError{
		Op:      "service.resolve",
		Kind:    ErrorService,
		Message: "Circular dependency detected",
		Hint:    "Remove the circular dependency from your service configuration",
		Details: []string{
			fmt.Sprintf("Dependency cycle: %s", strings.Join(cycle, " → ")),
		},
	}
}

// ErrInvalidConfig creates an error for invalid configuration
func ErrInvalidConfig(field, reason string) *OrkError {
	return &OrkError{
		Op:      "config.validate",
		Kind:    ErrorValidation,
		Message: fmt.Sprintf("Invalid configuration: %s", field),
		Hint:    reason,
	}
}

// ErrServiceFailed creates an error for when a service fails to start
func ErrServiceFailed(serviceName, reason string) *OrkError {
	return &OrkError{
		Op:      "service.start",
		Kind:    ErrorService,
		Message: fmt.Sprintf("Service '%s' failed to start", serviceName),
		Hint:    "Check service logs with: ork logs " + serviceName,
		Details: []string{reason},
	}
}

// ErrImageNotFound creates an error for missing Docker images
func ErrImageNotFound(imageName string) *OrkError {
	return &OrkError{
		Op:      "docker.image",
		Kind:    ErrorDocker,
		Message: fmt.Sprintf("Docker image not found: %s", imageName),
		Hint:    "Pull the image with: docker pull " + imageName,
		Details: []string{
			"Check if the image name is correct",
			"Verify you have access to the registry",
		},
	}
}

// ============================================================================
// Error Wrapping - Add context to existing errors
// ============================================================================

// Wrap adds context to an existing error
func Wrap(err error, op, message string) error {
	if err == nil {
		return nil
	}

	// If it's already an OrkError, just add to the operation chain
	if orkErr, ok := err.(*OrkError); ok {
		orkErr.Op = op + "." + orkErr.Op
		return orkErr
	}

	// Otherwise create a new OrkError
	return &OrkError{
		Op:      op,
		Err:     err,
		Message: message,
	}
}

// WrapWithHint adds context and a hint to an existing error
func WrapWithHint(err error, op, message, hint string) error {
	if err == nil {
		return nil
	}

	if orkErr, ok := err.(*OrkError); ok {
		orkErr.Op = op + "." + orkErr.Op
		if orkErr.Hint == "" {
			orkErr.Hint = hint
		}
		return orkErr
	}

	return &OrkError{
		Op:      op,
		Err:     err,
		Message: message,
		Hint:    hint,
	}
}

// ============================================================================
// Error Checking Helpers
// ============================================================================

// IsKind checks if an error is of a specific kind
func IsKind(err error, kind ErrorKind) bool {
	if orkErr, ok := err.(*OrkError); ok {
		return orkErr.Kind == kind
	}
	return false
}

// IsDockerError checks if error is Docker-related
func IsDockerError(err error) bool {
	return IsKind(err, ErrorDocker)
}

// IsConfigError checks if the error is configuration-related
func IsConfigError(err error) bool {
	return IsKind(err, ErrorConfig)
}

// IsNetworkError checks if the error is network-related
func IsNetworkError(err error) bool {
	return IsKind(err, ErrorNetwork)
}

// ============================================================================
// Did You Mean - Fuzzy matching for suggestions
// ============================================================================

// FindSuggestions returns similar strings using simple edit distance
// (basic implementation - can be improved with Levenshtein distance)
func FindSuggestions(input string, options []string, maxSuggestions int) []string {
	if len(options) == 0 {
		return nil
	}

	var suggestions []string
	input = strings.ToLower(input)

	// Check for prefix matches first
	for _, option := range options {
		if strings.HasPrefix(strings.ToLower(option), input) {
			suggestions = append(suggestions, option)
			if len(suggestions) >= maxSuggestions {
				return suggestions
			}
		}
	}

	// Check for contents matches
	if len(suggestions) < maxSuggestions {
		for _, option := range options {
			if strings.Contains(strings.ToLower(option), input) {
				// Skip if already in suggestions
				found := false
				for _, s := range suggestions {
					if s == option {
						found = true
						break
					}
				}
				if !found {
					suggestions = append(suggestions, option)
					if len(suggestions) >= maxSuggestions {
						return suggestions
					}
				}
			}
		}
	}

	return suggestions
}
