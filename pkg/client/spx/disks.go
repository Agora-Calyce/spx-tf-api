package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListDisks retrieves the list of disks for a given project and AZ
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

// CreateDisk creates a new disk
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

// GetDisk retrieves a disk by its ID
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

// UpdateDisk updates an existing disk
func (c *Client) UpdateDisk(ctx context.Context, orgaID, az, projectID, effectiveID string, body *models.UpdateDiskBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteDisk deletes a disk
func (c *Client) DeleteDisk(ctx context.Context, orgaID, az, projectID, effectiveID string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/disk/%s", orgaID, az, projectID, effectiveID)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// ListStorageClasses retrieves the list of StorageClasses for a given project and AZ
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
