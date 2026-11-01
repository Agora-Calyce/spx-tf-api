package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListInstances récupère la liste des instances pour un projet et une AZ donnés
func (c *Client) ListInstances(ctx context.Context, orgaID, az, projectID string) ([]models.InstanceFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/instance", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var instances []models.InstanceFullResponse
	if err := decodeResponse(resp, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

// CreateInstance crée une nouvelle instance
func (c *Client) CreateInstance(ctx context.Context, orgaID, az, projectID string, body *models.CreateInstanceBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/instance", orgaID, az, projectID)
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

// GetInstance récupère une instance par son ID effectif
func (c *Client) GetInstance(ctx context.Context, orgaID, az, projectID, effectiveID string) (*models.InstanceFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/instance/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var instance models.InstanceFullResponse
	if err := decodeResponse(resp, &instance); err != nil {
		return nil, err
	}

	return &instance, nil
}

// UpdateInstance met à jour une instance existante
func (c *Client) UpdateInstance(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateInstanceBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/instance/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteInstance supprime une instance
func (c *Client) DeleteInstance(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/instance/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
