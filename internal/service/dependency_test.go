package service

import (
	"strings"
	"testing"

	"github.com/ork-cli/ork/internal/config"
)

// ============================================================================
// ResolveDependencies Tests - Success Cases
// ============================================================================

// TestResolveDependencies_NoDependencies tests a simple case with no dependencies
func TestResolveDependencies_NoDependencies(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
		"api": {Image: "node:18"},
	}

	result, err := ResolveDependencies(services, []string{"web"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 service, got %d", len(result))
	}
	if result[0] != "web" {
		t.Errorf("expected 'web', got '%s'", result[0])
	}
}

// TestResolveDependencies_LinearChain tests a linear dependency chain (A -> B -> C)
func TestResolveDependencies_LinearChain(t *testing.T) {
	services := map[string]config.Service{
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

	result, err := ResolveDependencies(services, []string{"frontend"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// Verify order: postgres should come before api, api before frontend
	postgresIdx := indexOf(result, "postgres")
	apiIdx := indexOf(result, "api")
	frontendIdx := indexOf(result, "frontend")

	if postgresIdx == -1 || apiIdx == -1 || frontendIdx == -1 {
		t.Fatalf("missing expected services in result: %v", result)
	}

	if postgresIdx > apiIdx {
		t.Errorf("postgres (index %d) should come before api (index %d)", postgresIdx, apiIdx)
	}
	if apiIdx > frontendIdx {
		t.Errorf("api (index %d) should come before frontend (index %d)", apiIdx, frontendIdx)
	}
}

// TestResolveDependencies_MultipleDependencies tests a service with multiple dependencies
func TestResolveDependencies_MultipleDependencies(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api", "auth"},
		},
		"api": {
			Image: "node:18",
		},
		"auth": {
			Image: "keycloak:latest",
		},
	}

	result, err := ResolveDependencies(services, []string{"frontend"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// Verify frontend comes after both api and auth
	frontendIdx := indexOf(result, "frontend")
	apiIdx := indexOf(result, "api")
	authIdx := indexOf(result, "auth")

	if apiIdx > frontendIdx {
		t.Errorf("api should come before frontend")
	}
	if authIdx > frontendIdx {
		t.Errorf("auth should come before frontend")
	}
}

// TestResolveDependencies_DiamondDependency tests diamond dependency pattern
// A depends on B and C, both B and C depend on D
func TestResolveDependencies_DiamondDependency(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api", "cache"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"database"},
		},
		"cache": {
			Image:     "redis:alpine",
			DependsOn: []string{"database"},
		},
		"database": {
			Image: "postgres:15",
		},
	}

	result, err := ResolveDependencies(services, []string{"frontend"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 4 {
		t.Errorf("expected 4 services, got %d", len(result))
	}

	// Verify database comes first, frontend comes last
	dbIdx := indexOf(result, "database")
	apiIdx := indexOf(result, "api")
	cacheIdx := indexOf(result, "cache")
	frontendIdx := indexOf(result, "frontend")

	if dbIdx > apiIdx || dbIdx > cacheIdx {
		t.Errorf("database should come before api and cache")
	}
	if apiIdx > frontendIdx || cacheIdx > frontendIdx {
		t.Errorf("api and cache should come before frontend")
	}
}

// TestResolveDependencies_PartialServiceRequest tests requesting only some services
func TestResolveDependencies_PartialServiceRequest(t *testing.T) {
	services := map[string]config.Service{
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
			Image: "redis:alpine",
		},
	}

	// Request only api (should include postgres but not frontend or redis)
	result, err := ResolveDependencies(services, []string{"api"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 services (api and postgres), got %d: %v", len(result), result)
	}

	if !contains(result, "api") {
		t.Errorf("expected 'api' in result")
	}
	if !contains(result, "postgres") {
		t.Errorf("expected 'postgres' in result")
	}
	if contains(result, "frontend") {
		t.Errorf("did not expect 'frontend' in result")
	}
	if contains(result, "redis") {
		t.Errorf("did not expect 'redis' in result")
	}
}

// TestResolveDependencies_MultipleRequestedServices tests requesting multiple services
func TestResolveDependencies_MultipleRequestedServices(t *testing.T) {
	services := map[string]config.Service{
		"web": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"db"},
		},
		"worker": {
			Image:     "python:3.11",
			DependsOn: []string{"db", "redis"},
		},
		"db": {
			Image: "postgres:15",
		},
		"redis": {
			Image: "redis:alpine",
		},
	}

	result, err := ResolveDependencies(services, []string{"web", "worker"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should include: db, redis, api, web, worker
	if len(result) != 5 {
		t.Errorf("expected 5 services, got %d: %v", len(result), result)
	}

	// Verify all expected services are present
	expectedServices := []string{"web", "api", "worker", "db", "redis"}
	for _, svc := range expectedServices {
		if !contains(result, svc) {
			t.Errorf("expected '%s' in result", svc)
		}
	}

	// Verify ordering constraints
	dbIdx := indexOf(result, "db")
	apiIdx := indexOf(result, "api")
	webIdx := indexOf(result, "web")
	redisIdx := indexOf(result, "redis")
	workerIdx := indexOf(result, "worker")

	if dbIdx > apiIdx {
		t.Errorf("db should come before api")
	}
	if apiIdx > webIdx {
		t.Errorf("api should come before web")
	}
	if dbIdx > workerIdx || redisIdx > workerIdx {
		t.Errorf("db and redis should come before worker")
	}
}

// TestResolveDependencies_AllServices tests requesting all services in a complex graph
func TestResolveDependencies_AllServices(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine", DependsOn: []string{"api"}},
		"api":      {Image: "node:18", DependsOn: []string{"db"}},
		"db":       {Image: "postgres:15"},
	}

	result, err := ResolveDependencies(services, []string{"frontend", "api", "db"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}
}

// TestResolveDependencies_IndependentServices tests multiple independent services
func TestResolveDependencies_IndependentServices(t *testing.T) {
	services := map[string]config.Service{
		"web":   {Image: "nginx:alpine"},
		"api":   {Image: "node:18"},
		"cache": {Image: "redis:alpine"},
	}

	result, err := ResolveDependencies(services, []string{"web", "api", "cache"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// All three should be present (order doesn't matter for independent services)
	if !contains(result, "web") || !contains(result, "api") || !contains(result, "cache") {
		t.Errorf("expected all three services in result, got: %v", result)
	}
}

// ============================================================================
// ResolveDependencies Tests - Error Cases
// ============================================================================

// TestResolveDependencies_UnknownService tests error when requesting unknown service
func TestResolveDependencies_UnknownService(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}

	_, err := ResolveDependencies(services, []string{"api"})
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}

	if !strings.Contains(err.Error(), "service 'api' not found") {
		t.Errorf("expected 'service not found' error, got: %v", err)
	}
}

// TestResolveDependencies_MultipleUnknownServices tests multiple unknown services
func TestResolveDependencies_MultipleUnknownServices(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}

	_, err := ResolveDependencies(services, []string{"web", "api", "db"})
	if err == nil {
		t.Fatal("expected error for unknown services, got nil")
	}

	// Should fail on first unknown service
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestResolveDependencies_CircularDependencySimple tests simple circular dependency (A -> B -> A)
func TestResolveDependencies_CircularDependencySimple(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"frontend"},
		},
	}

	_, err := ResolveDependencies(services, []string{"frontend"})
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestResolveDependencies_CircularDependencyComplex tests complex circular (A -> B -> C -> A)
func TestResolveDependencies_CircularDependencyComplex(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"database"},
		},
		"database": {
			Image:     "postgres:15",
			DependsOn: []string{"frontend"}, // Creates cycle
		},
	}

	_, err := ResolveDependencies(services, []string{"frontend"})
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestResolveDependencies_SelfDependency tests service depending on itself
func TestResolveDependencies_SelfDependency(t *testing.T) {
	services := map[string]config.Service{
		"api": {
			Image:     "node:18",
			DependsOn: []string{"api"}, // Self-dependency
		},
	}

	_, err := ResolveDependencies(services, []string{"api"})
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestResolveDependencies_EmptyServices tests with an empty service map
func TestResolveDependencies_EmptyServices(t *testing.T) {
	services := map[string]config.Service{}

	_, err := ResolveDependencies(services, []string{"web"})
	if err == nil {
		t.Fatal("expected error for empty services, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestResolveDependencies_EmptyRequestedServices tests with empty requested services
func TestResolveDependencies_EmptyRequestedServices(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}

	result, err := ResolveDependencies(services, []string{})
	if err != nil {
		t.Fatalf("expected no error for empty requested services, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d services: %v", len(result), result)
	}
}

// TestResolveDependencies_MissingTransitiveDependency tests when a dependency references an unknown service
func TestResolveDependencies_MissingTransitiveDependency(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"postgres"}, // postgres doesn't exist
		},
	}

	// Note: Current implementation doesn't validate transitive dependencies exist
	// This might be a bug or by design - documenting current behavior
	result, err := ResolveDependencies(services, []string{"frontend"})

	// The function should handle this gracefully
	if err != nil {
		t.Logf("Got error (might be expected): %v", err)
	} else {
		// If no error, postgres should be silently ignored
		t.Logf("No error returned, result: %v", result)
	}
}

