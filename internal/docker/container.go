package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
)

// ============================================================================
// Type Definitions
// ============================================================================

// RunOptions contains configuration for running a container
type RunOptions struct {
	Name       string            // Container name
	Image      string            // Docker image (e.g., "nginx:alpine")
	Ports      map[string]string // Port mappings (e.g., "8080": "80")
	Env        map[string]string // Environment variables
	Labels     map[string]string // Container labels
	Command    []string          // Override command
	Entrypoint []string          // Override entrypoint
}

// ContainerInfo represents information about a running container
type ContainerInfo struct {
	ID     string            // Container ID (short version)
	Name   string            // Container name
	Image  string            // Image name
	Status string            // Container status (e.g., "Up 5 minutes")
	Ports  []string          // Port mappings
	Labels map[string]string // Container labels
}

// LogsOptions contains configuration for retrieving container logs
type LogsOptions struct {
	Follow     bool   // Stream logs continuously (like tail -f)
	Tail       string // Number of lines to show from the end ("all" or "100")
	Timestamps bool   // Show timestamps in log output
}

// ============================================================================
// Public Methods - Container Lifecycle
// ============================================================================

// Run creates and starts a Docker container
// This orchestrates the full container lifecycle but delegates to specialized functions
func (c *Client) Run(ctx context.Context, opts RunOptions) (containerID string, err error) {
	// Ensure the image is available locally
	if err := c.pullImageIfNeeded(ctx, opts.Image); err != nil {
		return "", err
	}

	// Build container configuration
	config, err := buildContainerConfig(opts)
	if err != nil {
		return "", err
	}

	// Build host configuration
	hostConfig := buildHostConfig(opts)

	// Create and start the container
	containerID, err = c.createAndStartContainer(ctx, config, hostConfig, opts.Name)
	if err != nil {
		return "", err
	}

	return containerID, nil
}

// Stop stops a running Docker container
func (c *Client) Stop(ctx context.Context, containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	// Stop the container (with a 10-second timeout for graceful shutdown)
	timeout := 10
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := c.cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	return nil
}

// Remove removes a Docker container (must be stopped first)
func (c *Client) Remove(ctx context.Context, containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	removeOptions := container.RemoveOptions{
		Force: false, // Don't force-remove running containers
	}

	if err := c.cli.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// StopAndRemove stops and removes a Docker container
func (c *Client) StopAndRemove(ctx context.Context, containerID string) error {
	// Stop first
	if err := c.Stop(ctx, containerID); err != nil {
		return err
	}

	// Then remove
	if err := c.Remove(ctx, containerID); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// Public Methods - Container Information
// ============================================================================

// List returns a list of containers managed by Ork
func (c *Client) List(ctx context.Context, projectName string) ([]ContainerInfo, error) {
	// Build filters to only show Ork-managed containers
	filterArgs := buildOrkFilters(projectName)

	// List containers
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true, // Include stopped containers
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Convert to our ContainerInfo format
	return convertToContainerInfo(containers), nil
}

// ============================================================================
// Public Methods - Container Logs
// ============================================================================

// Logs retrieves and streams container logs to stdout
// This is useful for debugging and monitoring container output
func (c *Client) Logs(ctx context.Context, containerID string, opts LogsOptions) error {
	// Validate input
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	// Build Docker API log options
	logOptions := container.LogsOptions{
		ShowStdout: true,            // Include stdout
		ShowStderr: true,            // Include stderr
		Follow:     opts.Follow,     // Stream continuously if requested
		Timestamps: opts.Timestamps, // Show timestamps if requested
		Tail:       opts.Tail,       // Limit output if specified
	}

	// Get logs reader from Docker
	reader, err := c.cli.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		return fmt.Errorf("failed to get logs for container %s: %w\n💡 Check if container exists with 'ork ps'", containerID, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			fmt.Printf("⚠️  Warning: failed to close logs reader: %v\n", closeErr)
		}
	}()

	// Stream logs to stdout
	// Docker multiplexes stdout/stderr into the reader, so we just copy it all
	_, err = io.Copy(os.Stdout, reader)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to stream logs: %w", err)
	}

	return nil
}

// ============================================================================
// Private Helpers - Run-related
// ============================================================================

