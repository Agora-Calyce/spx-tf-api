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

// Subnet represents the Subnet resource for Terraform.
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

// SubnetRequest represents the request body to create or update a Subnet.
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

// Validate checks the constraints on the Subnet.
func (s *SubnetRequest) Validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if len(s.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(s.Name))
	}
	if s.VPCID == "" {
		return errors.New("vpc_id is required")
	}

	switch s.Protocol {
	case ProtocolIPv4:
		if s.IPv4 == "" {
			return errors.New("the ipv4 field is required for the IPv4 protocol")
		}
	case ProtocolIPv6:
		if s.IPv6 == "" {
			return errors.New("the ipv6 field is required for the IPv6 protocol")
		}
	case ProtocolDual:
		if s.IPv4 == "" || s.IPv6 == "" {
			return errors.New("both ipv4 and ipv6 fields are required for the Dual protocol")
		}
	case "":
		return errors.New("protocol is required (IPv4, IPv6 or Dual)")
	default:
		return fmt.Errorf("invalid protocol: %s (must be IPv4, IPv6 or Dual)", s.Protocol)
	}

	return nil
}
