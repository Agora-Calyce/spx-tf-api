package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MaxSecurityGroupSubnets est le nombre maximal de subnets ciblés par un groupe de sécurité
const MaxSecurityGroupSubnets = 10

// SecurityGroup représente un groupe de sécurité dans l'API interne du proxy
type SecurityGroup struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	SubnetIDs   []string      `json:"subnetIds"`
}

// SecurityGroupRequest représente une requête de création ou mise à jour de groupe de sécurité
type SecurityGroupRequest struct {
	Name        string        `json:"name" validate:"required"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	// SubnetIDs limite le groupe de sécurité à ces subnets (vide = tous les subnets)
	SubnetIDs []string `json:"subnetIds"`
}

// LabelSelector représente un sélecteur de labels.
// MatchLabels est une liste de "clé:valeur".
type LabelSelector struct {
	MatchLabels      []string          `json:"matchLabels"`
	MatchExpressions []MatchExpression `json:"matchExpressions"`
}

// MatchExpression représente une expression de correspondance
type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// IngressRule représente une règle entrante
type IngressRule struct {
	Ports    []SgPort `json:"ports"`
	From     []Peer   `json:"from"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

// EgressRule représente une règle sortante
type EgressRule struct {
	Ports    []SgPort `json:"ports"`
	To       []Peer   `json:"to"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

// SgPort représente un port ou une plage de ports
type SgPort struct {
	Port     PortNumber `json:"port"`
	EndPort  PortNumber `json:"endPort"`
	Protocol string     `json:"protocol"`
}

// Peer représente un pair (pod selector ou IP block)
type Peer struct {
	PodSelector *LabelSelector `json:"podSelector,omitempty"`
	IPBlock     *IPBlock       `json:"IPBlock,omitempty"`
}

// IPBlock représente un bloc d'adresses IP
type IPBlock struct {
	CIDR   string   `json:"CIDR"`
	Except []string `json:"Except"`
}

// PortNumber est un numéro de port accepté en entrée sous forme de nombre ou de chaîne
// (compatibilité avec les clients qui envoient "80"), et toujours sérialisé en nombre.
type PortNumber int32

// UnmarshalJSON accepte un nombre, une chaîne numérique ou une chaîne vide (0)
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
			return fmt.Errorf("port invalide: %q", s)
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

// SplitMatchLabel découpe un label "clé:valeur"
func SplitMatchLabel(label string) (string, string, error) {
	key, value, ok := strings.Cut(label, ":")
	if !ok || key == "" {
		return "", "", fmt.Errorf("label invalide %q (format attendu: clé:valeur)", label)
	}
	return key, value, nil
}

// Validate valide la requête de groupe de sécurité
func (r *SecurityGroupRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("le nom est requis")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(r.Name))
	}

	if len(r.SubnetIDs) > MaxSecurityGroupSubnets {
		return fmt.Errorf("subnet_ids ne doit pas contenir plus de %d subnets (actuel: %d)", MaxSecurityGroupSubnets, len(r.SubnetIDs))
	}

	hasLabels := len(r.Target.MatchLabels) > 0
	hasExpressions := len(r.Target.MatchExpressions) > 0

	if !hasLabels && !hasExpressions {
		return fmt.Errorf("la cible (target) doit contenir soit match_labels, soit match_expressions")
	}

	if hasLabels && hasExpressions {
		return fmt.Errorf("la cible (target) ne peut pas contenir à la fois match_labels et match_expressions")
	}

	if err := validateLabelSelector(r.Target); err != nil {
		return fmt.Errorf("erreur dans la cible (target): %w", err)
	}

	// Validation des ports pour ingress
	if len(r.Ingress) > 1 {
		for _, ingress := range r.Ingress {
			if ingress.AllowAll {
				return fmt.Errorf("une règle ingress.allow_all=true ne peut pas coexister avec d'autres règles ingress")
			}
		}
	}
	for _, ingress := range r.Ingress {
		for i := range ingress.Ports {
			if err := validatePort(&ingress.Ports[i]); err != nil {
				return fmt.Errorf("erreur dans la règle ingress: %w", err)
			}
		}
		for _, from := range ingress.From {
			if err := validatePeer(from); err != nil {
				return fmt.Errorf("erreur dans la règle ingress (from): %w", err)
			}
		}
	}

	// Validation des ports pour egress
	if len(r.Egress) > 1 {
		for _, egress := range r.Egress {
			if egress.AllowAll {
				return fmt.Errorf("une règle egress.allow_all=true ne peut pas coexister avec d'autres règles egress")
			}
		}
	}
	for _, egress := range r.Egress {
		for i := range egress.Ports {
			if err := validatePort(&egress.Ports[i]); err != nil {
				return fmt.Errorf("erreur dans la règle egress: %w", err)
			}
		}
		for _, to := range egress.To {
			if err := validatePeer(to); err != nil {
				return fmt.Errorf("erreur dans la règle egress (to): %w", err)
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
			return fmt.Errorf("opérateur d'expression invalide: %s (doit être parmis In, NotIn, Exists, DoesNotExists)", exp.Operator)
		}
	}
	return nil
}

func validatePeer(p Peer) error {
	hasPodSelector := p.PodSelector != nil && (len(p.PodSelector.MatchLabels) > 0 || len(p.PodSelector.MatchExpressions) > 0)
	hasIPBlock := p.IPBlock != nil && p.IPBlock.CIDR != ""

	if !hasPodSelector && !hasIPBlock {
		return fmt.Errorf("le pair doit contenir soit un sélecteur de pods (pod_selector), soit un bloc IP (ip_block.cidr)")
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
		return fmt.Errorf("le port doit être compris entre 1 et 65535 (valeur: %d)", p.Port)
	}
	// Si end_port n'est pas défini, on prend la valeur de port
	if p.EndPort == 0 {
		p.EndPort = p.Port
	}
	if p.EndPort < 1 || p.EndPort > 65535 {
		return fmt.Errorf("le port final (end_port) doit être compris entre 1 et 65535 (valeur: %d)", p.EndPort)
	}
	if p.EndPort < p.Port {
		return fmt.Errorf("le port final (end_port: %d) doit être supérieur ou égal au port (port: %d)", p.EndPort, p.Port)
	}
	// Protocole par défaut : TCP
	if p.Protocol == "" {
		p.Protocol = "TCP"
	}
	p.Protocol = strings.ToUpper(p.Protocol)
	switch p.Protocol {
	case "TCP", "UDP", "SCTP":
	default:
		return fmt.Errorf("protocole invalide: %s (valeurs possibles: TCP, UDP, SCTP)", p.Protocol)
	}
	return nil
}
