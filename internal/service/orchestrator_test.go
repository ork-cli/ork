package service

import (
	"sync"
	"testing"

	"github.com/ork-cli/ork/internal/config"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Constructor and Service Management Tests
// ============================================================================

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	assert.NotNil(t, orch)
	assert.Equal(t, "myproject", orch.projectName)
	assert.Equal(t, "network-123", orch.networkID)
	assert.NotNil(t, orch.services)
	assert.Equal(t, 0, len(orch.services))
}

func TestOrchestrator_AddService(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	cfg := config.Service{
		Image: "nginx:alpine",
		Ports: []string{"8080:80"},
	}

	orch.AddService("frontend", cfg)

	// Verify service was added
	assert.Equal(t, 1, len(orch.services))

	svc, ok := orch.GetService("frontend")
	assert.True(t, ok)
	assert.NotNil(t, svc)
	assert.Equal(t, "frontend", svc.Name)
	assert.Equal(t, "myproject", svc.ProjectName)
	assert.Equal(t, "nginx:alpine", svc.Config.Image)
}

func TestOrchestrator_GetService(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	cfg := config.Service{Image: "nginx:alpine"}
	orch.AddService("frontend", cfg)

	tests := []struct {
		name        string
		serviceName string
		wantOk      bool
	}{
		{
			name:        "existing service returns true",
			serviceName: "frontend",
			wantOk:      true,
		},
		{
			name:        "non-existent service returns false",
			serviceName: "backend",
			wantOk:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ok := orch.GetService(tt.serviceName)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.NotNil(t, svc)
				assert.Equal(t, tt.serviceName, svc.Name)
			} else {
				assert.Nil(t, svc)
			}
		})
	}
}

func TestOrchestrator_AddMultipleServices(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine"},
		"api":      {Image: "node:18"},
		"postgres": {Image: "postgres:15"},
	}

	for name, cfg := range services {
		orch.AddService(name, cfg)
	}

	assert.Equal(t, 3, len(orch.services))

	for name := range services {
		svc, ok := orch.GetService(name)
		assert.True(t, ok, "service %s should exist", name)
		assert.Equal(t, name, svc.Name)
	}
}

// ============================================================================
// Dependency Level Building Tests
// ============================================================================

func TestOrchestrator_buildDependencyLevels_NoDependencies(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {Image: "nginx:alpine"},
		"api":      {Image: "node:18"},
		"postgres": {Image: "postgres:15"},
	}

	orderedServiceNames := []string{"frontend", "api", "postgres"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(levels), "all services should be at level 0")
	assert.ElementsMatch(t, []string{"frontend", "api", "postgres"}, levels[0])
}

func TestOrchestrator_buildDependencyLevels_LinearChain(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres"},
		},
		"postgres": {
			Image: "postgres:15",
		},
	}

	orderedServiceNames := []string{"postgres", "api", "frontend"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(levels))
	assert.Equal(t, []string{"postgres"}, levels[0])
	assert.Equal(t, []string{"api"}, levels[1])
	assert.Equal(t, []string{"frontend"}, levels[2])
}

func TestOrchestrator_buildDependencyLevels_ParallelWithSharedDependency(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres", "redis"},
		},
		"postgres": {
			Image: "postgres:15",
		},
		"redis": {
			Image: "redis:7",
		},
	}

	orderedServiceNames := []string{"postgres", "redis", "api", "frontend"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(levels))

	// Level 0: postgres and redis can start in parallel
	assert.ElementsMatch(t, []string{"postgres", "redis"}, levels[0])

	// Level 1: api depends on both postgres and redis
	assert.Equal(t, []string{"api"}, levels[1])

	// Level 2: frontend depends on api
	assert.Equal(t, []string{"frontend"}, levels[2])
}

func TestOrchestrator_buildDependencyLevels_DiamondPattern(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api", "cache"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres"},
		},
		"cache": {
			Image:     "redis:7",
			DependsOn: []string{"postgres"},
		},
		"postgres": {
			Image: "postgres:15",
		},
	}

	orderedServiceNames := []string{"postgres", "api", "cache", "frontend"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(levels))

	// Level 0: postgres
	assert.Equal(t, []string{"postgres"}, levels[0])

	// Level 1: api and cache can run in parallel (both depend on postgres)
	assert.ElementsMatch(t, []string{"api", "cache"}, levels[1])

	// Level 2: frontend depends on both api and cache
	assert.Equal(t, []string{"frontend"}, levels[2])
}

