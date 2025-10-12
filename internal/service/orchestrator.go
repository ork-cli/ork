package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hary-singh/ork/internal/config"
	"github.com/hary-singh/ork/internal/docker"
)

// ============================================================================
// Orchestrator - Parallel Service Management
// ============================================================================

// Orchestrator manages the lifecycle of multiple services with parallel execution
type Orchestrator struct {
	services     map[string]*Service // Map of service name -> Service instance
	dockerClient *docker.Client      // Docker client for operations
	projectName  string              // Project name
	networkID    string              // Network ID for inter-service communication
}

// NewOrchestrator creates a new service orchestrator
func NewOrchestrator(projectName string, dockerClient *docker.Client, networkID string) *Orchestrator {
	return &Orchestrator{
		services:     make(map[string]*Service),
		dockerClient: dockerClient,
		projectName:  projectName,
		networkID:    networkID,
	}
}

// AddService adds a service to the orchestrator
func (o *Orchestrator) AddService(name string, cfg config.Service) {
	o.services[name] = New(name, o.projectName, cfg)
}

// GetService returns a service by name
func (o *Orchestrator) GetService(name string) (*Service, bool) {
	svc, ok := o.services[name]
	return svc, ok
}

// ============================================================================
// Parallel Start with Health Check Waiting
// ============================================================================

// StartServicesInOrder starts services in dependency order with parallel execution
// Services at the same dependency level are started in parallel
// Returns an error if any service fails, rolling back successfully started services
func (o *Orchestrator) StartServicesInOrder(ctx context.Context, orderedServiceNames []string, cfg *config.Config) error {
	// Build dependency levels for parallel execution
	levels, err := o.buildDependencyLevels(orderedServiceNames, cfg.Services)
	if err != nil {
		return fmt.Errorf("failed to build dependency levels: %w", err)
	}

	// Track started services for potential rollback
	startedServices := make([]*Service, 0)

	// Start services level by level
	for levelNum, levelServices := range levels {
		fmt.Printf("📦 Starting level %d: %v\n", levelNum+1, levelServices)

		// Start all services in this level in parallel
		if err := o.startServicesInParallel(ctx, levelServices, &startedServices); err != nil {
			// Rollback on failure
			fmt.Printf("❌ Failed to start services: %v\n", err)
			o.rollbackStartedServices(ctx, startedServices)
			return err
		}

		// Wait for all services in this level to become healthy
		if err := o.waitForHealthy(ctx, levelServices); err != nil {
			// Rollback on health check failure
			fmt.Printf("❌ Health check failed: %v\n", err)
			o.rollbackStartedServices(ctx, startedServices)
			return err
		}
	}

	return nil
}

// ============================================================================
// Private Methods - Dependency Level Building
// ============================================================================

