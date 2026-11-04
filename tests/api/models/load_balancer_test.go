package models_test

import (
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadBalancerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.LoadBalancerRequest
		wantErr bool
		msg     string
	}{
		{
			name: "Valid",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			request: models.LoadBalancerRequest{
				VIP: "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "name is required",
		},
		{
			name: "Missing VIP",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "VIP is required",
		},
		{
			name: "VIP out of range",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "10.0.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "VIP address must be in the 198.18.0.0/16 range",
		},
		{
			name: "Invalid VIP",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "not-an-ip",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "invalid VIP address",
		},
		{
			name: "Mutually exclusive",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
				Selectors: []models.LBSelector{
					{Key: "app", Value: "test"},
				},
				Endpoints: []string{"10.0.0.2"},
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "selectors and endpoints are mutually exclusive",
		},
		{
			name: "Missing ports",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
			},
			wantErr: true,
			msg:     "at least one port is required",
		},
		{
			name: "Invalid protocol",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "HTTP"},
				},
			},
			wantErr: true,
			msg:     "invalid protocol",
		},
		{
			name: "Valid with Selectors",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
				Selectors: []models.LBSelector{
					{Key: "app", Value: "test"},
				},
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid with Endpoints",
			request: models.LoadBalancerRequest{
				Name:      "lb-test",
				VIP:       "198.18.0.1",
				Endpoints: []string{"10.0.0.1", "10.0.0.2"},
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.msg != "" {
					assert.Contains(t, err.Error(), tt.msg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
