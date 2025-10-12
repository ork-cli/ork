package service

import (
	"fmt"

	"github.com/ork-cli/ork/internal/config"
)

// ============================================================================
// Type Definitions
// ============================================================================

// DependencyGraph represents the service dependency relationships
type DependencyGraph struct {
	services     map[string]config.Service // All services in the project
	dependencies map[string][]string       // Service -> list of dependencies
	dependents   map[string][]string       // Service -> list of services that depend on it
}

// ============================================================================
// Public API
// ============================================================================

// ResolveDependencies resolves service dependencies and returns them in start order
// Returns a list of service names in the order they should be started
// Detects circular dependencies and returns an error if found
func ResolveDependencies(services map[string]config.Service, requestedServices []string) ([]string, error) {
	// Build the dependency graph
	graph := buildDependencyGraph(services)

	// Validate that requested services exist
	if err := validateServices(graph, requestedServices); err != nil {
		return nil, err
	}

	// Collect all services needed (requested and their dependencies)
	allNeeded := collectAllDependencies(graph, requestedServices)

	// Detect circular dependencies
	if err := detectCircularDependencies(graph, allNeeded); err != nil {
		return nil, err
	}

	// Perform topological sort to get the correct start order
	orderedServices := topologicalSort(graph, allNeeded)

	return orderedServices, nil
}

// ============================================================================
// Private Helpers - Graph Building
// ============================================================================

// buildDependencyGraph constructs a dependency graph from service configurations
func buildDependencyGraph(services map[string]config.Service) *DependencyGraph {
	graph := &DependencyGraph{
		services:     services,
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
	}

	// Build dependency relationships
	for serviceName, service := range services {
		// Initialize an empty dependency list for this service
		if graph.dependencies[serviceName] == nil {
			graph.dependencies[serviceName] = []string{}
		}

		// Add each dependency
		for _, dep := range service.DependsOn {
			graph.dependencies[serviceName] = append(graph.dependencies[serviceName], dep)

			// Track reverse relationship (who depends on this service)
			if graph.dependents[dep] == nil {
				graph.dependents[dep] = []string{}
			}
			graph.dependents[dep] = append(graph.dependents[dep], serviceName)
		}
	}

	return graph
}

// ============================================================================
// Private Helpers - Validation
// ============================================================================

// validateServices checks that all requested services exist in the configuration
func validateServices(graph *DependencyGraph, requestedServices []string) error {
	for _, serviceName := range requestedServices {
		if _, exists := graph.services[serviceName]; !exists {
			return fmt.Errorf("service '%s' not found in configuration", serviceName)
		}
	}
	return nil
}

// ============================================================================
// Private Helpers - Dependency Collection
// ============================================================================

// collectAllDependencies recursively collects a service and all its dependencies
func collectAllDependencies(graph *DependencyGraph, requestedServices []string) []string {
	visited := make(map[string]bool)
	var result []string

	var collectDeps func(serviceName string)
	collectDeps = func(serviceName string) {
		// Skip if already visited
		if visited[serviceName] {
			return
		}
		visited[serviceName] = true

		// Recursively collect dependencies first
		for _, dep := range graph.dependencies[serviceName] {
			collectDeps(dep)
		}

		// Add this service
		result = append(result, serviceName)
	}

	// Collect dependencies for all requested services
	for _, serviceName := range requestedServices {
		collectDeps(serviceName)
	}

	return result
}

// ============================================================================
// Private Helpers - Circular Dependency Detection
// ============================================================================

// detectCircularDependencies checks for circular dependencies in the graph
// Uses depth-first search with a recursion stack to detect cycles
func detectCircularDependencies(graph *DependencyGraph, services []string) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool) // Recursion stack for cycle detection
	var path []string                 // Track the path for error reporting

	var detectCycle func(serviceName string) error
	detectCycle = func(serviceName string) error {
		// Mark as visited and add to recursion stack
		visited[serviceName] = true
		recStack[serviceName] = true
		path = append(path, serviceName)

		// Check all dependencies
		for _, dep := range graph.dependencies[serviceName] {
			// If dependency is in recursion stack, we found a cycle
			if recStack[dep] {
				// Build cycle path for an error message
				cyclePath := append(path, dep)
				return fmt.Errorf("circular dependency detected: %v", cyclePath)
			}

			// Recursively check if not visited
			if !visited[dep] {
				if err := detectCycle(dep); err != nil {
					return err
				}
			}
		}

		// Remove from recursion stack and path
		recStack[serviceName] = false
		path = path[:len(path)-1]

		return nil
	}

	// Check each service
	for _, serviceName := range services {
		if !visited[serviceName] {
			if err := detectCycle(serviceName); err != nil {
				return err
			}
		}
	}

	return nil
}

// ============================================================================
// Private Helpers - Topological Sort
// ============================================================================

// topologicalSort performs Kahn's algorithm to get services in dependency order
// Services with no dependencies come first, followed by services that depend on them
func topologicalSort(graph *DependencyGraph, services []string) []string {
	// Create a set of services we care about
	serviceSet := make(map[string]bool)
	for _, s := range services {
		serviceSet[s] = true
	}

	// Calculate in-degree (number of dependencies) for each service
	inDegree := make(map[string]int)
	for _, serviceName := range services {
		// Only count dependencies that are in our service set
		count := 0
		for _, dep := range graph.dependencies[serviceName] {
			if serviceSet[dep] {
				count++
			}
		}
		inDegree[serviceName] = count
	}

	// Initialize queue with services that have no dependencies
	var queue []string
	for _, serviceName := range services {
		if inDegree[serviceName] == 0 {
			queue = append(queue, serviceName)
		}
	}

	// Process queue and build result
	var result []string
	for len(queue) > 0 {
		// Pop from the queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependents
		for _, dependent := range graph.dependents[current] {
			// Only process if this dependent is in our service set
			if !serviceSet[dependent] {
				continue
			}

			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	return result
}