// ============================================================================
// buildDependencyGraph Tests
// ============================================================================

// TestBuildDependencyGraph_EmptyServices tests building graph from empty services
func TestBuildDependencyGraph_EmptyServices(t *testing.T) {
	services := map[string]config.Service{}
	graph := buildDependencyGraph(services)

	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.services) != 0 {
		t.Errorf("expected 0 services, got %d", len(graph.services))
	}
	if len(graph.dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(graph.dependencies))
	}
}

// TestBuildDependencyGraph_NoDependencies tests graph with independent services
func TestBuildDependencyGraph_NoDependencies(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
		"api": {Image: "node:18"},
	}
	graph := buildDependencyGraph(services)

	if len(graph.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(graph.services))
	}

	// Each service should have an empty dependency list
	if len(graph.dependencies["web"]) != 0 {
		t.Errorf("expected web to have 0 dependencies, got %d", len(graph.dependencies["web"]))
	}
	if len(graph.dependencies["api"]) != 0 {
		t.Errorf("expected api to have 0 dependencies, got %d", len(graph.dependencies["api"]))
	}
}

// TestBuildDependencyGraph_WithDependencies tests graph building with dependencies
func TestBuildDependencyGraph_WithDependencies(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {
			Image:     "nginx:alpine",
			DependsOn: []string{"api", "auth"},
		},
		"api": {
			Image:     "node:18",
			DependsOn: []string{"db"},
		},
		"auth": {
			Image: "keycloak:latest",
		},
		"db": {
			Image: "postgres:15",
		},
	}
	graph := buildDependencyGraph(services)

	// Check dependencies map
	if len(graph.dependencies["frontend"]) != 2 {
		t.Errorf("expected frontend to have 2 dependencies, got %d", len(graph.dependencies["frontend"]))
	}
	if !contains(graph.dependencies["frontend"], "api") {
		t.Errorf("expected frontend to depend on api")
	}
	if !contains(graph.dependencies["frontend"], "auth") {
		t.Errorf("expected frontend to depend on auth")
	}

	// Check dependents map (reverse relationships)
	if len(graph.dependents["api"]) != 1 {
		t.Errorf("expected api to have 1 dependent, got %d", len(graph.dependents["api"]))
	}
	if !contains(graph.dependents["api"], "frontend") {
		t.Errorf("expected api to be depended on by frontend")
	}

	if len(graph.dependents["db"]) != 1 {
		t.Errorf("expected db to have 1 dependent, got %d", len(graph.dependents["db"]))
	}
	if !contains(graph.dependents["db"], "api") {
		t.Errorf("expected db to be depended on by api")
	}
}

