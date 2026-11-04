package models

import (
	"errors"
	"fmt"
	"net"
)

// Eip represents the EIP resource for Terraform.
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

// EipDNAT represents a DNAT rule for an EIP.
type EipDNAT struct {
	ExternalPort string `json:"external_port" example:"80"`
	InternalIP   string `json:"internal_ip" example:"10.0.0.10"`
	InternalPort string `json:"internal_port" example:"8080"`
	Protocol     string `json:"protocol" example:"tcp"`
}

// EipRequest represents the request body to create or update an EIP.
type EipRequest struct {
	Name       string    `json:"name" example:"mon-eip"`
	SubnetEId  string    `json:"subnet_id" example:"subnet-123"`
	InternalIP string    `json:"internal_ip,omitempty" example:"10.0.0.5"`
	SNAT       []string  `json:"snat,omitempty"`
	DNAT       []EipDNAT `json:"dnat,omitempty"`
}

// Validate checks the constraints on the EIP.
func (e *EipRequest) Validate() error {
	if e.Name == "" {
		return errors.New("name is required")
	}
	if len(e.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(e.Name))
	}
	if e.SubnetEId == "" {
		return errors.New("subnet_id is required")
	}

	hasInternalIP := e.InternalIP != ""
	hasSnatOrDnat := len(e.SNAT) > 0 || len(e.DNAT) > 0

	if !hasInternalIP && !hasSnatOrDnat {
		return errors.New("you must provide either an internal_ip or SNAT/DNAT configurations")
	}
	if hasInternalIP && hasSnatOrDnat {
		return errors.New("you cannot provide both an internal_ip and SNAT/DNAT configurations")
	}

	for _, cidr := range e.SNAT {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR format for SNAT: %s", cidr)
		}
	}

	return nil
}
