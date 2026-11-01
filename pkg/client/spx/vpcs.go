package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListVPCs récupère la liste des VPCs pour un projet et une AZ donnés
func (c *Client) ListVPCs(ctx context.Context, orgaID, az, projectID string) ([]models.VPCFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vpc", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var vpcs []models.VPCFullResponse
	if err := decodeResponse(resp, &vpcs); err != nil {
		return nil, err
	}

	return vpcs, nil
}

// CreateVPC crée un nouveau VPC
func (c *Client) CreateVPC(ctx context.Context, orgaID, az, projectID string, body *models.CreateVPCBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vpc", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return nil, err
	}

	var createResp models.CreateResponse
	if err := decodeResponse(resp, &createResp); err != nil {
		return nil, err
	}

	return &createResp, nil
}

// GetVPC récupère un VPC par son ID effectif
func (c *Client) GetVPC(ctx context.Context, orgaID, az, projectID, effectiveID string) (*models.VPCFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vpc/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var vpc models.VPCFullResponse
	if err := decodeResponse(resp, &vpc); err != nil {
		return nil, err
	}

	return &vpc, nil
}

// UpdateVPC met à jour un VPC existant
func (c *Client) UpdateVPC(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateVPCBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vpc/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteVPC supprime un VPC
func (c *Client) DeleteVPC(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vpc/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
