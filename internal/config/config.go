package config

// Config represents the entire ork.yml file structure
type Config struct {
	Version  string             `yaml:"version"`  // e.g., "1.0"
	Project  string             `yaml:"project"`  // Project name
	Services map[string]Service `yaml:"services"` // Map of service name -> Service
}

// Service represents a single service definition
type Service struct {
	// Source configuration (mutually exclusive)
	Git   string `yaml:"git,omitempty"`   // Git repo URL (e.g., github.com/org/repo)
	Image string `yaml:"image,omitempty"` // Docker image (e.g., nginx:alpine)
	Build *Build `yaml:"build,omitempty"` // Build from a local source

	// Runtime configuration
	Ports      []string          `yaml:"ports,omitempty"`      // Port mappings (e.g., "3000:3000")
	Env        map[string]string `yaml:"env,omitempty"`        // Environment variables
	DependsOn  []string          `yaml:"depends_on,omitempty"` // Service dependencies
	Health     *HealthCheck      `yaml:"health,omitempty"`     // Health check config
	Command    []string          `yaml:"command,omitempty"`    // Override container command
	Entrypoint []string          `yaml:"entrypoint,omitempty"` // Override entrypoint
}

// Build represents build configuration for building from source
type Build struct {
	Context    string            `yaml:"context"`              // Build context path
	Dockerfile string            `yaml:"dockerfile,omitempty"` // Dockerfile path (default: Dockerfile)
	Args       map[string]string `yaml:"args,omitempty"`       // Build arguments
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Endpoint string `yaml:"endpoint"` // HTTP endpoint to check (e.g., /health)
	Interval string `yaml:"interval"` // Check interval (e.g., 5s)
	Timeout  string `yaml:"timeout"`  // Request timeout (e.g., 3s)
	Retries  int    `yaml:"retries"`  // Number of retries before unhealthy
}