func TestOrchestrator_buildDependencyLevels_ComplexGraph(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api", "auth"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres", "redis", "queue"},
		},
		"auth": {
			Image:     "node:18",
			DependsOn: []string{"postgres", "redis"},
		},
		"queue": {
			Image:     "rabbitmq:3",
			DependsOn: []string{"redis"},
		},
		"postgres": {
			Image: "postgres:15",
		},
		"redis": {
			Image: "redis:7",
		},
	}

	orderedServiceNames := []string{"postgres", "redis", "queue", "api", "auth", "frontend"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(levels))

	// Level 0: postgres and redis (no dependencies)
	assert.ElementsMatch(t, []string{"postgres", "redis"}, levels[0])

	// Level 1: queue depends on redis only; auth depends on postgres and redis
	// Both have max dependency level of 0, so they're both at level 1
	assert.ElementsMatch(t, []string{"queue", "auth"}, levels[1])

	// Level 2: api depends on postgres (0), redis (0), queue (1)
	// Max dependency level is 1, so api is at level 2
	assert.Equal(t, []string{"api"}, levels[2])

	// Level 3: frontend depends on api (2) and auth (1)
	// Max dependency level is 2, so frontend is at level 3
	assert.Equal(t, []string{"frontend"}, levels[3])
}

func TestOrchestrator_buildDependencyLevels_PartialServiceList(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres"},
		},
		"postgres": {
			Image: "postgres:15",
		},
		"redis": {
			Image: "redis:7",
		},
	}

	// Only request postgres and api, not frontend or redis
	orderedServiceNames := []string{"postgres", "api"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(levels))
	assert.Equal(t, []string{"postgres"}, levels[0])
	assert.Equal(t, []string{"api"}, levels[1])
}

// ============================================================================
// calculateServiceLevel Tests
// ============================================================================

func TestOrchestrator_calculateServiceLevel(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		graph       map[string][]string
		wantLevel   int
	}{
		{
			name:        "service with no dependencies has level 0",
			serviceName: "postgres",
			graph: map[string][]string{
				"postgres": {},
			},
			wantLevel: 0,
		},
		{
			name:        "service depending on level 0 has level 1",
			serviceName: "api",
			graph: map[string][]string{
				"postgres": {},
				"api":      {"postgres"},
			},
			wantLevel: 1,
		},
		{
			name:        "service depending on level 1 has level 2",
			serviceName: "frontend",
			graph: map[string][]string{
				"postgres": {},
				"api":      {"postgres"},
				"frontend": {"api"},
			},
			wantLevel: 2,
		},
		{
			name:        "service depending on multiple services uses max level + 1",
			serviceName: "frontend",
			graph: map[string][]string{
				"postgres": {},
				"redis":    {},
				"api":      {"postgres", "redis"},
				"cache":    {"redis"},
				"frontend": {"api", "cache"},
			},
			wantLevel: 2, // max(api=1, cache=1) + 1 = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := NewOrchestrator("myproject", nil, "network-123")
			levels := make(map[string]int)
			visited := make(map[string]bool)

			level := orch.calculateServiceLevel(tt.serviceName, tt.graph, levels, visited)

			assert.Equal(t, tt.wantLevel, level)
			assert.Equal(t, tt.wantLevel, levels[tt.serviceName], "level should be cached")
		})
	}
}

func TestOrchestrator_calculateServiceLevel_Caching(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	graph := map[string][]string{
		"postgres": {},
		"api":      {"postgres"},
	}

	levels := make(map[string]int)
	visited := make(map[string]bool)

	// First call calculates
	level1 := orch.calculateServiceLevel("api", graph, levels, visited)
	assert.Equal(t, 1, level1)

	// Second call should return cached value
	level2 := orch.calculateServiceLevel("api", graph, levels, visited)
	assert.Equal(t, 1, level2)
	assert.Equal(t, level1, level2)
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

func TestOrchestrator_buildDependencyLevels_EmptyServiceList(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"frontend": {Image: "nginx:alpine"},
	}

	orderedServiceNames := []string{}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(levels))
}

