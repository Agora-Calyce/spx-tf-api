package models

import (
	"errors"
	"fmt"
)

// Disk represents the Disk resource for Terraform.
type Disk struct {
	ID           string      `json:"id" example:"disk-12345"`
	Name         string      `json:"name" example:"mon-disque"`
	AZ           string      `json:"az" example:"az1"`
	Size         int         `json:"size" example:"10"`
	StorageClass string      `json:"storage_class" example:"standard"`
	Source       *DiskSource `json:"source,omitempty"`
	Labels       []string    `json:"labels"`
}

// DiskSource represents the source of a disk
type DiskSource struct {
	Type     string `json:"type" example:"blank"` // blank, http, snapshot, clone, registry
	URL      string `json:"url,omitempty" example:"http://example.com/image.qcow2"`
	Clone    string `json:"clone,omitempty" example:"disk-eid"`
	Snapshot string `json:"snapshot,omitempty" example:"snapshot-eid"`
}

// DiskRequest represents the request body to create or update a Disk.
type DiskRequest struct {
	Name         string      `json:"name" example:"mon-disque"`
	Size         int         `json:"size" example:"10"`
	StorageClass string      `json:"storage_class" example:"standard"`
	Source       *DiskSource `json:"source,omitempty"`
	// Labels in the "user.superphenix.net/key:value" format
	Labels []string `json:"labels"`
}

// Validate checks the constraints on the Disk.
func (d *DiskRequest) Validate() error {
	if d.Name == "" {
		return errors.New("name is required")
	}
	if len(d.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(d.Name))
	}
	if d.Size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if err := ValidateCustomLabels(d.Labels); err != nil {
		return err
	}
	if d.StorageClass == "" {
		return errors.New("storage_class is required")
	}

	if d.Source == nil {
		return errors.New("source is required")
	}

	switch d.Source.Type {
	case "blank":
		// OK
	case "http", "registry":
		if d.Source.URL == "" {
			return fmt.Errorf("url is required for type %s", d.Source.Type)
		}
	case "clone":
		if d.Source.Clone == "" {
			return errors.New("the clone field is required for type clone")
		}
	case "snapshot":
		if d.Source.Snapshot == "" {
			return errors.New("the snapshot field is required for type snapshot")
		}
	default:
		return fmt.Errorf("invalid source type: %s (possible values: blank, http, snapshot, clone, registry)", d.Source.Type)
	}

	return nil
}
