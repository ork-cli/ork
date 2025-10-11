package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
)

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