// ============================================================================
// validateServices Tests
// ============================================================================

// TestValidateServices_AllExist tests validation when all services exist
func TestValidateServices_AllExist(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
		"api": {Image: "node:18"},
	}
	graph := buildDependencyGraph(services)

	err := validateServices(graph, []string{"web", "api"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestValidateServices_ServiceNotFound tests validation with unknown service
func TestValidateServices_ServiceNotFound(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}
	graph := buildDependencyGraph(services)

	err := validateServices(graph, []string{"api"})
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}

	if !strings.Contains(err.Error(), "service 'api' not found") {
		t.Errorf("expected 'service not found' error, got: %v", err)
	}
}

// TestValidateServices_EmptyRequested tests validation with an empty list
func TestValidateServices_EmptyRequested(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}
	graph := buildDependencyGraph(services)

	err := validateServices(graph, []string{})
	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
}

// ============================================================================
// collectAllDependencies Tests
// ============================================================================

// TestCollectAllDependencies_NoDependencies tests collecting with no dependencies
func TestCollectAllDependencies_NoDependencies(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}
	graph := buildDependencyGraph(services)

	result := collectAllDependencies(graph, []string{"web"})
	if len(result) != 1 {
		t.Errorf("expected 1 service, got %d", len(result))
	}
	if result[0] != "web" {
		t.Errorf("expected 'web', got '%s'", result[0])
	}
}

