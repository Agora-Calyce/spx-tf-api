package models

import (
	"errors"
	"fmt"
	"net"
)

// Eip représente la ressource EIP pour Terraform.
type Eip struct {
	ID         string    `json:"id" example:"eip-12345"`
	Name       string    `json:"name" example:"mon-eip"`
	AZ         string    `json:"az" example:"az1"`
	SubnetEId  string    `json:"subnet_id" example:"subnet-123"`
	InternalIP string    `json:"internal_ip" example:"10.0.0.5"`
	ExternalIP string    `json:"external_ip" example:"1.2.3.4"`
	SNAT       []string  `json:"snat" example:"10.0.0.10,10.0.0.11"`
	DNAT       []EipDNAT `json:"dnat"`
	Nat        string    `json:"nat" example:"fip"`
}

// EipDNAT représente une règle DNAT pour un EIP.
type EipDNAT struct {
	ExternalPort string `json:"external_port" example:"80"`
	InternalIP   string `json:"internal_ip" example:"10.0.0.10"`
	InternalPort string `json:"internal_port" example:"8080"`
	Protocol     string `json:"protocol" example:"tcp"`
}

// EipRequest représente le corps de la requête pour créer ou mettre à jour un EIP.
type EipRequest struct {
	Name       string    `json:"name" example:"mon-eip"`
	SubnetEId  string    `json:"subnet_id" example:"subnet-123"`
	InternalIP string    `json:"internal_ip,omitempty" example:"10.0.0.5"`
	SNAT       []string  `json:"snat,omitempty"`
	DNAT       []EipDNAT `json:"dnat,omitempty"`
}

// Validate vérifie les contraintes sur l'EIP.
func (e *EipRequest) Validate() error {
	if e.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(e.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(e.Name))
	}
	if e.SubnetEId == "" {
		return errors.New("le subnet_id est requis")
	}

	hasInternalIP := e.InternalIP != ""
	hasSnatOrDnat := len(e.SNAT) > 0 || len(e.DNAT) > 0

	if !hasInternalIP && !hasSnatOrDnat {
		return errors.New("vous devez fournir soit une internal_ip, soit des configurations SNAT/DNAT")
	}
	if hasInternalIP && hasSnatOrDnat {
		return errors.New("vous ne pouvez pas fournir à la fois une internal_ip et des configurations SNAT/DNAT")
	}

	for _, cidr := range e.SNAT {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("format CIDR invalide pour SNAT : %s", cidr)
		}
	}

	return nil
}
