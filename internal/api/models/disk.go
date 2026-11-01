package models

import (
	"errors"
	"fmt"
)

// Disk représente la ressource Disk pour Terraform.
type Disk struct {
	ID           string      `json:"id" example:"disk-12345"`
	Name         string      `json:"name" example:"mon-disque"`
	AZ           string      `json:"az" example:"az1"`
	Size         int         `json:"size" example:"10"`
	StorageClass string      `json:"storage_class" example:"standard"`
	Source       *DiskSource `json:"source,omitempty"`
	Labels       []string    `json:"labels"`
}

// DiskSource représente la source d'un disque
type DiskSource struct {
	Type     string `json:"type" example:"blank"` // blank, http, snapshot, clone, registry
	URL      string `json:"url,omitempty" example:"http://example.com/image.qcow2"`
	Clone    string `json:"clone,omitempty" example:"disk-eid"`
	Snapshot string `json:"snapshot,omitempty" example:"snapshot-eid"`
}

// DiskRequest représente le corps de la requête pour créer ou mettre à jour un Disk.
type DiskRequest struct {
	Name         string      `json:"name" example:"mon-disque"`
	Size         int         `json:"size" example:"10"`
	StorageClass string      `json:"storage_class" example:"standard"`
	Source       *DiskSource `json:"source,omitempty"`
	// Labels au format "user.superphenix.net/clé:valeur"
	Labels []string `json:"labels"`
}

// Validate vérifie les contraintes sur le Disk.
func (d *DiskRequest) Validate() error {
	if d.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(d.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(d.Name))
	}
	if d.Size <= 0 {
		return errors.New("la taille (size) doit être supérieure à 0")
	}
	if err := ValidateCustomLabels(d.Labels); err != nil {
		return err
	}
	if d.StorageClass == "" {
		return errors.New("la storage_class est requise")
	}

	if d.Source == nil {
		return errors.New("la source est requise")
	}

	switch d.Source.Type {
	case "blank":
		// OK
	case "http", "registry":
		if d.Source.URL == "" {
			return fmt.Errorf("l'url est obligatoire pour le type %s", d.Source.Type)
		}
	case "clone":
		if d.Source.Clone == "" {
			return errors.New("le champ clone est obligatoire pour le type clone")
		}
	case "snapshot":
		if d.Source.Snapshot == "" {
			return errors.New("le champ snapshot est obligatoire pour le type snapshot")
		}
	default:
		return fmt.Errorf("type de source invalide: %s (valeurs possibles: blank, http, snapshot, clone, registry)", d.Source.Type)
	}

	return nil
}
