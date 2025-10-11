package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client with Ork-specific functionality
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client and verifies Docker is running
func NewClient() (*Client, error) {
	// Create Docker client (automatically detects DOCKER_HOST, etc.)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w\n💡 Is Docker installed? Try 'docker --version'", err)
	}

	// Verify Docker daemon is reachable
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w\n💡 Is Docker running? Try 'docker ps' or start Docker Desktop", err)
	}

	return &Client{cli: cli}, nil
}

// Close releases resources used by the Docker client
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}
