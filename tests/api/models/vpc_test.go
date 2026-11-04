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
			name:    "Valid",
			vpcName: "mon-vpc",
			wantErr: false,
		},
		{
			name:    "Empty name",
			vpcName: "",
			wantErr: true,
		},
		{
			name:    "Name exactly 63 characters",
			vpcName: strings.Repeat("a", 63),
			wantErr: false,
		},
		{
			name:    "Name too long (64 characters)",
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
