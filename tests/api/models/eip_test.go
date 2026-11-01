package models_test

import (
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"strings"
	"testing"
)

func TestEipRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.EipRequest
		wantErr bool
	}{
		{
			name: "Valide avec InternalIP",
			request: models.EipRequest{
				Name:       "mon-eip",
				SubnetEId:  "subnet-123",
				InternalIP: "10.0.0.5",
			},
			wantErr: false,
		},
		{
			name: "Valide avec SNAT (CIDR)",
			request: models.EipRequest{
				Name:      "mon-eip",
				SubnetEId: "subnet-123",
				SNAT:      []string{"10.0.0.0/24"},
			},
			wantErr: false,
		},
		{
			name: "Invalide : SNAT n'est pas un CIDR",
			request: models.EipRequest{
				Name:      "mon-eip",
				SubnetEId: "subnet-123",
				SNAT:      []string{"10.0.0.10"},
			},
			wantErr: true,
		},
		{
			name: "Valide avec DNAT",
			request: models.EipRequest{
				Name:      "mon-eip",
				SubnetEId: "subnet-123",
				DNAT: []models.EipDNAT{
					{ExternalPort: "80", InternalIP: "10.0.0.10", InternalPort: "8080", Protocol: "TCP"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalide : les deux fournis",
			request: models.EipRequest{
				Name:       "mon-eip",
				SubnetEId:  "subnet-123",
				InternalIP: "10.0.0.5",
				SNAT:       []string{"10.0.0.10"},
			},
			wantErr: true,
		},
		{
			name: "Invalide : aucun fourni",
			request: models.EipRequest{
				Name:      "mon-eip",
				SubnetEId: "subnet-123",
			},
			wantErr: true,
		},
		{
			name: "Nom vide",
			request: models.EipRequest{
				Name:      "",
				SubnetEId: "subnet-123",
			},
			wantErr: true,
		},
		{
			name: "Nom trop long",
			request: models.EipRequest{
				Name:      strings.Repeat("a", 64),
				SubnetEId: "subnet-123",
			},
			wantErr: true,
		},
		{
			name: "Subnet ID vide",
			request: models.EipRequest{
				Name:      "mon-eip",
				SubnetEId: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.request.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EipRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
