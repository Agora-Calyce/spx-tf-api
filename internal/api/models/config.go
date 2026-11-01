package models

import "fmt"

// Configuration représente les paramètres globaux (Endpoint).
type Configuration struct {
	Endpoint string `json:"endpoint"`
}

// Validate vérifie la validité de la configuration globale.
func (c *Configuration) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint est obligatoire")
	}
	return nil
}