// buildDependencyLevels groups services into levels based on dependencies
// Services in the same level can be started in parallel
func (o *Orchestrator) buildDependencyLevels(orderedServiceNames []string, allServices map[string]config.Service) ([][]string, error) {
	// Return an empty slice if no services to start
	if len(orderedServiceNames) == 0 {
		return [][]string{}, nil
	}

	// Build dependency graph
	graph := make(map[string][]string)
	for name, svc := range allServices {
		graph[name] = svc.DependsOn
	}

	// Track the level of each service
	serviceLevels := make(map[string]int)

	// Calculate levels based on dependencies
	for _, name := range orderedServiceNames {
		o.calculateServiceLevel(name, graph, serviceLevels, make(map[string]bool))
	}

	// Group services by level
	levelGroups := make(map[int][]string)
	maxLevel := 0
	for _, name := range orderedServiceNames {
		level := serviceLevels[name]
		levelGroups[level] = append(levelGroups[level], name)
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Convert to ordered slice
	levels := make([][]string, maxLevel+1)
	for i := 0; i <= maxLevel; i++ {
		levels[i] = levelGroups[i]
	}

	return levels, nil
}

// calculateServiceLevel recursively calculates the dependency level of a service
// Level 0 = no dependencies, Level N = max(dependency levels) + 1
func (o *Orchestrator) calculateServiceLevel(serviceName string, graph map[string][]string, levels map[string]int, visited map[string]bool) int {
	// Return cached level if already calculated
	if level, ok := levels[serviceName]; ok {
		return level
	}

	// Check for circular dependencies
	if visited[serviceName] {
		return 0 // Already handled by dependency resolution
	}
	visited[serviceName] = true

	// Get dependencies
	deps := graph[serviceName]
	if len(deps) == 0 {
		// No dependencies = level 0
		levels[serviceName] = 0
		return 0
	}

	// Calculate level as max(dependency levels) + 1
	maxDepLevel := -1
	for _, dep := range deps {
		depLevel := o.calculateServiceLevel(dep, graph, levels, visited)
		if depLevel > maxDepLevel {
			maxDepLevel = depLevel
		}
	}

	level := maxDepLevel + 1
	levels[serviceName] = level
	return level
}

// ============================================================================
// Private Methods - Parallel Start
// ============================================================================

// startServicesInParallel starts multiple services concurrently
func (o *Orchestrator) startServicesInParallel(ctx context.Context, serviceNames []string, startedServices *[]*Service) error {
	// Use a wait group to track parallel starts
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects concurrent access to startedServices slice
	errChan := make(chan error, len(serviceNames))

	// Start each service in a separate goroutine
	for _, name := range serviceNames {
		wg.Add(1)
		go func(serviceName string) {
			defer wg.Done()

			// Get the service
			svc, ok := o.services[serviceName]
			if !ok {
				errChan <- fmt.Errorf("service %s not found in orchestrator", serviceName)
				return
			}

			// Start the service
			fmt.Printf("🐳 Starting %s...\n", serviceName)
			if err := svc.Start(ctx, o.dockerClient, o.networkID); err != nil {
				errChan <- fmt.Errorf("failed to start %s: %w", serviceName, err)
				return
			}

			fmt.Printf("✅ Started %s (container: %s)\n", serviceName, svc.GetContainerID()[:12])

			// Track successfully started service (protected by mutex)
			mu.Lock()
			*startedServices = append(*startedServices, svc)
			mu.Unlock()
		}(name)
	}

	// Wait for all starts to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// ============================================================================
// Private Methods - Health Check Waiting
// ============================================================================

// waitForHealthy waits for all services to become healthy
func (o *Orchestrator) waitForHealthy(ctx context.Context, serviceNames []string) error {
	// Skip if no services
	if len(serviceNames) == 0 {
		return nil
	}

	// Check if any services have health checks configured
	hasHealthChecks := false
	for _, name := range serviceNames {
		svc, ok := o.services[name]
		if ok && svc.Config.Health != nil {
			hasHealthChecks = true
			break
		}
	}

	// Skip health check waiting if no health checks are configured
	if !hasHealthChecks {
		return nil
	}

	fmt.Printf("🏥 Waiting for services to become healthy...\n")

	// Wait for each service with a health check
	var wg sync.WaitGroup
	errChan := make(chan error, len(serviceNames))

	for _, name := range serviceNames {
		svc, ok := o.services[name]
		if !ok {
			continue
		}

		// Only wait for services with health checks
		if svc.Config.Health == nil {
			continue
		}

		wg.Add(1)
		go func(service *Service) {
			defer wg.Done()

			// Wait for health with timeout
			if err := o.waitForServiceHealth(ctx, service); err != nil {
				errChan <- err
				return
			}

			fmt.Printf("✅ %s is healthy\n", service.Name)
		}(svc)
	}

	// Wait for all health checks
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// waitForServiceHealth waits for a single service to become healthy
func (o *Orchestrator) waitForServiceHealth(ctx context.Context, svc *Service) error {
	// Parse health check interval
	interval := 5 * time.Second
	if svc.Config.Health.Interval != "" {
		if d, err := time.ParseDuration(svc.Config.Health.Interval); err == nil {
			interval = d
		}
	}

	// Maximum wait time (30 seconds)
	maxWait := 30 * time.Second
	deadline := time.Now().Add(maxWait)

	// Poll health until healthy or timeout
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if we've exceeded the deadline
			if time.Now().After(deadline) {
				return fmt.Errorf("service %s did not become healthy within %v", svc.Name, maxWait)
			}

			// Perform health check
			if err := svc.CheckHealth(ctx); err == nil {
				// Service is healthy
				return nil
			}
			// Continue waiting if unhealthy
		}
	}
}

// ============================================================================
// Private Methods - Rollback
// ============================================================================

// rollbackStartedServices stops and removes all successfully started services
func (o *Orchestrator) rollbackStartedServices(ctx context.Context, startedServices []*Service) {
	if len(startedServices) == 0 {
		return
	}

	fmt.Printf("🔄 Rolling back %d started service(s)...\n", len(startedServices))

	// Stop services in reverse order
	for i := len(startedServices) - 1; i >= 0; i-- {
		svc := startedServices[i]
		fmt.Printf("🛑 Rolling back %s...\n", svc.Name)

		if err := svc.Stop(ctx, o.dockerClient); err != nil {
			fmt.Printf("⚠️  Warning: failed to rollback %s: %v\n", svc.Name, err)
		} else {
			fmt.Printf("✅ Rolled back %s\n", svc.Name)
		}
	}
}

// ============================================================================
// Cleanup Methods
// ============================================================================

// StopAll stops all services managed by the orchestrator
func (o *Orchestrator) StopAll(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(o.services))

	// Stop all services in parallel
	for _, svc := range o.services {
		if !svc.IsRunning() {
			continue
		}

		wg.Add(1)
		go func(service *Service) {
			defer wg.Done()

			fmt.Printf("🛑 Stopping %s...\n", service.Name)
			if err := service.Stop(ctx, o.dockerClient); err != nil {
				errChan <- fmt.Errorf("failed to stop %s: %w", service.Name, err)
				return
			}
			fmt.Printf("✅ Stopped %s\n", service.Name)
		}(svc)
	}

	// Wait for all stops to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some services: %v", errors)
	}

	return nil
}
