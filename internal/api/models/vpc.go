package models

import (
	"errors"
	"fmt"
)

// VPC représente la ressource VPC pour Terraform.
type VPC struct {
	ID           string        `json:"id" example:"vpc-12345"`
	Name         string        `json:"name" example:"mon-vpc-reseau"`
	AZ           string        `json:"az" example:"az1"`
	StaticRoutes []StaticRoute `json:"static_routes"`
}

// StaticRoute représente une route statique d'un VPC.
// Les routes générées automatiquement par les NAT Gateways ne sont pas exposées.
type StaticRoute struct {
	Policy     string `json:"policy" example:"policyDst"` // policyDst ou policySrc
	CIDR       string `json:"cidr" example:"192.168.0.0/24"`
	NextHopIP  string `json:"next_hop_ip" example:"10.0.0.254"`
	RouteTable string `json:"route_table" example:"subnet-eid"` // EID du subnet
}

// VPCRequest représente le corps de la requête pour créer ou mettre à jour un VPC.
type VPCRequest struct {
	Name string `json:"name" example:"mon-vpc-reseau"`
	// StaticRoutes (mise à jour uniquement) : nil = routes existantes conservées, [] = toutes supprimées
	StaticRoutes *[]StaticRoute `json:"static_routes,omitempty"`
}

// Validate vérifie que le nom du VPC ne dépasse pas 63 caractères.
func (v *VPCRequest) Validate() error {
	if v.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(v.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(v.Name))
	}
	if v.StaticRoutes != nil {
		for _, sr := range *v.StaticRoutes {
			if sr.CIDR == "" {
				return errors.New("le cidr est requis pour chaque route statique")
			}
			if sr.RouteTable == "" {
				return errors.New("la route_table (EID du subnet) est requise pour chaque route statique")
			}
			if sr.Policy != "" && sr.Policy != "policyDst" && sr.Policy != "policySrc" {
				return fmt.Errorf("policy invalide: %s (valeurs possibles: policyDst, policySrc)", sr.Policy)
			}
		}
	}
	return nil
}
