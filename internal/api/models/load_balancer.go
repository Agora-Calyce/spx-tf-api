package models

import (
	"errors"
	"fmt"
	"net"
)

// LoadBalancer represents the Load Balancer resource for Terraform.
type LoadBalancer struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	AZ        string             `json:"az"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

// LBSelector represents a selector for a Load Balancer.
type LBSelector struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LoadBalancerPort represents a port for a Load Balancer.
type LoadBalancerPort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

// LoadBalancerRequest represents the request body to create or update an LB.
type LoadBalancerRequest struct {
	Name      string             `json:"name"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

// Validate checks the validity of the request.
func (r *LoadBalancerRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(r.Name))
	}
	if r.VIP == "" {
		return errors.New("VIP is required")
	}
	ip := net.ParseIP(r.VIP)
	if ip == nil {
		return fmt.Errorf("invalid VIP address: %s", r.VIP)
	}
	_, cidr, _ := net.ParseCIDR("198.18.0.0/16")
	if !cidr.Contains(ip) {
		return fmt.Errorf("VIP address must be in the 198.18.0.0/16 range (current: %s)", r.VIP)
	}
	if len(r.Selectors) > 0 && len(r.Endpoints) > 0 {
		return errors.New("selectors and endpoints are mutually exclusive")
	}
	if len(r.Ports) == 0 {
		return errors.New("at least one port is required")
	}
	for _, p := range r.Ports {
		if p.Port <= 0 || p.Port > 65535 {
			return fmt.Errorf("invalid port: %d", p.Port)
		}
		if p.TargetPort <= 0 || p.TargetPort > 65535 {
			return fmt.Errorf("invalid targetPort: %d", p.TargetPort)
		}
		if p.Protocol != "TCP" && p.Protocol != "UDP" {
			return fmt.Errorf("invalid protocol: %s (must be TCP or UDP)", p.Protocol)
		}
	}
	return nil
}
