package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StoreConfig defines a single store's configuration.
type StoreConfig struct {
	SchemaPath string `json:"schema"`        // Relative path to schema file
	Dir        string `json:"dir,omitempty"` // Directory for store files (default: .bipartite/)
}

// StoreRegistry is the configuration file format for stores.json.
type StoreRegistry struct {
	Stores map[string]*StoreConfig `json:"stores"`
}

// RegistryFilename is the name of the registry file within .bipartite/
const RegistryFilename = "stores.json"

// LoadRegistry loads the store registry from a repository root.
// If the registry file doesn't exist, returns an empty registry.
func LoadRegistry(repoRoot string) (*StoreRegistry, error) {
	path := filepath.Join(repoRoot, ".bipartite", RegistryFilename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &StoreRegistry{Stores: make(map[string]*StoreConfig)}, nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	var registry StoreRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	if registry.Stores == nil {
		registry.Stores = make(map[string]*StoreConfig)
	}

	return &registry, nil
}

// SaveRegistry saves the store registry to a repository root.
func SaveRegistry(repoRoot string, registry *StoreRegistry) error {
	dir := filepath.Join(repoRoot, ".bipartite")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating .bipartite directory: %w", err)
	}

	path := filepath.Join(dir, RegistryFilename)

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}

	return nil
}

// ListStores returns information about all registered stores.
func ListStores(repoRoot string) ([]StoreInfo, error) {
	registry, err := LoadRegistry(repoRoot)
	if err != nil {
		return nil, err
	}

	var stores []StoreInfo
	for name, config := range registry.Stores {
		// Load store to get full info
		s, err := OpenStore(repoRoot, name)
		if err != nil {
			// Store might be corrupted, include partial info
			stores = append(stores, StoreInfo{
				Name:       name,
				SchemaPath: config.SchemaPath,
				Error:      err.Error(),
			})
			continue
		}

		info, err := s.Info()
		if err != nil {
			stores = append(stores, StoreInfo{
				Name:       name,
				SchemaPath: config.SchemaPath,
				Error:      err.Error(),
			})
			continue
		}

		stores = append(stores, *info)
	}

	return stores, nil
}
