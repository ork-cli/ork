package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hary-singh/ork/internal/config"
	"github.com/hary-singh/ork/internal/docker"
)

// ============================================================================
// Type Definitions - Service State
// ============================================================================

// State represents the current state of a service
type State string

const (
	StatePending  State = "pending"  // Service not yet started
	StateStarting State = "starting" // Service is being started
	StateRunning  State = "running"  // Service is running
	StateStopping State = "stopping" // Service is being stopped
	StateStopped  State = "stopped"  // Service has been stopped
	StateFailed   State = "failed"   // Service failed to start or crashed
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"   // Health status not yet determined
	HealthHealthy   HealthStatus = "healthy"   // Service is healthy
	HealthUnhealthy HealthStatus = "unhealthy" // Service is unhealthy
	HealthStarting  HealthStatus = "starting"  // Service is starting (health check has not run yet)
)

// ============================================================================
// Service Structure
// ============================================================================

// Service represents a runtime service instance with state tracking
type Service struct {
	// Service identification
	Name        string         // Service name (e.g., "frontend", "api")
	ProjectName string         // Project this service belongs to
	Config      config.Service // Service configuration from ork.yml

	// Runtime state
	state        State        // Current service state
	healthStatus HealthStatus // Current health status
	containerID  string       // Docker container ID (when running)
	networkID    string       // Network ID the service is connected to
	startedAt    time.Time    // When the service was started
	stoppedAt    time.Time    // When the service was stopped
	lastError    error        // Last error encountered

	// Synchronization
	mu sync.RWMutex // Protects state changes
}

// ============================================================================
// Constructor
// ============================================================================

// New creates a new Service instance
func New(name string, projectName string, cfg config.Service) *Service {
	return &Service{
		Name:         name,
		ProjectName:  projectName,
		Config:       cfg,
		state:        StatePending,
		healthStatus: HealthUnknown,
	}
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Start starts the service container
func (s *Service) Start(ctx context.Context, client *docker.Client, networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already running
	if s.state == StateRunning {
		return fmt.Errorf("service %s is already running", s.Name)
	}

	// Update state to starting
	s.state = StateStarting
	s.healthStatus = HealthStarting
	s.lastError = nil

	// Check if a container already exists
	if err := s.checkAndCleanupExistingContainer(ctx, client); err != nil {
		s.state = StateFailed
		s.lastError = err
		return err
	}

	// Load environment variables
	envVars, err := config.LoadAllEnvForService(s.Name, s.Config.Env)
	if err != nil {
		s.state = StateFailed
		s.lastError = fmt.Errorf("failed to load environment variables: %w", err)
		return s.lastError
	}

	// Build run options
	runOpts := s.buildRunOptions(envVars)

	// Start the container
	containerID, err := client.Run(ctx, runOpts)
	if err != nil {
		s.state = StateFailed
		s.lastError = fmt.Errorf("failed to start container: %w", err)
		return s.lastError
	}

	// Connect to network if provided
	if networkID != "" {
		if err := client.ConnectContainer(ctx, s.ProjectName, containerID); err != nil {
			// Non-fatal - log but continue
			fmt.Printf("⚠️  Warning: failed to connect %s to network: %v\n", s.Name, err)
		}
	}

	// Update state
	s.containerID = containerID
	s.networkID = networkID
	s.startedAt = time.Now()
	s.state = StateRunning
	s.healthStatus = HealthUnknown // Will be checked later

	return nil
}

// Stop stops the service container
func (s *Service) Stop(ctx context.Context, client *docker.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already stopped
	if s.state == StateStopped || s.state == StatePending {
		return fmt.Errorf("service %s is not running", s.Name)
	}

	// Check if we have a container ID
	if s.containerID == "" {
		return fmt.Errorf("service %s has no container ID", s.Name)
	}

	// Update state to stopping
	s.state = StateStopping

	// Stop and remove the container
	if err := client.StopAndRemove(ctx, s.containerID); err != nil {
		s.state = StateFailed
		s.lastError = fmt.Errorf("failed to stop container: %w", err)
		return s.lastError
	}

	// Update state
	s.state = StateStopped
	s.healthStatus = HealthUnknown
	s.stoppedAt = time.Now()
	s.containerID = ""

	return nil
}

// Restart restarts the service by stopping and then starting it
func (s *Service) Restart(ctx context.Context, client *docker.Client, networkID string) error {
	// Stop the service (if running)
	if s.GetState() == StateRunning {
		if err := s.Stop(ctx, client); err != nil {
			return fmt.Errorf("failed to stop service during restart: %w", err)
		}
	}

	// Start the service
	if err := s.Start(ctx, client, networkID); err != nil {
		return fmt.Errorf("failed to start service during restart: %w", err)
	}

	return nil
}

// ============================================================================
// State Getters
// ============================================================================

// GetState returns the current service state
func (s *Service) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetHealthStatus returns the current health status
func (s *Service) GetHealthStatus() HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthStatus
}

