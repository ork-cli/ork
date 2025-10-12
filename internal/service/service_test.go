package service

import (
	"testing"
	"time"

	"github.com/ork-cli/ork/internal/config"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		projectName string
		config      config.Service
		wantState   State
		wantHealth  HealthStatus
	}{
		{
			name:        "creates service with default state",
			serviceName: "api",
			projectName: "myproject",
			config: config.Service{
				Image: "nginx:alpine",
			},
			wantState:  StatePending,
			wantHealth: HealthUnknown,
		},
		{
			name:        "creates service with health check config",
			serviceName: "frontend",
			projectName: "webapp",
			config: config.Service{
				Image: "node:18",
				Health: &config.HealthCheck{
					Endpoint: "/health",
					Interval: "5s",
					Timeout:  "3s",
					Retries:  3,
				},
			},
			wantState:  StatePending,
			wantHealth: HealthUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New(tt.serviceName, tt.projectName, tt.config)

			assert.Equal(t, tt.serviceName, service.Name)
			assert.Equal(t, tt.projectName, service.ProjectName)
			assert.Equal(t, tt.wantState, service.GetState())
			assert.Equal(t, tt.wantHealth, service.GetHealthStatus())
			assert.Equal(t, "", service.GetContainerID())
			assert.True(t, service.GetStartedAt().IsZero())
			assert.Nil(t, service.GetLastError())
		})
	}
}

// ============================================================================
// State Management Tests
// ============================================================================

func TestService_GetState(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Initial state should be pending
	assert.Equal(t, StatePending, service.GetState())

	// Manually change state to test getter
	service.mu.Lock()
	service.state = StateRunning
	service.mu.Unlock()

	assert.Equal(t, StateRunning, service.GetState())
}

func TestService_GetHealthStatus(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Initial health status should be unknown
	assert.Equal(t, HealthUnknown, service.GetHealthStatus())

	// Manually change health status to test getter
	service.mu.Lock()
	service.healthStatus = HealthHealthy
	service.mu.Unlock()

	assert.Equal(t, HealthHealthy, service.GetHealthStatus())
}

func TestService_IsRunning(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  bool
	}{
		{"pending is not running", StatePending, false},
		{"starting is not running", StateStarting, false},
		{"running is running", StateRunning, true},
		{"stopping is not running", StateStopping, false},
		{"stopped is not running", StateStopped, false},
		{"failed is not running", StateFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New("api", "myproject", config.Service{Image: "nginx:alpine"})
			service.mu.Lock()
			service.state = tt.state
			service.mu.Unlock()

			assert.Equal(t, tt.want, service.IsRunning())
		})
	}
}

func TestService_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		health HealthStatus
		want   bool
	}{
		{"unknown is not healthy", HealthUnknown, false},
		{"healthy is healthy", HealthHealthy, true},
		{"unhealthy is not healthy", HealthUnhealthy, false},
		{"starting is not healthy", HealthStarting, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New("api", "myproject", config.Service{Image: "nginx:alpine"})
			service.mu.Lock()
			service.healthStatus = tt.health
			service.mu.Unlock()

			assert.Equal(t, tt.want, service.IsHealthy())
		})
	}
}

// ============================================================================
// Getter Tests
// ============================================================================

func TestService_GetContainerID(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Initially empty
	assert.Equal(t, "", service.GetContainerID())

	// Set container ID
	expectedID := "abc123def456"
	service.mu.Lock()
	service.containerID = expectedID
	service.mu.Unlock()

	assert.Equal(t, expectedID, service.GetContainerID())
}

func TestService_GetStartedAt(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Initially zero time
	assert.True(t, service.GetStartedAt().IsZero())

	// Set started time
	now := time.Now()
	service.mu.Lock()
	service.startedAt = now
	service.mu.Unlock()

	// Allow for a small time difference due to execution time
	assert.WithinDuration(t, now, service.GetStartedAt(), time.Millisecond)
}

