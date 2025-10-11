package config

import (
	"fmt"
	"strings"
)

// Validate checks if the config is valid and returns helpful error messages
func (c *Config) Validate() error {
	// Check required fields
	if c.Version == "" {
		return fmt.Errorf("version is required in ork.yml")
	}

	if c.Project == "" {
		return fmt.Errorf("project name is required in ork.yml")
	}

	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service must be defined in ork.yml")
	}

	// Validate each service
	for name, service := range c.Services {
		if err := validateService(name, service, c.Services); err != nil {
			return fmt.Errorf("service '%s': %w", name, err)
		}
	}

	return nil
}

// validateService validates a single service definition
func validateService(name string, service Service, allServices map[string]Service) error {
	if err := validateServiceSource(service); err != nil {
		return err
	}

	if err := validateBuildConfig(service); err != nil {
		return err
	}

	if err := validateDependencies(name, service.DependsOn, allServices); err != nil {
		return err
	}

	if err := validatePorts(service.Ports); err != nil {
		return err
	}

	return nil
}

// validateServiceSource ensures exactly one source is specified (git, image, or build)
func validateServiceSource(service Service) error {
	sources := countSources(service)

	if sources == 0 {
		return fmt.Errorf("must specify one of: git, image, or build")
	}

	if sources > 1 {
		return fmt.Errorf("can only specify one of: git, image, or build (found %d)", sources)
	}

	return nil
}

// countSources returns how many sources are configured
func countSources(service Service) int {
	count := 0
	if service.Git != "" {
		count++
	}
	if service.Image != "" {
		count++
	}
	if service.Build != nil {
		count++
	}
	return count
}

// validateBuildConfig ensures build configuration is valid
func validateBuildConfig(service Service) error {
	if service.Build != nil && service.Build.Context == "" {
		return fmt.Errorf("build.context is required when using build")
	}
	return nil
}

// validateDependencies checks that all dependencies exist and no self-dependencies
func validateDependencies(serviceName string, deps []string, allServices map[string]Service) error {
	for _, dep := range deps {
		if dep == serviceName {
			return fmt.Errorf("service cannot depend on itself")
		}

		if _, exists := allServices[dep]; !exists {
			return fmt.Errorf("depends_on references unknown service '%s'", dep)
		}
	}
	return nil
}

// validatePorts ensures port mappings are in the correct format
func validatePorts(ports []string) error {
	for _, port := range ports {
		if !strings.Contains(port, ":") {
			return fmt.Errorf("invalid port format '%s', expected 'host:container' (e.g., '3000:3000')", port)
		}
	}
	return nil
}
