package models_test

import (
	"encoding/json"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityGroupRequest_Validate(t *testing.T) {
	webTarget := models.LabelSelector{MatchLabels: []string{"app:web"}}

	tests := []struct {
		name    string
		req     models.SecurityGroupRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valide avec labels",
			req:     models.SecurityGroupRequest{Name: "sg-test", Target: webTarget},
			wantErr: false,
		},
		{
			name: "Valide avec expressions",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchExpressions: []models.MatchExpression{
						{Key: "app", Operator: "In", Values: []string{"web", "api"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nom requis",
			req:     models.SecurityGroupRequest{Target: webTarget},
			wantErr: true,
			errMsg:  "le nom est requis",
		},
		{
			name:    "Invalide sans rien",
			req:     models.SecurityGroupRequest{Name: "sg-test"},
			wantErr: true,
			errMsg:  "la cible (target) doit contenir soit match_labels, soit match_expressions",
		},
		{
			name: "Invalide avec les deux",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchLabels:      []string{"app:web"},
					MatchExpressions: []models.MatchExpression{{Key: "env", Operator: "In", Values: []string{"prod"}}},
				},
			},
			wantErr: true,
			errMsg:  "la cible (target) ne peut pas contenir à la fois match_labels et match_expressions",
		},
		{
			name: "Label mal formé",
			req: models.SecurityGroupRequest{
				Name:   "sg-test",
				Target: models.LabelSelector{MatchLabels: []string{"app"}},
			},
			wantErr: true,
			errMsg:  "format attendu: clé:valeur",
		},
		{
			name: "Opérateur invalide",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchExpressions: []models.MatchExpression{{Key: "app", Operator: "Invalid", Values: []string{"web"}}},
				},
			},
			wantErr: true,
			errMsg:  "opérateur d'expression invalide: Invalid",
		},
		{
			name: "Opérateur Exists valide",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchExpressions: []models.MatchExpression{{Key: "app", Operator: "Exists"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Port invalide (trop petit)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 0, EndPort: 80, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "le port doit être compris entre 1 et 65535",
		},
		{
			name: "Port invalide (trop grand)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 70000, EndPort: 80, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "le port doit être compris entre 1 et 65535",
		},
		{
			name: "EndPort absent (normalisé)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}}},
			},
			wantErr: false,
		},
		{
			name: "EndPort invalide (trop grand)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 70000, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "le port final (end_port) doit être compris entre 1 et 65535",
		},
		{
			name: "EndPort < Port",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 79, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "doit être supérieur ou égal au port",
		},
		{
			name: "Protocole invalide",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, Protocol: "ICMP"}}}},
			},
			wantErr: true,
			errMsg:  "protocole invalide: ICMP",
		},
		{
			name: "Ports valides",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 80, Protocol: "TCP"}}}},
				Egress:  []models.EgressRule{{Ports: []models.SgPort{{Port: 1, EndPort: 65535, Protocol: "udp"}}}},
			},
			wantErr: false,
		},
		{
			name: "Ingress AllowAll avec d'autres règles",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{
					{AllowAll: true},
					{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}},
				},
			},
			wantErr: true,
			errMsg:  "une règle ingress.allow_all=true ne peut pas coexister avec d'autres règles ingress",
		},
		{
			name: "Egress AllowAll avec d'autres règles",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Egress: []models.EgressRule{
					{AllowAll: true},
					{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}},
				},
			},
			wantErr: true,
			errMsg:  "une règle egress.allow_all=true ne peut pas coexister avec d'autres règles egress",
		},
		{
			name: "AllowAll seul est valide",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{AllowAll: true}},
				Egress:  []models.EgressRule{{AllowAll: true}},
			},
			wantErr: false,
		},
		{
			name: "Valide avec Peer PodSelector",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{
					Ports: []models.SgPort{{Port: 80}},
					From:  []models.Peer{{PodSelector: &models.LabelSelector{MatchLabels: []string{"app:db"}}}},
				}},
			},
			wantErr: false,
		},
		{
			name: "Valide avec Peer IPBlock CIDR",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{
					Ports: []models.SgPort{{Port: 80}},
					From:  []models.Peer{{IPBlock: &models.IPBlock{CIDR: "10.0.0.0/24"}}},
				}},
			},
			wantErr: false,
		},
		{
			name: "Invalide avec Peer vide",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80}}, From: []models.Peer{{}}}},
			},
			wantErr: true,
			errMsg:  "le pair doit contenir soit un sélecteur de pods (pod_selector), soit un bloc IP (ip_block.cidr)",
		},
		{
			name: "Invalide avec Egress Peer vide",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Egress: []models.EgressRule{{
					Ports: []models.SgPort{{Port: 80}},
					To:    []models.Peer{{IPBlock: &models.IPBlock{CIDR: ""}}},
				}},
			},
			wantErr: true,
			errMsg:  "le pair doit contenir soit un sélecteur de pods (pod_selector), soit un bloc IP (ip_block.cidr)",
		},
		{
			name: "10 subnets autorisés",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				SubnetIDs: []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10"},
			},
			wantErr: false,
		},
		{
			name: "Plus de 10 subnets",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				SubnetIDs: []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10", "s11"},
			},
			wantErr: true,
			errMsg:  "subnet_ids ne doit pas contenir plus de 10 subnets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityGroupRequest_Validate_Normalizes(t *testing.T) {
	req := models.SecurityGroupRequest{
		Name:    "sg-test",
		Target:  models.LabelSelector{MatchLabels: []string{"app:web"}},
		Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 443, Protocol: "udp"}, {Port: 80}}}},
	}
	assert.NoError(t, req.Validate())
	assert.Equal(t, models.PortNumber(443), req.Ingress[0].Ports[0].EndPort, "end_port vaut port par défaut")
	assert.Equal(t, "UDP", req.Ingress[0].Ports[0].Protocol, "protocole mis en majuscules")
	assert.Equal(t, "TCP", req.Ingress[0].Ports[1].Protocol, "protocole TCP par défaut")
}

// Le provider Terraform envoie les ports sous forme de chaînes : ils doivent rester acceptés.
func TestPortNumber_UnmarshalJSON(t *testing.T) {
	var p models.SgPort
	assert.NoError(t, json.Unmarshal([]byte(`{"port": "8080", "endPort": "", "protocol": "TCP"}`), &p))
	assert.Equal(t, models.PortNumber(8080), p.Port)
	assert.Equal(t, models.PortNumber(0), p.EndPort)

	assert.NoError(t, json.Unmarshal([]byte(`{"port": 22, "endPort": 23}`), &p))
	assert.Equal(t, models.PortNumber(22), p.Port)
	assert.Equal(t, models.PortNumber(23), p.EndPort)

	assert.Error(t, json.Unmarshal([]byte(`{"port": "http"}`), &p))

	// Toujours sérialisé en nombre
	out, _ := json.Marshal(models.SgPort{Port: 80, EndPort: 81, Protocol: "TCP"})
	assert.JSONEq(t, `{"port": 80, "endPort": 81, "protocol": "TCP"}`, string(out))
}
