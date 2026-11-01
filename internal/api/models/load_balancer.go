package models

import (
	"errors"
	"fmt"
	"net"
)

// LoadBalancer représente la ressource Load Balancer pour Terraform.
type LoadBalancer struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	AZ        string             `json:"az"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

// LBSelector représente un sélecteur pour un Load Balancer.
type LBSelector struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LoadBalancerPort représente un port pour un Load Balancer.
type LoadBalancerPort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

// LoadBalancerRequest représente le corps de la requête pour créer ou mettre à jour un LB.
type LoadBalancerRequest struct {
	Name      string             `json:"name"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

// Validate vérifie la validité de la requête.
func (r *LoadBalancerRequest) Validate() error {
	if r.Name == "" {
		return errors.New("le nom est requis")
	}
	if len(r.Name) > 63 {
		return fmt.Errorf("le nom ne doit pas dépasser 63 caractères (actuel: %d)", len(r.Name))
	}
	if r.VIP == "" {
		return errors.New("le VIP est requis")
	}
	ip := net.ParseIP(r.VIP)
	if ip == nil {
		return fmt.Errorf("adresse VIP invalide: %s", r.VIP)
	}
	_, cidr, _ := net.ParseCIDR("198.18.0.0/16")
	if !cidr.Contains(ip) {
		return fmt.Errorf("l'adresse VIP doit appartenir au range 198.18.0.0/16 (actuel: %s)", r.VIP)
	}
	if len(r.Selectors) > 0 && len(r.Endpoints) > 0 {
		return errors.New("les sélecteurs et les endpoints sont mutuellement exclusifs")
	}
	if len(r.Ports) == 0 {
		return errors.New("au moins un port est requis")
	}
	for _, p := range r.Ports {
		if p.Port <= 0 || p.Port > 65535 {
			return fmt.Errorf("port invalide: %d", p.Port)
		}
		if p.TargetPort <= 0 || p.TargetPort > 65535 {
			return fmt.Errorf("targetPort invalide: %d", p.TargetPort)
		}
		if p.Protocol != "TCP" && p.Protocol != "UDP" {
			return fmt.Errorf("protocole invalide: %s (doit être TCP ou UDP)", p.Protocol)
		}
	}
	return nil
}
