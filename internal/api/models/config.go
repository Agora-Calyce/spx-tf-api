package models

import "fmt"

// Configuration represents the global settings (Endpoint).
type Configuration struct {
	Endpoint string `json:"endpoint"`
}

// Validate checks the validity of the global configuration.
func (c *Configuration) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	return nil
}
