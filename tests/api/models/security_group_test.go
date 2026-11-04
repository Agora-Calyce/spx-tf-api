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
			name:    "Valid with labels",
			req:     models.SecurityGroupRequest{Name: "sg-test", Target: webTarget},
			wantErr: false,
		},
		{
			name: "Valid with expressions",
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
			name:    "Name required",
			req:     models.SecurityGroupRequest{Target: webTarget},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "Invalid with nothing",
			req:     models.SecurityGroupRequest{Name: "sg-test"},
			wantErr: true,
			errMsg:  "target must contain either match_labels or match_expressions",
		},
		{
			name: "Invalid with both",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchLabels:      []string{"app:web"},
					MatchExpressions: []models.MatchExpression{{Key: "env", Operator: "In", Values: []string{"prod"}}},
				},
			},
			wantErr: true,
			errMsg:  "target cannot contain both match_labels and match_expressions",
		},
		{
			name: "Malformed label",
			req: models.SecurityGroupRequest{
				Name:   "sg-test",
				Target: models.LabelSelector{MatchLabels: []string{"app"}},
			},
			wantErr: true,
			errMsg:  "expected format: key:value",
		},
		{
			name: "Invalid operator",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchExpressions: []models.MatchExpression{{Key: "app", Operator: "Invalid", Values: []string{"web"}}},
				},
			},
			wantErr: true,
			errMsg:  "invalid expression operator: Invalid",
		},
		{
			name: "Valid Exists operator",
			req: models.SecurityGroupRequest{
				Name: "sg-test",
				Target: models.LabelSelector{
					MatchExpressions: []models.MatchExpression{{Key: "app", Operator: "Exists"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid port (too small)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 0, EndPort: 80, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "Invalid port (too large)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 70000, EndPort: 80, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "Missing EndPort (normalized)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}}},
			},
			wantErr: false,
		},
		{
			name: "Invalid EndPort (too large)",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 70000, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "end_port must be between 1 and 65535",
		},
		{
			name: "EndPort < Port",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 79, Protocol: "TCP"}}}},
			},
			wantErr: true,
			errMsg:  "must be greater than or equal to port",
		},
		{
			name: "Invalid protocol",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, Protocol: "ICMP"}}}},
			},
			wantErr: true,
			errMsg:  "invalid protocol: ICMP",
		},
		{
			name: "Valid ports",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80, EndPort: 80, Protocol: "TCP"}}}},
				Egress:  []models.EgressRule{{Ports: []models.SgPort{{Port: 1, EndPort: 65535, Protocol: "udp"}}}},
			},
			wantErr: false,
		},
		{
			name: "Ingress AllowAll with other rules",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{
					{AllowAll: true},
					{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}},
				},
			},
			wantErr: true,
			errMsg:  "an ingress.allow_all=true rule cannot coexist with other ingress rules",
		},
		{
			name: "Egress AllowAll with other rules",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Egress: []models.EgressRule{
					{AllowAll: true},
					{Ports: []models.SgPort{{Port: 80, Protocol: "TCP"}}},
				},
			},
			wantErr: true,
			errMsg:  "an egress.allow_all=true rule cannot coexist with other egress rules",
		},
		{
			name: "AllowAll alone is valid",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{AllowAll: true}},
				Egress:  []models.EgressRule{{AllowAll: true}},
			},
			wantErr: false,
		},
		{
			name: "Valid with Peer PodSelector",
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
			name: "Valid with Peer IPBlock CIDR",
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
			name: "Invalid with empty Peer",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Ingress: []models.IngressRule{{Ports: []models.SgPort{{Port: 80}}, From: []models.Peer{{}}}},
			},
			wantErr: true,
			errMsg:  "peer must contain either a pod selector (pod_selector) or an IP block (ip_block.cidr)",
		},
		{
			name: "Invalid with empty Egress Peer",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				Egress: []models.EgressRule{{
					Ports: []models.SgPort{{Port: 80}},
					To:    []models.Peer{{IPBlock: &models.IPBlock{CIDR: ""}}},
				}},
			},
			wantErr: true,
			errMsg:  "peer must contain either a pod selector (pod_selector) or an IP block (ip_block.cidr)",
		},
		{
			name: "10 subnets allowed",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				SubnetIDs: []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10"},
			},
			wantErr: false,
		},
		{
			name: "More than 10 subnets",
			req: models.SecurityGroupRequest{
				Name: "sg-test", Target: webTarget,
				SubnetIDs: []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10", "s11"},
			},
			wantErr: true,
			errMsg:  "subnet_ids must not contain more than 10 subnets",
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
	assert.Equal(t, models.PortNumber(443), req.Ingress[0].Ports[0].EndPort, "end_port defaults to port")
	assert.Equal(t, "UDP", req.Ingress[0].Ports[0].Protocol, "protocol uppercased")
	assert.Equal(t, "TCP", req.Ingress[0].Ports[1].Protocol, "TCP protocol by default")
}

// The Terraform provider sends ports as strings: they must stay accepted.
func TestPortNumber_UnmarshalJSON(t *testing.T) {
	var p models.SgPort
	assert.NoError(t, json.Unmarshal([]byte(`{"port": "8080", "endPort": "", "protocol": "TCP"}`), &p))
	assert.Equal(t, models.PortNumber(8080), p.Port)
	assert.Equal(t, models.PortNumber(0), p.EndPort)

	assert.NoError(t, json.Unmarshal([]byte(`{"port": 22, "endPort": 23}`), &p))
	assert.Equal(t, models.PortNumber(22), p.Port)
	assert.Equal(t, models.PortNumber(23), p.EndPort)

	assert.Error(t, json.Unmarshal([]byte(`{"port": "http"}`), &p))

	// Always serialized as a number
	out, _ := json.Marshal(models.SgPort{Port: 80, EndPort: 81, Protocol: "TCP"})
	assert.JSONEq(t, `{"port": 80, "endPort": 81, "protocol": "TCP"}`, string(out))
}
