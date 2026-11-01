package store

import (
	"sync"
)

// ConfigStore gère la persistance en mémoire de la configuration globale.
type ConfigStore struct {
	mu       sync.RWMutex
	endpoint string
}

// NewConfigStore crée un nouveau store pour la configuration.
func NewConfigStore() *ConfigStore {
	return &ConfigStore{}
}

// SetEndpoint définit l'endpoint global (ne doit être appelé qu'au démarrage).
func (s *ConfigStore) SetEndpoint(endpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoint = endpoint
}

// GetEndpoint retourne l'endpoint global.
func (s *ConfigStore) GetEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endpoint
}
