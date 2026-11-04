package spx

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
)

// GetSSHKey retrieves an SSH key by its EID
func (c *Client) GetSSHKey(ctx context.Context, orgaId, az, projectId, eid string) (*models.SSHKeyFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/ssh/%s", orgaId, az, projectId, eid)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var sshKey models.SSHKeyFullResponse
	if err := decodeResponse(resp, &sshKey); err != nil {
		return nil, err
	}

	return &sshKey, nil
}

// ListSSHKeys retrieves the list of SSH keys for a given project and AZ
func (c *Client) ListSSHKeys(ctx context.Context, orgaId, az, projectId string) ([]models.SSHKeyFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/ssh", orgaId, az, projectId)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var sshKeys []models.SSHKeyFullResponse
	if err := decodeResponse(resp, &sshKeys); err != nil {
		return nil, err
	}

	return sshKeys, nil
}

// CreateSSHKey creates a new SSH key
func (c *Client) CreateSSHKey(ctx context.Context, orgaId, az, projectId string, body models.CreateSSHKeyBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/ssh", orgaId, az, projectId)
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

// DeleteSSHKey deletes an SSH key
func (c *Client) DeleteSSHKey(ctx context.Context, orgaId, az, projectId, eid string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/ssh/%s", orgaId, az, projectId, eid)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