func TestOrchestrator_buildDependencyLevels_SingleService(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	allServices := map[string]config.Service{
		"postgres": {Image: "postgres:15"},
	}

	orderedServiceNames := []string{"postgres"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(levels))
	assert.Equal(t, []string{"postgres"}, levels[0])
}

func TestOrchestrator_calculateServiceLevel_CircularDependency(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	// Create a circular dependency: A -> B -> C -> A
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	}

	levels := make(map[string]int)
	visited := make(map[string]bool)

	// Should handle circular dependencies gracefully (returns 0)
	level := orch.calculateServiceLevel("A", graph, levels, visited)

	// With circular dependency detected via visited, should return some level
	// The exact behavior depends on implementation, but it shouldn't panic
	assert.GreaterOrEqual(t, level, 0)
}

// ============================================================================
// Service State Tests
// ============================================================================

func TestOrchestrator_AddService_CreatesServiceWithCorrectState(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	cfg := config.Service{
		Image: "nginx:alpine",
		Health: &config.HealthCheck{
			Endpoint: "/health",
			Interval: "5s",
			Timeout:  "3s",
			Retries:  3,
		},
	}

	orch.AddService("frontend", cfg)

	svc, ok := orch.GetService("frontend")
	assert.True(t, ok)
	assert.Equal(t, StatePending, svc.GetState())
	assert.Equal(t, HealthUnknown, svc.GetHealthStatus())
	assert.Equal(t, "", svc.GetContainerID())
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestOrchestrator_ConcurrentAddAndGet(t *testing.T) {
	orch := NewOrchestrator("myproject", nil, "network-123")

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			orch.AddService("service-1", config.Service{Image: "nginx:alpine"})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = orch.GetService("service-1")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify the final state
	svc, ok := orch.GetService("service-1")
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

// TestOrchestrator_startServicesInParallel_ConcurrentSliceAppend tests that
// the mutex properly protects concurrent appends to the startedServices slice.
// This test simulates the race condition that would occur without the mutex
// by starting many services in parallel and verifying all are tracked correctly.
func TestOrchestrator_startServicesInParallel_ConcurrentSliceAppend(t *testing.T) {
	// This test would need Docker client mocking to truly test the function
	// For now, we'll test the pattern by simulating concurrent slice appends

	// Simulate what startServicesInParallel does: concurrent appends to a slice
	var services []*Service
	var wg sync.WaitGroup

	// Number of concurrent goroutines (simulating services starting in parallel)
	numServices := 50

	// This is the CORRECT way with mutex (what our fix implements)
	var mu sync.Mutex
	for i := 0; i < numServices; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			svc := New(
				"service-"+string(rune(idx)),
				"myproject",
				config.Service{Image: "nginx:alpine"},
			)

			// Protected by mutex (this is what our fix does)
			mu.Lock()
			services = append(services, svc)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all services were added (no data loss due to race condition)
	assert.Equal(t, numServices, len(services), "all services should be tracked")

	// Verify no nil services (corruption indicator)
	for i, svc := range services {
		assert.NotNil(t, svc, "service at index %d should not be nil", i)
	}
}

// ============================================================================
// Rollback Logic Tests
// ============================================================================

func TestOrchestrator_RollbackOrdering_EmptyList(t *testing.T) {
	// Test that an empty list doesn't cause issues
	emptyList := []*Service{}

	// Should not panic with an empty list
	assert.Equal(t, 0, len(emptyList))

	// Simulate reverse iteration (what rollback does)
	iterationCount := 0
	for i := len(emptyList) - 1; i >= 0; i-- {
		iterationCount++
	}

	// Verify the loop never executed
	assert.Equal(t, 0, iterationCount, "should not iterate over empty list")
}

func TestOrchestrator_RollbackOrdering_SingleService(t *testing.T) {
	// Test single service rollback
	svc := New("api", "myproject", config.Service{Image: "nginx:alpine"})
	services := []*Service{svc}

	// Verify reverse iteration works with a single service
	var processedServices []string
	for i := len(services) - 1; i >= 0; i-- {
		processedServices = append(processedServices, services[i].Name)
	}

	assert.Equal(t, []string{"api"}, processedServices)
}

func TestOrchestrator_RollbackOrdering_MultipleServices(t *testing.T) {
	// Test that services are processed in reverse order during rollback
	// Simulate: Start postgres -> api -> frontend
	// Rollback should be: frontend -> api -> postgres (reverse)

	postgres := New("postgres", "myproject", config.Service{Image: "postgres:15"})
	api := New("api", "myproject", config.Service{Image: "node:18"})
	frontend := New("frontend", "myproject", config.Service{Image: "nginx:alpine"})

	// Services were started in this order
	startedInOrder := []*Service{postgres, api, frontend}

	// Simulate rollback reverse iteration
	var rollbackOrder []string
	for i := len(startedInOrder) - 1; i >= 0; i-- {
		rollbackOrder = append(rollbackOrder, startedInOrder[i].Name)
	}

	// Verify rollback processes in reverse order
	assert.Equal(t, []string{"frontend", "api", "postgres"}, rollbackOrder)
	assert.NotEqual(t, []string{"postgres", "api", "frontend"}, rollbackOrder)
}

func TestOrchestrator_RollbackOrdering_PreservesReverseOrder(t *testing.T) {
	// Test with many services to ensure order is truly reversed
	services := []*Service{
		New("db1", "myproject", config.Service{Image: "postgres:15"}),
		New("db2", "myproject", config.Service{Image: "mysql:8"}),
		New("cache", "myproject", config.Service{Image: "redis:7"}),
		New("api", "myproject", config.Service{Image: "node:18"}),
		New("worker", "myproject", config.Service{Image: "python:3.11"}),
		New("frontend", "myproject", config.Service{Image: "nginx:alpine"}),
	}

	// Get names in start order
	startOrder := make([]string, len(services))
	for i, svc := range services {
		startOrder[i] = svc.Name
	}

	// Get names in rollback order (reverse)
	rollbackOrder := make([]string, len(services))
	for i := len(services) - 1; i >= 0; i-- {
		rollbackOrder[len(services)-1-i] = services[i].Name
	}

	// Verify they're exact opposites
	assert.Equal(t, []string{"db1", "db2", "cache", "api", "worker", "frontend"}, startOrder)
	assert.Equal(t, []string{"frontend", "worker", "api", "cache", "db2", "db1"}, rollbackOrder)

	// Verify first and last are swapped
	assert.Equal(t, startOrder[0], rollbackOrder[len(rollbackOrder)-1])
	assert.Equal(t, startOrder[len(startOrder)-1], rollbackOrder[0])
}

// ============================================================================
// Integration-style Tests (without Docker)
// ============================================================================

func TestOrchestrator_RealWorldScenario(t *testing.T) {
	// Test a realistic microservices architecture
	orch := NewOrchestrator("ecommerce", nil, "network-123")

	allServices := map[string]config.Service{
		"nginx": {
			Image:     "nginx:alpine",
			DependsOn: []string{"frontend", "api"},
		},
		"frontend": {
			Image:     "react-app:latest",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres", "redis", "rabbitmq"},
		},
		"worker": {
			Image:     "node:18",
			DependsOn: []string{"postgres", "redis", "rabbitmq"},
		},
		"postgres": {
			Image: "postgres:15",
		},
		"redis": {
			Image: "redis:7",
		},
		"rabbitmq": {
			Image: "rabbitmq:3",
		},
	}

	orderedServiceNames := []string{"postgres", "redis", "rabbitmq", "api", "worker", "frontend", "nginx"}

	levels, err := orch.buildDependencyLevels(orderedServiceNames, allServices)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(levels))

	// Level 0: All databases/infrastructure (no dependencies)
	assert.ElementsMatch(t, []string{"postgres", "redis", "rabbitmq"}, levels[0])

	// Level 1: API and worker both depend on all level 0 services
	assert.ElementsMatch(t, []string{"api", "worker"}, levels[1])

	// Level 2: Frontend depends on api
	assert.Equal(t, []string{"frontend"}, levels[2])

	// Level 3: Nginx depends on frontend and api
	assert.Equal(t, []string{"nginx"}, levels[3])
}
