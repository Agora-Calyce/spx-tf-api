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
			name: "Valid",
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
			name: "CPU too high",
			req: models.InstanceRequest{
				Name: "ma-vm",
				CPU:  33,
			},
			wantErr: true,
			msg:     "CPU count must not exceed 32",
		},
		{
			name: "Memory too high",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 65,
			},
			wantErr: true,
			msg:     "memory must not exceed 64 GB",
		},
		{
			name: "Empty name",
			req: models.InstanceRequest{
				Name: "",
			},
			wantErr: true,
			msg:     "name is required",
		},
		{
			name: "Invalid CPU",
			req: models.InstanceRequest{
				Name: "ma-vm",
				CPU:  0,
			},
			wantErr: true,
			msg:     "CPU count must be greater than 0",
		},
		{
			name: "Invalid memory",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 0,
			},
			wantErr: true,
			msg:     "memory must be greater than 0",
		},
		{
			name: "Missing network",
			req: models.InstanceRequest{
				Name:    "ma-vm",
				CPU:     1,
				Memory:  1,
				Network: []models.InstanceNetwork{},
			},
			wantErr: true,
			msg:     "at least one network is required",
		},
		{
			name: "Missing subnet ID",
			req: models.InstanceRequest{
				Name:   "ma-vm",
				CPU:    1,
				Memory: 1,
				Network: []models.InstanceNetwork{
					{SubnetID: ""},
				},
			},
			wantErr: true,
			msg:     "the subnet ID is required for each network interface",
		},
		{
			name: "Valid labels",
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
			name: "Invalid label",
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
			msg:     "label key 'env:prod' must start with 'user.superphenix.net/'",
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
