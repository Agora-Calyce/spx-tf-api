package models_test

import (
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"testing"
)

func TestConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  models.Configuration
		wantErr bool
	}{
		{
			name: "Configuration valide",
			config: models.Configuration{
				Endpoint: "https://api.example.com",
			},
			wantErr: false,
		},
		{
			name:    "Endpoint manquant",
			config:  models.Configuration{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Configuration.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