// buildContainerConfig creates the container configuration from options
func buildContainerConfig(opts RunOptions) (*container.Config, error) {
	config := &container.Config{
		Image:  opts.Image,
		Env:    convertEnvMapToSlice(opts.Env),
		Labels: opts.Labels,
	}

	// Override command/entrypoint if specified
	if len(opts.Command) > 0 {
		config.Cmd = opts.Command
	}
	if len(opts.Entrypoint) > 0 {
		config.Entrypoint = opts.Entrypoint
	}

	// Add exposed ports
	if len(opts.Ports) > 0 {
		exposedPorts, err := createExposedPorts(opts.Ports)
		if err != nil {
			return nil, err
		}
		config.ExposedPorts = exposedPorts
	}

	return config, nil
}

// buildHostConfig creates the host configuration from options
func buildHostConfig(opts RunOptions) *container.HostConfig {
	return &container.HostConfig{
		PortBindings: convertPortsToBindings(opts.Ports),
		AutoRemove:   false, // Keep containers for debugging
	}
}

// createExposedPorts converts a port map to Docker's exposed ports format
func createExposedPorts(ports map[string]string) (nat.PortSet, error) {
	exposedPorts := make(nat.PortSet)

	for _, containerPort := range ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return nil, fmt.Errorf("invalid port %s: %w", containerPort, err)
		}
		exposedPorts[port] = struct{}{}
	}

	return exposedPorts, nil
}

// createAndStartContainer creates and starts a Docker container
func (c *Client) createAndStartContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
	// Create the container
	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w\n💡 Check if port is already in use", err)
	}

	// Start the container
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container %s: %w", resp.ID, err)
	}

	return resp.ID, nil
}

// pullImageIfNeeded pulls an image if it doesn't exist locally
func (c *Client) pullImageIfNeeded(ctx context.Context, imageName string) error {
	// Check if the image exists locally
	_, err := c.cli.ImageInspect(ctx, imageName)
	if err == nil {
		// Image exists locally, no need to pull
		return nil
	}

	// Image doesn't exist, pull it
	fmt.Printf("📥 Pulling image %s...\n", imageName)

	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w\n💡 Check image name and Docker Hub access", imageName, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			fmt.Printf("⚠️  Warning: failed to close image pull reader: %v\n", closeErr)
		}
	}()

	// Stream pull output (this shows download progress)
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	fmt.Printf("✅ Image %s pulled successfully\n", imageName)
	return nil
}

// ============================================================================
// Private Helpers - List-related
// ============================================================================

// buildOrkFilters creates filters to find Ork-managed containers
func buildOrkFilters(projectName string) filters.Args {
	filterArgs := filters.NewArgs()

	// Filter by Ork label
	filterArgs.Add("label", "ork.managed=true")

	// If a project name is specified, filter by project
	if projectName != "" {
		filterArgs.Add("label", fmt.Sprintf("ork.project=%s", projectName))
	}

	return filterArgs
}

// convertToContainerInfo converts Docker API containers to our format
func convertToContainerInfo(containers []container.Summary) []ContainerInfo {
	result := make([]ContainerInfo, 0, len(containers))

	for _, c := range containers {
		info := ContainerInfo{
			ID:     c.ID[:12], // Use short ID (first 12 chars)
			Image:  c.Image,
			Status: c.Status,
			Labels: c.Labels,
		}

		// Extract container name (remove leading slash)
		if len(c.Names) > 0 {
			info.Name = c.Names[0]
			if len(info.Name) > 0 && info.Name[0] == '/' {
				info.Name = info.Name[1:] // Remove the leading slash
			}
		}

		// Format port mappings
		info.Ports = formatPorts(c.Ports)

		result = append(result, info)
	}

	return result
}

// formatPorts converts Docker port bindings to human-readable strings
func formatPorts(ports []container.Port) []string {
	if len(ports) == 0 {
		return nil
	}

	result := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort > 0 {
			portStr := fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			result = append(result, portStr)
		}
	}

	return result
}

// ============================================================================
// Utility Converters
// ============================================================================

// convertEnvMapToSlice converts an environment map to Docker's env slice format
// Docker expects: ["KEY=VALUE", "KEY2=VALUE2"]
func convertEnvMapToSlice(envMap map[string]string) []string {
	if envMap == nil {
		return nil
	}

	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

// convertPortsToBindings converts a port map to Docker port bindings
// Input: {"8080": "80"} means host:8080 -> container:80
func convertPortsToBindings(ports map[string]string) nat.PortMap {
	if ports == nil {
		return nil
	}

	bindings := make(nat.PortMap)
	for hostPort, containerPort := range ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			continue // Skip invalid ports
		}

		bindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}
	return bindings
}
