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
			name: "Valide",
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
			name: "Nom manquant",
			request: models.LoadBalancerRequest{
				VIP: "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "le nom est requis",
		},
		{
			name: "VIP manquant",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "le VIP est requis",
		},
		{
			name: "VIP hors range",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "10.0.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "l'adresse VIP doit appartenir au range 198.18.0.0/16",
		},
		{
			name: "VIP invalide",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "not-an-ip",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "TCP"},
				},
			},
			wantErr: true,
			msg:     "adresse VIP invalide",
		},
		{
			name: "Mutuellement exclusif",
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
			msg:     "les sélecteurs et les endpoints sont mutuellement exclusifs",
		},
		{
			name: "Ports manquants",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
			},
			wantErr: true,
			msg:     "au moins un port est requis",
		},
		{
			name: "Protocole invalide",
			request: models.LoadBalancerRequest{
				Name: "lb-test",
				VIP:  "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 8080, Protocol: "HTTP"},
				},
			},
			wantErr: true,
			msg:     "protocole invalide",
		},
		{
			name: "Valide avec Selectors",
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
			name: "Valide avec Endpoints",
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