// GetContainerID returns the container ID
func (s *Service) GetContainerID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.containerID
}

// GetStartedAt returns when the service was started
func (s *Service) GetStartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

// GetLastError returns the last error encountered
func (s *Service) GetLastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// GetUptime returns how long the service has been running
func (s *Service) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != StateRunning || s.startedAt.IsZero() {
		return 0
	}

	return time.Since(s.startedAt)
}

// IsRunning returns true if the service is currently running
func (s *Service) IsRunning() bool {
	return s.GetState() == StateRunning
}

// IsHealthy returns true if the service is healthy
func (s *Service) IsHealthy() bool {
	return s.GetHealthStatus() == HealthHealthy
}

// ============================================================================
// Health Check Methods
// ============================================================================

// CheckHealth performs a health check on the service
func (s *Service) CheckHealth(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Can only check health if service is running
	if s.state != StateRunning {
		return fmt.Errorf("service %s is not running", s.Name)
	}

	// If no health check is configured, assume healthy
	if s.Config.Health == nil {
		s.healthStatus = HealthHealthy
		return nil
	}

	// Perform HTTP health check
	if s.Config.Health.Endpoint != "" {
		if err := s.performHTTPHealthCheck(ctx); err != nil {
			s.healthStatus = HealthUnhealthy
			return err
		}
		s.healthStatus = HealthHealthy
		return nil
	}

	// No health check configured
	s.healthStatus = HealthHealthy
	return nil
}

// performHTTPHealthCheck performs an HTTP health check
func (s *Service) performHTTPHealthCheck(ctx context.Context) error {
	// Parse timeout (default to 3 seconds)
	timeout := 3 * time.Second
	if s.Config.Health.Timeout != "" {
		if d, err := time.ParseDuration(s.Config.Health.Timeout); err == nil {
			timeout = d
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Build health check URL
	// Use localhost since we're checking from the host
	url := fmt.Sprintf("http://localhost:%s%s", s.getFirstPort(), s.Config.Health.Endpoint)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Perform health check with retries
	retries := s.Config.Health.Retries
	if retries == 0 {
		retries = 3 // Default to 3 retries
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("health check returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		// Wait before retry (except on the last attempt)
		if i < retries-1 {
			time.Sleep(time.Second)
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", retries, lastErr)
}

// getFirstPort extracts the first host port from the service configuration
func (s *Service) getFirstPort() string {
	if len(s.Config.Ports) == 0 {
		return "80" // Default port
	}

	// Parse port mapping like "8080:80"
	parts := strings.Split(s.Config.Ports[0], ":")
	if len(parts) >= 1 {
		return parts[0]
	}

	return "80"
}

// ============================================================================
// Private Helpers
// ============================================================================

// checkAndCleanupExistingContainer checks if a container for this service already exists
func (s *Service) checkAndCleanupExistingContainer(ctx context.Context, client *docker.Client) error {
	containers, err := client.List(ctx, s.ProjectName)
	if err != nil {
		return fmt.Errorf("failed to check existing containers: %w", err)
	}

	for _, container := range containers {
		if container.Labels["ork.service"] == s.Name {
			// Check if it's running
			if strings.HasPrefix(container.Status, "Up") {
				// Update our state to match reality
				s.containerID = container.ID
				s.state = StateRunning
				return fmt.Errorf("service %s is already running (container: %s)", s.Name, container.ID)
			}

			// Container exists but is stopped - remove it
			if err := client.Remove(ctx, container.ID); err != nil {
				return fmt.Errorf("failed to remove stopped container: %w", err)
			}
		}
	}

	return nil
}

// buildRunOptions constructs Docker run options from the service configuration
func (s *Service) buildRunOptions(envVars map[string]string) docker.RunOptions {
	return docker.RunOptions{
		Name:       fmt.Sprintf("ork-%s-%s", s.ProjectName, s.Name),
		Image:      s.Config.Image,
		Ports:      s.parsePortMappings(),
		Env:        envVars,
		Labels:     s.buildLabels(),
		Command:    s.Config.Command,
		Entrypoint: s.Config.Entrypoint,
	}
}

// parsePortMappings converts port strings like "8080:80" to map["8080"]="80"
func (s *Service) parsePortMappings() map[string]string {
	ports := make(map[string]string)

	for _, mapping := range s.Config.Ports {
		// Split "8080:80" into ["8080", "80"]
		parts := strings.Split(mapping, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPort := parts[1]
			ports[hostPort] = containerPort
		}
	}

	return ports
}

// buildLabels creates standard Ork labels for container tracking
func (s *Service) buildLabels() map[string]string {
	return map[string]string{
		"ork.managed": "true",
		"ork.project": s.ProjectName,
		"ork.service": s.Name,
	}
}

// ============================================================================
// String Representation
// ============================================================================

// String returns a string representation of the service
func (s *Service) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("Service{name=%s, state=%s, health=%s, containerID=%s}",
		s.Name, s.state, s.healthStatus, s.containerID)
}
