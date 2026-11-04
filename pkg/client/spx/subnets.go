package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListSubnets retrieves the list of subnets for a given project and AZ
func (c *Client) ListSubnets(ctx context.Context, orgaID, az, projectID string) ([]models.SubnetFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/subnet", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var subnets []models.SubnetFullResponse
	if err := decodeResponse(resp, &subnets); err != nil {
		return nil, err
	}

	return subnets, nil
}

// CreateSubnet creates a new subnet
func (c *Client) CreateSubnet(ctx context.Context, orgaID, az, projectID string, body *models.CreateSubnetBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/subnet", orgaID, az, projectID)
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

// GetSubnet retrieves a subnet by its ID
func (c *Client) GetSubnet(ctx context.Context, orgaID, az, projectID, effectiveID string) (*models.SubnetFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/subnet/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var subnet models.SubnetFullResponse
	if err := decodeResponse(resp, &subnet); err != nil {
		return nil, err
	}

	return &subnet, nil
}

// UpdateSubnet updates an existing subnet
func (c *Client) UpdateSubnet(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateSubnetBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/subnet/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteSubnet deletes a subnet
func (c *Client) DeleteSubnet(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/subnet/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
