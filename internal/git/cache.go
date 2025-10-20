package git

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// cacheEntry represents a cached discovery result
type cacheEntry struct {
	Timestamp    time.Time    `json:"timestamp"`
	Repositories []Repository `json:"repositories"`
}

const (
	cacheFileName = "discovery-cache.json"
	cacheMaxAge   = 24 * time.Hour // Cache is valid for 24 hours
)

// getCachePath returns the path to the discovery cache file
func getCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	orkDir := filepath.Join(home, ".ork")
	return filepath.Join(orkDir, cacheFileName), nil
}

// LoadCache loads cached repositories if the cache is still valid
// Returns nil if the cache doesn't exist or is expired
func LoadCache() ([]Repository, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	// Check if the cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil // No cache
	}

	// Read the cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse cache
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Check if the cache is expired
	if time.Since(entry.Timestamp) > cacheMaxAge {
		return nil, nil // Expired cache
	}

	return entry.Repositories, nil
}

// SaveCache saves repositories to the cache file
func SaveCache(repos []Repository) error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	// Ensure .ork directory exists
	orkDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(orkDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ork directory: %w", err)
	}

	// Create a cache entry
	entry := cacheEntry{
		Timestamp:    time.Now(),
		Repositories: repos,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to the file
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// InvalidateCache removes the cache file
func InvalidateCache() error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	// Remove cache file (ignore error if it doesn't exist)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}
