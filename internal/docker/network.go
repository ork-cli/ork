package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
)

// ============================================================================
// Type Definitions
// ============================================================================

// NetworkInfo represents information about a Docker network
type NetworkInfo struct {
	ID   string // Network ID
	Name string // Network name
}

// ============================================================================
// Public Methods - Network Lifecycle
// ============================================================================

// CreateNetwork creates a Docker network for the project
// All containers in the same project will be connected to this network
// This allows services to communicate using service names (e.g., postgres:5432)
func (c *Client) CreateNetwork(ctx context.Context, projectName string) (string, error) {
	networkName := buildNetworkName(projectName)

	// Check if the network already exists
	existingNetworks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range existingNetworks {
		if net.Name == networkName {
			// Network already exists, return its ID
			return net.ID, nil
		}
	}

	// Create the network
	opts := network.CreateOptions{
		Driver: "bridge", // Use bridge driver for local networking
		Labels: buildNetworkLabels(projectName),
	}

	response, err := c.cli.NetworkCreate(ctx, networkName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create network %s: %w\n💡 Check if Docker daemon is running", networkName, err)
	}

	return response.ID, nil
}

// DeleteNetwork removes a Docker network
func (c *Client) DeleteNetwork(ctx context.Context, projectName string) error {
	networkName := buildNetworkName(projectName)

	// Get network ID
	networkID, err := c.findNetworkByName(ctx, networkName)
	if err != nil {
		// Network doesn't exist, nothing to delete
		return nil
	}

	// Remove the network
	if err := c.cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("failed to remove network %s: %w", networkName, err)
	}

	return nil
}

// ConnectContainer connects a container to the project network
// This must be called after the container is created but can be before or after it's started
func (c *Client) ConnectContainer(ctx context.Context, projectName, containerID string) error {
	networkName := buildNetworkName(projectName)

	// Get network ID
	networkID, err := c.findNetworkByName(ctx, networkName)
	if err != nil {
		return fmt.Errorf("project network not found: %w\n💡 Network should be created before starting containers", err)
	}

	// Connect container to network
	err = c.cli.NetworkConnect(ctx, networkID, containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to connect container %s to network: %w", containerID[:12], err)
	}

	return nil
}

// ============================================================================
// Private Helpers - Network Discovery
// ============================================================================

// findNetworkByName finds a network by name and returns its ID
func (c *Client) findNetworkByName(ctx context.Context, networkName string) (string, error) {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return net.ID, nil
		}
	}

	return "", fmt.Errorf("network '%s' not found", networkName)
}

// ============================================================================
// Private Helpers - Naming and Labels
// ============================================================================

// buildNetworkName creates a consistent network name for a project
func buildNetworkName(projectName string) string {
	return fmt.Sprintf("ork-%s-network", projectName)
}

// buildNetworkLabels creates standard Ork labels for network tracking
func buildNetworkLabels(projectName string) map[string]string {
	return map[string]string{
		"ork.managed": "true",
		"ork.project": projectName,
	}
}