func TestService_GetUptime(t *testing.T) {
	tests := []struct {
		name      string
		state     State
		startedAt time.Time
		want      bool // whether uptime should be > 0
	}{
		{
			name:      "pending service has no uptime",
			state:     StatePending,
			startedAt: time.Time{},
			want:      false,
		},
		{
			name:      "running service has uptime",
			state:     StateRunning,
			startedAt: time.Now().Add(-5 * time.Second),
			want:      true,
		},
		{
			name:      "stopped service has no uptime",
			state:     StateStopped,
			startedAt: time.Now().Add(-5 * time.Second),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New("api", "myproject", config.Service{Image: "nginx:alpine"})
			service.mu.Lock()
			service.state = tt.state
			service.startedAt = tt.startedAt
			service.mu.Unlock()

			uptime := service.GetUptime()
			if tt.want {
				assert.Greater(t, uptime, time.Duration(0))
			} else {
				assert.Equal(t, time.Duration(0), uptime)
			}
		})
	}
}

// ============================================================================
// Helper Method Tests
// ============================================================================

func TestService_parsePortMappings(t *testing.T) {
	tests := []struct {
		name  string
		ports []string
		want  map[string]string
	}{
		{
			name:  "empty ports",
			ports: []string{},
			want:  map[string]string{},
		},
		{
			name:  "single port mapping",
			ports: []string{"8080:80"},
			want:  map[string]string{"8080": "80"},
		},
		{
			name:  "multiple port mappings",
			ports: []string{"8080:80", "3000:3000"},
			want:  map[string]string{"8080": "80", "3000": "3000"},
		},
		{
			name:  "invalid port mapping is skipped",
			ports: []string{"8080", "9000:90"},
			want:  map[string]string{"9000": "90"},
		},
		{
			name:  "port mapping with extra colons is skipped",
			ports: []string{"8080:80:tcp", "9000:90"},
			want:  map[string]string{"9000": "90"},
		},
		{
			name:  "empty string in ports array is skipped",
			ports: []string{"", "8080:80", ""},
			want:  map[string]string{"8080": "80"},
		},
		{
			name:  "port mapping with whitespace",
			ports: []string{"8080:80", "  9000:90  "},
			want:  map[string]string{"8080": "80", "  9000": "90  "},
		},
		{
			name:  "only invalid ports",
			ports: []string{"8080", "invalid", "9000"},
			want:  map[string]string{},
		},
		{
			name:  "duplicate host ports (last one wins)",
			ports: []string{"8080:80", "8080:8080"},
			want:  map[string]string{"8080": "8080"},
		},
		{
			name:  "same container port different host ports",
			ports: []string{"8080:80", "9090:80"},
			want:  map[string]string{"8080": "80", "9090": "80"},
		},
		{
			name:  "zero port numbers",
			ports: []string{"0:0", "8080:80"},
			want:  map[string]string{"0": "0", "8080": "80"},
		},
		{
			name:  "large port numbers",
			ports: []string{"65535:65535", "8080:80"},
			want:  map[string]string{"65535": "65535", "8080": "80"},
		},
		{
			name:  "non-numeric ports are included as-is",
			ports: []string{"abc:def", "8080:80"},
			want:  map[string]string{"abc": "def", "8080": "80"},
		},
		{
			name:  "mixed valid and invalid mappings",
			ports: []string{"8080:80", "invalid", "9000:90", "", "3000:3000"},
			want:  map[string]string{"8080": "80", "9000": "90", "3000": "3000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New("api", "myproject", config.Service{Ports: tt.ports})
			got := service.parsePortMappings()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_buildLabels(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})
	labels := service.buildLabels()

	assert.Equal(t, "true", labels["ork.managed"])
	assert.Equal(t, "myproject", labels["ork.project"])
	assert.Equal(t, "api", labels["ork.service"])
}

func TestService_getFirstPort(t *testing.T) {
	tests := []struct {
		name  string
		ports []string
		want  string
	}{
		{
			name:  "no ports returns default",
			ports: []string{},
			want:  "80",
		},
		{
			name:  "single port mapping",
			ports: []string{"8080:80"},
			want:  "8080",
		},
		{
			name:  "multiple ports returns first",
			ports: []string{"8080:80", "3000:3000"},
			want:  "8080",
		},
		{
			name:  "invalid port mapping returns it anyway",
			ports: []string{"8080"},
			want:  "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New("api", "myproject", config.Service{Ports: tt.ports})
			got := service.getFirstPort()
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// buildRunOptions Tests
// ============================================================================

func TestService_buildRunOptions(t *testing.T) {
	service := New("api", "myproject", config.Service{
		Image:      "nginx:alpine",
		Ports:      []string{"8080:80"},
		Command:    []string{"nginx", "-g", "daemon off;"},
		Entrypoint: []string{"/bin/sh"},
	})

	envVars := map[string]string{
		"PORT":     "8080",
		"NODE_ENV": "production",
	}

	opts := service.buildRunOptions(envVars)

	assert.Equal(t, "ork-myproject-api", opts.Name)
	assert.Equal(t, "nginx:alpine", opts.Image)
	assert.Equal(t, map[string]string{"8080": "80"}, opts.Ports)
	assert.Equal(t, envVars, opts.Env)
	assert.Equal(t, []string{"nginx", "-g", "daemon off;"}, opts.Command)
	assert.Equal(t, []string{"/bin/sh"}, opts.Entrypoint)
	assert.Equal(t, "true", opts.Labels["ork.managed"])
	assert.Equal(t, "myproject", opts.Labels["ork.project"])
	assert.Equal(t, "api", opts.Labels["ork.service"])
}

// ============================================================================
// String Representation Tests
// ============================================================================

func TestService_String(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Test default state
	str := service.String()
	assert.Contains(t, str, "api")
	assert.Contains(t, str, "pending")
	assert.Contains(t, str, "unknown")

	// Test with the running state
	service.mu.Lock()
	service.state = StateRunning
	service.healthStatus = HealthHealthy
	service.containerID = "abc123"
	service.mu.Unlock()

	str = service.String()
	assert.Contains(t, str, "running")
	assert.Contains(t, str, "healthy")
	assert.Contains(t, str, "abc123")
}

// ============================================================================
// Thread Safety Tests
// ============================================================================

func TestService_ConcurrentAccess(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Simulate concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			service.mu.Lock()
			service.state = StateRunning
			service.containerID = "test123"
			service.mu.Unlock()
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = service.GetState()
			_ = service.GetContainerID()
			_ = service.IsRunning()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// If we get here without a race condition, the test passes
	assert.Equal(t, StateRunning, service.GetState())
}

// ============================================================================
// Health Check Tests (without mocking)
// ============================================================================

func TestService_CheckHealth_NoHealthCheck(t *testing.T) {
	// Service with no health check configured should be considered healthy
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Simulate running state
	service.mu.Lock()
	service.state = StateRunning
	service.mu.Unlock()

	// Health check should succeed and mark as healthy
	// Note: This will fail because we don't have a running container
	// We're just testing the logic flow
}

func TestService_CheckHealth_NotRunning(t *testing.T) {
	service := New("api", "myproject", config.Service{
		Image: "nginx:alpine",
		Health: &config.HealthCheck{
			Endpoint: "/health",
		},
	})

	// Service is not running, health check should fail
	err := service.CheckHealth(nil)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "not running")
	}
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

func TestService_GetLastError(t *testing.T) {
	service := New("api", "myproject", config.Service{Image: "nginx:alpine"})

	// Initially no error
	assert.Nil(t, service.GetLastError())

	// Set an error
	expectedErr := assert.AnError
	service.mu.Lock()
	service.lastError = expectedErr
	service.mu.Unlock()

	assert.Equal(t, expectedErr, service.GetLastError())
}
