package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// UserConfig holds user-customizable settings
type UserConfig struct {
	InterfaceLabels map[string]string `json:"interface_labels"` // Interface name -> Custom label
	mu              sync.RWMutex      `json:"-"`
}

// UserConfigManager manages user configuration persistence
type UserConfigManager struct {
	config   *UserConfig
	filePath string
	mu       sync.RWMutex
}

const (
	defaultDataDir      = "data"
	userConfigFileName  = "config.json"
)

// NewUserConfigManager creates a new user configuration manager
func NewUserConfigManager() (*UserConfigManager, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(defaultDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	configPath := filepath.Join(defaultDataDir, userConfigFileName)

	manager := &UserConfigManager{
		filePath: configPath,
		config: &UserConfig{
			InterfaceLabels: make(map[string]string),
		},
	}

	// Load existing config if present
	if err := manager.Load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[UserConfig] Warning: Failed to load config: %v", err)
		}
		// Save default config
		if err := manager.Save(); err != nil {
			log.Printf("[UserConfig] Warning: Failed to save default config: %v", err)
		}
	}

	log.Printf("[UserConfig] Loaded user configuration from: %s", configPath)
	return manager, nil
}

// Load reads configuration from disk
func (m *UserConfigManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.config)
}

// Save writes configuration to disk
func (m *UserConfigManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// GetInterfaceLabel returns custom label for an interface
func (m *UserConfigManager) GetInterfaceLabel(interfaceName string) string {
	m.config.mu.RLock()
	defer m.config.mu.RUnlock()

	if label, ok := m.config.InterfaceLabels[interfaceName]; ok && label != "" {
		return label
	}
	return interfaceName // Return original name if no label set
}

// SetInterfaceLabel sets custom label for an interface
func (m *UserConfigManager) SetInterfaceLabel(interfaceName, label string) error {
	m.config.mu.Lock()
	m.config.InterfaceLabels[interfaceName] = label
	m.config.mu.Unlock()

	return m.Save()
}

// GetAllInterfaceLabels returns all interface labels
func (m *UserConfigManager) GetAllInterfaceLabels() map[string]string {
	m.config.mu.RLock()
	defer m.config.mu.RUnlock()

	// Return a copy to avoid race conditions
	labels := make(map[string]string, len(m.config.InterfaceLabels))
	for k, v := range m.config.InterfaceLabels {
		labels[k] = v
	}
	return labels
}

// UpdateInterfaceLabels updates multiple interface labels at once
func (m *UserConfigManager) UpdateInterfaceLabels(labels map[string]string) error {
	m.config.mu.Lock()
	for interfaceName, label := range labels {
		m.config.InterfaceLabels[interfaceName] = label
	}
	m.config.mu.Unlock()

	return m.Save()
}
