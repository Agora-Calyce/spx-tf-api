package models_test

import (
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"strings"
	"testing"
)

func TestVPCRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		vpcName string
		wantErr bool
	}{
		{
			name:    "Valide",
			vpcName: "mon-vpc",
			wantErr: false,
		},
		{
			name:    "Nom vide",
			vpcName: "",
			wantErr: true,
		},
		{
			name:    "Nom exactement 63 caractères",
			vpcName: strings.Repeat("a", 63),
			wantErr: false,
		},
		{
			name:    "Nom trop long (64 caractères)",
			vpcName: strings.Repeat("a", 64),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &models.VPCRequest{
				Name: tt.vpcName,
			}
			if err := v.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("VPCRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
