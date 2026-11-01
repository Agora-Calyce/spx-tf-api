package models

import (
	"errors"
	"fmt"
)

const (
	ProtocolIPv4 = "IPv4"
	ProtocolIPv6 = "IPv6"
	ProtocolDual = "Dual"
)

// Subnet représente la ressource Subnet pour Terraform.
type Subnet struct {
	ID           string `json:"id" example:"subnet-12345"`
	Name         string `json:"name" example:"mon-subnet"`
	AZ           string `json:"az" example:"az1"`
	VPCID        string `json:"vpc_id" example:"vpc-12345"`
	IPv4         string `json:"ipv4" example:"10.0.1.0/24"`
	IPv6         string `json:"ipv6,omitempty" example:"2001:db8::/64"`
	Private      bool   `json:"private" example:"true"`
	Protocol     string `json:"protocol" example:"IPv4"`
	DNSV4        string `json:"dns_v4,omitempty" example:"8.8.8.8"`
	DNSV6        string `json:"dns_v6,omitempty" example:"2001:4860:4860::8888"`
	NatGateway   bool   `json:"nat_gateway" example:"false"`
	NatGatewayIp string `json:"nat_gateway_ip,omitempty" example:"10.0.1.1"`
}

// SubnetRequest représente le corps de la requête pour créer ou mettre à jour un Subnet.
type SubnetRequest struct {
	Name       string `json:"name" example:"mon-subnet"`
	VPCID      string `json:"vpc_id" example:"vpc-12345"`
	IPv4       string `json:"ipv4" example:"10.0.1.0/24"`
	IPv6       string `json:"ipv6,omitempty" example:"2001:db8::/64"`
	Private    bool   `json:"private" example:"true"`
	Protocol   string `json:"protocol" example:"IPv4"`
	DNSV4      string `json:"dns_v4,omitempty" example:"8.8.8.8"`
	DNSV6      string `json:"dns_v6,omitempty" example:"2001:4860:4860::8888"`
	NatGateway bool   `json:"nat_gateway" example:"false"`
}

// Validate vérifie les contraintes sur le Subnet.
func (s *SubnetRequest) Validate() error {
	if s.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(s.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(s.Name))
	}
	if s.VPCID == "" {
		return errors.New("le vpc_id est requis")
	}

	switch s.Protocol {
	case ProtocolIPv4:
		if s.IPv4 == "" {
			return errors.New("le champ ipv4 est obligatoire pour le protocole IPv4")
		}
	case ProtocolIPv6:
		if s.IPv6 == "" {
			return errors.New("le champ ipv6 est obligatoire pour le protocole IPv6")
		}
	case ProtocolDual:
		if s.IPv4 == "" || s.IPv6 == "" {
			return errors.New("les deux champs ipv4 et ipv6 sont obligatoires pour le protocole Dual")
		}
	case "":
		return errors.New("le protocole est obligatoire (IPv4, IPv6 ou Dual)")
	default:
		return fmt.Errorf("protocole invalide: %s (doit être IPv4, IPv6 ou Dual)", s.Protocol)
	}

	return nil
}
