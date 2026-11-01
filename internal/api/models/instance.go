package models

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Instance représente une instance (VM) pour Terraform.
type Instance struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	RunStrategy  string            `json:"run_strategy"`
	VMType       string            `json:"vm_type"`
	CPU          int               `json:"cpu"`
	Memory       int               `json:"memory"`
	Network      []InstanceNetwork `json:"network"`
	Disks        []InstanceDisk    `json:"disks"`
	CloudInit    InstanceCloudInit `json:"cloud_init"`
	SSHKeys      []string          `json:"ssh_keys"`
	Labels       []string          `json:"labels"`
	Ready        bool              `json:"ready"`
	IPAddress    string            `json:"ip_address"`   // IP principale (depuis VMI)
	Architecture string            `json:"architecture"` // Architecture CPU
	// ContainerDisks liste les container disks montés (IDs du catalogue)
	ContainerDisks []string          `json:"container_disks"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

// InstanceAdvanced représente les options avancées (TPM / EFI) d'une instance.
// Un champ nil signifie "hérité du vm_type" (aucune surcharge).
type InstanceAdvanced struct {
	TPM InstanceTPM `json:"tpm"`
	EFI InstanceEFI `json:"efi"`
}

// InstanceTPM représente les options TPM
type InstanceTPM struct {
	Enabled    *bool `json:"enabled,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

// InstanceEFI représente les options EFI (enabled=false force le BIOS)
type InstanceEFI struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SecureBoot *bool `json:"secure_boot,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

// InstanceNetwork représente un réseau d'instance
type InstanceNetwork struct {
	Order      int    `json:"order"`
	SubnetID   string `json:"subnet_id"`
	Model      string `json:"model"`
	Enabled    *bool  `json:"enabled,omitempty"` // nil = interface active
	IPv4       string `json:"ipv4,omitempty"`
	IPv6       string `json:"ipv6,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}

// InstanceDisk représente un disque d'instance
type InstanceDisk struct {
	Order  int    `json:"order"`
	Bus    string `json:"bus"`
	DiskID string `json:"disk_id,omitempty"`
	Cdrom  bool   `json:"cdrom"`
}

// InstanceCloudInit représente la configuration cloud-init
type InstanceCloudInit struct {
	Custom bool   `json:"custom"`
	Bus    string `json:"bus"`
	Config string `json:"config,omitempty"`
}

// InstanceRequest représente la requête de création/mise à jour d'une instance.
type InstanceRequest struct {
	Name        string            `json:"name"`
	RunStrategy string            `json:"run_strategy"`
	VMType      string            `json:"vm_type"`
	CPU         int               `json:"cpu"`
	Memory      int               `json:"memory"`
	Network     []InstanceNetwork `json:"network"`
	Disks       []InstanceDisk    `json:"disks"`
	CloudInit   InstanceCloudInit `json:"cloud_init"`
	SSHKeys     []string          `json:"ssh_keys"`
	Labels      []string          `json:"labels"`
	// ContainerDisks : nil = jeu recommandé (création) / montages conservés (mise à jour), [] = aucun
	ContainerDisks *[]string         `json:"container_disks,omitempty"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

// Validate vérifie la validité de la requête d'instance.
func (r *InstanceRequest) Validate() error {
	if r.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(r.Name))
	}
	if r.CPU <= 0 {
		return errors.New("le nombre de CPU doit être supérieur à 0")
	}
	if r.CPU > 32 {
		return errors.New("le nombre de CPU ne doit pas dépasser 32")
	}
	if r.Memory <= 0 {
		return errors.New("la mémoire doit être supérieure à 0")
	}
	if r.Memory > 64 {
		return errors.New("la mémoire ne doit pas dépasser 64 GB")
	}
	if len(r.Network) == 0 {
		return errors.New("au moins un réseau est requis")
	}
	for _, nw := range r.Network {
		if nw.SubnetID == "" {
			return errors.New("l'ID du subnet est requis pour chaque interface réseau")
		}
		if nw.MACAddress != "" {
			if _, err := net.ParseMAC(nw.MACAddress); err != nil {
				return fmt.Errorf("adresse MAC invalide: %s", nw.MACAddress)
			}
		}
	}
	return ValidateCustomLabels(r.Labels)
}

// CustomLabelPrefix est le préfixe obligatoire des labels utilisateur SPX
const CustomLabelPrefix = "user.superphenix.net/"

// ValidateCustomLabels vérifie que chaque label commence par le préfixe utilisateur SPX
func ValidateCustomLabels(labels []string) error {
	for _, label := range labels {
		if !strings.HasPrefix(label, CustomLabelPrefix) {
			return fmt.Errorf("la clé du label '%s' doit commencer par '%s'", label, CustomLabelPrefix)
		}
	}
	return nil
}
