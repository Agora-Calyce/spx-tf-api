package models_test

import (
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.InstanceRequest
		wantErr bool
		msg     string
	}{
		{
			name: "Valide",
			req: models.InstanceRequest{
				Name:        "ma-vm",
				RunStrategy: "Always",
				CPU:         1,
				Memory:      2,
				Network: []models.InstanceNetwork{
					{SubnetID: "sub1"},
				},
			},
			wantErr: false,
		},
		{
			name: "CPU trop élevé",
			req: models.InstanceRequest{
				Name: "ma-vm",
				CPU:  33,
			},
			wantErr: true,
			msg:     "le nombre de CPU ne doit pas dépasser 32",
		},
		{
			name: "Mémoire trop élevée",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 65,
			},
			wantErr: true,
			msg:     "la mémoire ne doit pas dépasser 64 GB",
		},
		{
			name: "Nom vide",
			req: models.InstanceRequest{
				Name: "",
			},
			wantErr: true,
			msg:     "le nom est requis",
		},
		{
			name: "CPU invalide",
			req: models.InstanceRequest{
				Name: "ma-vm",
				CPU:  0,
			},
			wantErr: true,
			msg:     "le nombre de CPU doit être supérieur à 0",
		},
		{
			name: "Mémoire invalide",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 0,
			},
			wantErr: true,
			msg:     "la mémoire doit être supérieure à 0",
		},
		{
			name: "Réseau manquant",
			req: models.InstanceRequest{
				Name:    "ma-vm",
				CPU:     1,
				Memory:  1,
				Network: []models.InstanceNetwork{},
			},
			wantErr: true,
			msg:     "au moins un réseau est requis",
		},
		{
			name: "Subnet ID manquant",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 1,
				Network: []models.InstanceNetwork{
					{SubnetID: ""},
				},
			},
			wantErr: true,
			msg:     "l'ID du subnet est requis pour chaque interface réseau",
		},
		{
			name: "Labels valides",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 1,
				Network: []models.InstanceNetwork{
					{SubnetID: "sub1"},
				},
				Labels: []string{"user.superphenix.net/env:prod", "user.superphenix.net/app:web"},
			},
			wantErr: false,
		},
		{
			name: "Label invalide",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 1,
				Network: []models.InstanceNetwork{
					{SubnetID: "sub1"},
				},
				Labels: []string{"env:prod"},
			},
			wantErr: true,
			msg:     "la clé du label 'env:prod' doit commencer par 'user.superphenix.net/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
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
