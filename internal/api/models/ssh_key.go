package models

import "fmt"

// SSHKey représente une clé SSH pour le fournisseur Terraform
type SSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
}

// SSHKeyRequest est la requête pour créer une clé SSH
type SSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

// Validate valide la requête de création de clé SSH
func (r *SSHKeyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("publicKey is required")
	}
	return nil
}
