package models

import "fmt"

// SSHKey represents an SSH key for the Terraform provider
type SSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
}

// SSHKeyRequest is the request to create an SSH key
type SSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

// Validate validates the SSH key creation request
func (r *SSHKeyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("publicKey is required")
	}
	return nil
}
