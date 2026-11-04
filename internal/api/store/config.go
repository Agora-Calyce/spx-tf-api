package store

import (
	"sync"
)

// ConfigStore handles the in-memory persistence of the global configuration.
type ConfigStore struct {
	mu       sync.RWMutex
	endpoint string
}

// NewConfigStore creates a new store for the configuration.
func NewConfigStore() *ConfigStore {
	return &ConfigStore{}
}

// SetEndpoint sets the global endpoint (must only be called at startup).
func (s *ConfigStore) SetEndpoint(endpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoint = endpoint
}

// GetEndpoint returns the global endpoint.
func (s *ConfigStore) GetEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endpoint
}
