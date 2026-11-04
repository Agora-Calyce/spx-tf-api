package models

import (
	"errors"
	"fmt"
)

// VPC represents the VPC resource for Terraform.
type VPC struct {
	ID           string        `json:"id" example:"vpc-12345"`
	Name         string        `json:"name" example:"mon-vpc-reseau"`
	AZ           string        `json:"az" example:"az1"`
	StaticRoutes []StaticRoute `json:"static_routes"`
}

// StaticRoute represents a static route of a VPC.
// Routes generated automatically by NAT Gateways are not exposed.
type StaticRoute struct {
	Policy     string `json:"policy" example:"policyDst"` // policyDst or policySrc
	CIDR       string `json:"cidr" example:"192.168.0.0/24"`
	NextHopIP  string `json:"next_hop_ip" example:"10.0.0.254"`
	RouteTable string `json:"route_table" example:"subnet-eid"` // Subnet EID
}

// VPCRequest represents the request body to create or update a VPC.
type VPCRequest struct {
	Name string `json:"name" example:"mon-vpc-reseau"`
	// StaticRoutes (update only): nil = existing routes kept, [] = all removed
	StaticRoutes *[]StaticRoute `json:"static_routes,omitempty"`
}

// Validate checks that the VPC name does not exceed 63 characters.
func (v *VPCRequest) Validate() error {
	if v.Name == "" {
		return errors.New("name is required")
	}
	if len(v.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(v.Name))
	}
	if v.StaticRoutes != nil {
		for _, sr := range *v.StaticRoutes {
			if sr.CIDR == "" {
				return errors.New("cidr is required for each static route")
			}
			if sr.RouteTable == "" {
				return errors.New("route_table (subnet EID) is required for each static route")
			}
			if sr.Policy != "" && sr.Policy != "policyDst" && sr.Policy != "policySrc" {
				return fmt.Errorf("invalid policy: %s (possible values: policyDst, policySrc)", sr.Policy)
			}
		}
	}
	return nil
}