// TestCollectAllDependencies_WithDependencies tests collecting with dependencies
func TestCollectAllDependencies_WithDependencies(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine", DependsOn: []string{"api"}},
		"api":      {Image: "node:18", DependsOn: []string{"db"}},
		"db":       {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := collectAllDependencies(graph, []string{"frontend"})
	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// All three should be present
	if !contains(result, "frontend") || !contains(result, "api") || !contains(result, "db") {
		t.Errorf("expected all services in result, got: %v", result)
	}
}

// TestCollectAllDependencies_MultipleBranches tests collecting with multiple dependency branches
func TestCollectAllDependencies_MultipleBranches(t *testing.T) {
	services := map[string]config.Service{
		"web":   {Image: "nginx:alpine", DependsOn: []string{"api", "cache"}},
		"api":   {Image: "node:18", DependsOn: []string{"db"}},
		"cache": {Image: "redis:alpine"},
		"db":    {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := collectAllDependencies(graph, []string{"web"})
	if len(result) != 4 {
		t.Errorf("expected 4 services, got %d: %v", len(result), result)
	}
}

// TestCollectAllDependencies_SharedDependencies tests that shared dependencies are not duplicated
func TestCollectAllDependencies_SharedDependencies(t *testing.T) {
	services := map[string]config.Service{
		"web":    {Image: "nginx:alpine", DependsOn: []string{"db"}},
		"worker": {Image: "python:3.11", DependsOn: []string{"db"}},
		"db":     {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := collectAllDependencies(graph, []string{"web", "worker"})

	// Should have exactly 3 services, not 4 (db should appear only once)
	if len(result) != 3 {
		t.Errorf("expected 3 services (no duplicates), got %d: %v", len(result), result)
	}

	// Verify db appears exactly once
	dbCount := 0
	for _, svc := range result {
		if svc == "db" {
			dbCount++
		}
	}
	if dbCount != 1 {
		t.Errorf("expected db to appear once, got %d times", dbCount)
	}
}

// ============================================================================
// detectCircularDependencies Tests
// ============================================================================

// TestDetectCircularDependencies_NoCycle tests detection with no cycles
func TestDetectCircularDependencies_NoCycle(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine", DependsOn: []string{"api"}},
		"api":      {Image: "node:18", DependsOn: []string{"db"}},
		"db":       {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	err := detectCircularDependencies(graph, []string{"frontend", "api", "db"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestDetectCircularDependencies_SimpleCycle tests simple cycle detection (A -> B -> A)
func TestDetectCircularDependencies_SimpleCycle(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine", DependsOn: []string{"b"}},
		"b": {Image: "node:18", DependsOn: []string{"a"}},
	}
	graph := buildDependencyGraph(services)

	err := detectCircularDependencies(graph, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestDetectCircularDependencies_SelfCycle tests self-referencing cycle
func TestDetectCircularDependencies_SelfCycle(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine", DependsOn: []string{"a"}},
	}
	graph := buildDependencyGraph(services)

	err := detectCircularDependencies(graph, []string{"a"})
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestDetectCircularDependencies_LongCycle tests longer cycle (A -> B -> C -> D -> A)
func TestDetectCircularDependencies_LongCycle(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine", DependsOn: []string{"b"}},
		"b": {Image: "node:18", DependsOn: []string{"c"}},
		"c": {Image: "redis:alpine", DependsOn: []string{"d"}},
		"d": {Image: "postgres:15", DependsOn: []string{"a"}},
	}
	graph := buildDependencyGraph(services)

	err := detectCircularDependencies(graph, []string{"a", "b", "c", "d"})
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' error, got: %v", err)
	}
}

// TestDetectCircularDependencies_PartialCycle tests cycle in part of the graph
func TestDetectCircularDependencies_PartialCycle(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine", DependsOn: []string{"api"}},
		"api":      {Image: "node:18", DependsOn: []string{"cache"}},
		"cache":    {Image: "redis:alpine", DependsOn: []string{"api"}}, // Cycle here
		"db":       {Image: "postgres:15"},                              // Independent
	}
	graph := buildDependencyGraph(services)

	err := detectCircularDependencies(graph, []string{"frontend", "api", "cache"})
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	// db should not be involved in the error
	err2 := detectCircularDependencies(graph, []string{"db"})
	if err2 != nil {
		t.Errorf("expected no error for independent service, got: %v", err2)
	}
}

// ============================================================================
// topologicalSort Tests
// ============================================================================

// TestTopologicalSort_LinearChain tests sorting a linear dependency chain
func TestTopologicalSort_LinearChain(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine", DependsOn: []string{"b"}},
		"b": {Image: "node:18", DependsOn: []string{"c"}},
		"c": {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := topologicalSort(graph, []string{"a", "b", "c"})

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// c should come first, then b, then a
	if result[0] != "c" {
		t.Errorf("expected 'c' first, got '%s'", result[0])
	}
	if result[2] != "a" {
		t.Errorf("expected 'a' last, got '%s'", result[2])
	}
}

// TestTopologicalSort_IndependentServices tests sorting independent services
func TestTopologicalSort_IndependentServices(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine"},
		"b": {Image: "node:18"},
		"c": {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := topologicalSort(graph, []string{"a", "b", "c"})

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	// All should be present (order doesn't matter for independent services)
	if !contains(result, "a") || !contains(result, "b") || !contains(result, "c") {
		t.Errorf("expected all services in result, got: %v", result)
	}
}

// TestTopologicalSort_DiamondPattern tests sorting a diamond dependency pattern
func TestTopologicalSort_DiamondPattern(t *testing.T) {
	services := map[string]config.Service{
		"a": {Image: "nginx:alpine", DependsOn: []string{"b", "c"}},
		"b": {Image: "node:18", DependsOn: []string{"d"}},
		"c": {Image: "redis:alpine", DependsOn: []string{"d"}},
		"d": {Image: "postgres:15"},
	}
	graph := buildDependencyGraph(services)

	result := topologicalSort(graph, []string{"a", "b", "c", "d"})

	if len(result) != 4 {
		t.Errorf("expected 4 services, got %d", len(result))
	}

	// d must come first, a must come last
	if result[0] != "d" {
		t.Errorf("expected 'd' first, got '%s'", result[0])
	}
	if result[3] != "a" {
		t.Errorf("expected 'a' last, got '%s'", result[3])
	}

	// b and c must come before a
	aIdx := indexOf(result, "a")
	bIdx := indexOf(result, "b")
	cIdx := indexOf(result, "c")

	if bIdx > aIdx || cIdx > aIdx {
		t.Errorf("b and c should come before a")
	}
}

// TestTopologicalSort_PartialGraph tests sorting only a subset of services
func TestTopologicalSort_PartialGraph(t *testing.T) {
	services := map[string]config.Service{
		"frontend": {Image: "nginx:alpine", DependsOn: []string{"api"}},
		"api":      {Image: "node:18", DependsOn: []string{"db"}},
		"db":       {Image: "postgres:15"},
		"cache":    {Image: "redis:alpine"}, // Not in our subset
	}
	graph := buildDependencyGraph(services)

	result := topologicalSort(graph, []string{"frontend", "api", "db"})

	if len(result) != 3 {
		t.Errorf("expected 3 services, got %d", len(result))
	}

	if contains(result, "cache") {
		t.Errorf("did not expect 'cache' in result")
	}
}

// TestTopologicalSort_EmptyList tests sorting with an empty service list
func TestTopologicalSort_EmptyList(t *testing.T) {
	services := map[string]config.Service{
		"web": {Image: "nginx:alpine"},
	}
	graph := buildDependencyGraph(services)

	result := topologicalSort(graph, []string{})

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d services", len(result))
	}
}

// ============================================================================
// Helper Functions for Tests
// ============================================================================

// indexOf returns the index of an element in a slice, or -1 if not found
func indexOf(slice []string, element string) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return -1
}

// contains checks if a slice contains a specific element
func contains(slice []string, element string) bool {
	return indexOf(slice, element) != -1
}
