package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListEips retrieves the list of EIPs for a given project and AZ
func (c *Client) ListEips(ctx context.Context, orgaID, az, projectID string) ([]models.EipFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/eip", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var eips []models.EipFullResponse
	if err := decodeResponse(resp, &eips); err != nil {
		return nil, err
	}

	return eips, nil
}

// CreateEip creates a new EIP
func (c *Client) CreateEip(ctx context.Context, orgaID, az, projectID string, body *models.CreateEipBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/eip", orgaID, az, projectID)
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

// GetEip retrieves an EIP by its ID
func (c *Client) GetEip(ctx context.Context, orgaID, az, projectID, effectiveID string) (*models.EipFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/eip/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var eip models.EipFullResponse
	if err := decodeResponse(resp, &eip); err != nil {
		return nil, err
	}

	return &eip, nil
}

// UpdateEip updates an existing EIP
func (c *Client) UpdateEip(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateEipBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/eip/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteEip deletes an EIP
func (c *Client) DeleteEip(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/eip/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
