package models

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Instance represents an instance (VM) for Terraform.
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
	IPAddress    string            `json:"ip_address"`   // Primary IP (from VMI)
	Architecture string            `json:"architecture"` // CPU architecture
	// ContainerDisks lists the mounted container disks (catalog IDs)
	ContainerDisks []string          `json:"container_disks"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

// InstanceAdvanced represents the advanced options (TPM / EFI) of an instance.
// A nil field means "inherited from the vm_type" (no override).
type InstanceAdvanced struct {
	TPM InstanceTPM `json:"tpm"`
	EFI InstanceEFI `json:"efi"`
}

// InstanceTPM represents the TPM options
type InstanceTPM struct {
	Enabled    *bool `json:"enabled,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

// InstanceEFI represents the EFI options (enabled=false forces BIOS)
type InstanceEFI struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SecureBoot *bool `json:"secure_boot,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

// InstanceNetwork represents an instance network
type InstanceNetwork struct {
	Order      int    `json:"order"`
	SubnetID   string `json:"subnet_id"`
	Model      string `json:"model"`
	Enabled    *bool  `json:"enabled,omitempty"` // nil = interface enabled
	IPv4       string `json:"ipv4,omitempty"`
	IPv6       string `json:"ipv6,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}

// InstanceDisk represents an instance disk
type InstanceDisk struct {
	Order  int    `json:"order"`
	Bus    string `json:"bus"`
	DiskID string `json:"disk_id,omitempty"`
	Cdrom  bool   `json:"cdrom"`
}

// InstanceCloudInit represents the cloud-init configuration
type InstanceCloudInit struct {
	Custom bool   `json:"custom"`
	Bus    string `json:"bus"`
	Config string `json:"config,omitempty"`
}

// InstanceRequest represents the create/update request of an instance.
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
	// ContainerDisks: nil = recommended set (create) / existing mounts kept (update), [] = none
	ContainerDisks *[]string         `json:"container_disks,omitempty"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

// Validate checks the validity of the instance request.
func (r *InstanceRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(r.Name))
	}
	if r.CPU <= 0 {
		return errors.New("CPU count must be greater than 0")
	}
	if r.CPU > 32 {
		return errors.New("CPU count must not exceed 32")
	}
	if r.Memory <= 0 {
		return errors.New("memory must be greater than 0")
	}
	if r.Memory > 64 {
		return errors.New("memory must not exceed 64 GB")
	}
	if len(r.Network) == 0 {
		return errors.New("at least one network is required")
	}
	for _, nw := range r.Network {
		if nw.SubnetID == "" {
			return errors.New("the subnet ID is required for each network interface")
		}
		if nw.MACAddress != "" {
			if _, err := net.ParseMAC(nw.MACAddress); err != nil {
				return fmt.Errorf("invalid MAC address: %s", nw.MACAddress)
			}
		}
	}
	return ValidateCustomLabels(r.Labels)
}

// CustomLabelPrefix is the mandatory prefix of SPX user labels
const CustomLabelPrefix = "user.superphenix.net/"

// ValidateCustomLabels checks that each label starts with the SPX user prefix
func ValidateCustomLabels(labels []string) error {
	for _, label := range labels {
		if !strings.HasPrefix(label, CustomLabelPrefix) {
			return fmt.Errorf("label key '%s' must start with '%s'", label, CustomLabelPrefix)
		}
	}
	return nil
}
