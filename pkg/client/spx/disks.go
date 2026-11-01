package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListDisks récupère la liste des disques pour un projet et une AZ donnés
func (c *Client) ListDisks(ctx context.Context, orgaID, az, projectID string) ([]models.DiskFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var disks []models.DiskFullResponse
	if err := decodeResponse(resp, &disks); err != nil {
		return nil, err
	}

	return disks, nil
}

// CreateDisk crée un nouveau disque
func (c *Client) CreateDisk(ctx context.Context, orgaID, az, projectID string, body *models.CreateDiskBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk", orgaID, az, projectID)
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

// GetDisk récupère un disque par son ID effectif
func (c *Client) GetDisk(ctx context.Context, orgaID, az, projectID, effectiveID string) (*models.DiskFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var disk models.DiskFullResponse
	if err := decodeResponse(resp, &disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

// UpdateDisk met à jour un disque existant
func (c *Client) UpdateDisk(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateDiskBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteDisk supprime un disque
func (c *Client) DeleteDisk(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// ListStorageClasses récupère la liste des StorageClasses pour un projet et une AZ donnés
func (c *Client) ListStorageClasses(ctx context.Context, orgaID, az, projectID string) ([]string, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/storage-class", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var storageClasses []string
	if err := decodeResponse(resp, &storageClasses); err != nil {
		return nil, err
	}

	return storageClasses, nil
}
