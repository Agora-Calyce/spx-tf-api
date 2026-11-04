package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MaxSecurityGroupSubnets is the maximum number of subnets targeted by a security group
const MaxSecurityGroupSubnets = 10

// SecurityGroup represents a security group in the proxy's internal API
type SecurityGroup struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	SubnetIDs   []string      `json:"subnetIds"`
}

// SecurityGroupRequest represents a security group create or update request
type SecurityGroupRequest struct {
	Name        string        `json:"name" validate:"required"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	// SubnetIDs limits the security group to these subnets (empty = all subnets)
	SubnetIDs []string `json:"subnetIds"`
}

// LabelSelector represents a label selector.
// MatchLabels is a list of "key:value".
type LabelSelector struct {
	MatchLabels      []string          `json:"matchLabels"`
	MatchExpressions []MatchExpression `json:"matchExpressions"`
}

// MatchExpression represents a match expression
type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Ports    []SgPort `json:"ports"`
	From     []Peer   `json:"from"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

// EgressRule represents an egress rule
type EgressRule struct {
	Ports    []SgPort `json:"ports"`
	To       []Peer   `json:"to"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

// SgPort represents a port or a port range
type SgPort struct {
	Port     PortNumber `json:"port"`
	EndPort  PortNumber `json:"endPort"`
	Protocol string     `json:"protocol"`
}

// Peer represents a peer (pod selector or IP block)
type Peer struct {
	PodSelector *LabelSelector `json:"podSelector,omitempty"`
	IPBlock     *IPBlock       `json:"IPBlock,omitempty"`
}

// IPBlock represents an IP address block
type IPBlock struct {
	CIDR   string   `json:"CIDR"`
	Except []string `json:"Except"`
}

// PortNumber is a port number accepted as input as a number or a string
// (compatibility with clients that send "80"), and always serialized as a number.
type PortNumber int32

// UnmarshalJSON accepts a number, a numeric string or an empty string (0)
func (p *PortNumber) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*p = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid port: %q", s)
		}
		*p = PortNumber(n)
		return nil
	}
	var n int32
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*p = PortNumber(n)
	return nil
}

// SplitMatchLabel splits a "key:value" label
func SplitMatchLabel(label string) (string, string, error) {
	key, value, ok := strings.Cut(label, ":")
	if !ok || key == "" {
		return "", "", fmt.Errorf("invalid label %q (expected format: key:value)", label)
	}
	return key, value, nil
}

// Validate validates the security group request
func (r *SecurityGroupRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("name must not exceed 63 characters (current: %d)", len(r.Name))
	}

	if len(r.SubnetIDs) > MaxSecurityGroupSubnets {
		return fmt.Errorf("subnet_ids must not contain more than %d subnets (current: %d)", MaxSecurityGroupSubnets, len(r.SubnetIDs))
	}

	hasLabels := len(r.Target.MatchLabels) > 0
	hasExpressions := len(r.Target.MatchExpressions) > 0

	if !hasLabels && !hasExpressions {
		return fmt.Errorf("target must contain either match_labels or match_expressions")
	}

	if hasLabels && hasExpressions {
		return fmt.Errorf("target cannot contain both match_labels and match_expressions")
	}

	if err := validateLabelSelector(r.Target); err != nil {
		return fmt.Errorf("error in target: %w", err)
	}

	// Validate the ingress ports
	if len(r.Ingress) > 1 {
		for _, ingress := range r.Ingress {
			if ingress.AllowAll {
				return fmt.Errorf("an ingress.allow_all=true rule cannot coexist with other ingress rules")
			}
		}
	}
	for _, ingress := range r.Ingress {
		for i := range ingress.Ports {
			if err := validatePort(&ingress.Ports[i]); err != nil {
				return fmt.Errorf("error in ingress rule: %w", err)
			}
		}
		for _, from := range ingress.From {
			if err := validatePeer(from); err != nil {
				return fmt.Errorf("error in ingress rule (from): %w", err)
			}
		}
	}

	// Validate the egress ports
	if len(r.Egress) > 1 {
		for _, egress := range r.Egress {
			if egress.AllowAll {
				return fmt.Errorf("an egress.allow_all=true rule cannot coexist with other egress rules")
			}
		}
	}
	for _, egress := range r.Egress {
		for i := range egress.Ports {
			if err := validatePort(&egress.Ports[i]); err != nil {
				return fmt.Errorf("error in egress rule: %w", err)
			}
		}
		for _, to := range egress.To {
			if err := validatePeer(to); err != nil {
				return fmt.Errorf("error in egress rule (to): %w", err)
			}
		}
	}

	return nil
}

func validateLabelSelector(ls LabelSelector) error {
	for _, l := range ls.MatchLabels {
		if _, _, err := SplitMatchLabel(l); err != nil {
			return err
		}
	}
	validOps := map[string]bool{
		"In":            true,
		"NotIn":         true,
		"Exists":        true,
		"DoesNotExists": true,
	}
	for _, exp := range ls.MatchExpressions {
		if !validOps[exp.Operator] {
			return fmt.Errorf("invalid expression operator: %s (must be one of In, NotIn, Exists, DoesNotExists)", exp.Operator)
		}
	}
	return nil
}

func validatePeer(p Peer) error {
	hasPodSelector := p.PodSelector != nil && (len(p.PodSelector.MatchLabels) > 0 || len(p.PodSelector.MatchExpressions) > 0)
	hasIPBlock := p.IPBlock != nil && p.IPBlock.CIDR != ""

	if !hasPodSelector && !hasIPBlock {
		return fmt.Errorf("peer must contain either a pod selector (pod_selector) or an IP block (ip_block.cidr)")
	}

	if hasPodSelector {
		if err := validateLabelSelector(*p.PodSelector); err != nil {
			return err
		}
	}

	return nil
}

func validatePort(p *SgPort) error {
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535 (value: %d)", p.Port)
	}
	// If end_port is not set, use the port value
	if p.EndPort == 0 {
		p.EndPort = p.Port
	}
	if p.EndPort < 1 || p.EndPort > 65535 {
		return fmt.Errorf("end_port must be between 1 and 65535 (value: %d)", p.EndPort)
	}
	if p.EndPort < p.Port {
		return fmt.Errorf("end_port (%d) must be greater than or equal to port (%d)", p.EndPort, p.Port)
	}
	// Default protocol: TCP
	if p.Protocol == "" {
		p.Protocol = "TCP"
	}
	p.Protocol = strings.ToUpper(p.Protocol)
	switch p.Protocol {
	case "TCP", "UDP", "SCTP":
	default:
		return fmt.Errorf("invalid protocol: %s (possible values: TCP, UDP, SCTP)", p.Protocol)
	}
	return nil
}
