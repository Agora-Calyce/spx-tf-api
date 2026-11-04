package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListSecurityGroups returns the list of all security groups
func (c *Client) ListSecurityGroups(ctx context.Context, orgaID, projectID string) ([]models.SecurityGroupFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/security-group", orgaID, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var res []models.SecurityGroupFullResponse
	err = decodeResponse(resp, &res)
	return res, err
}

// ListAZSecurityGroups returns the list of security groups for a specific AZ
func (c *Client) ListAZSecurityGroups(ctx context.Context, orgaID, az, projectID string) ([]models.SecurityGroupFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/security-group", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var res []models.SecurityGroupFullResponse
	err = decodeResponse(resp, &res)
	return res, err
}

// GetSecurityGroup returns a security group by its EID
func (c *Client) GetSecurityGroup(ctx context.Context, orgaID, az, projectID, eid string) (*models.SecurityGroupFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/security-group/%s", orgaID, az, projectID, eid)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var res models.SecurityGroupFullResponse
	err = decodeResponse(resp, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateSecurityGroup creates a new security group
func (c *Client) CreateSecurityGroup(ctx context.Context, orgaID, az, projectID string, body models.CreateSecurityGroupBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/security-group", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return nil, err
	}
	var res models.CreateResponse
	err = decodeResponse(resp, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// UpdateSecurityGroup updates an existing security group
func (c *Client) UpdateSecurityGroup(ctx context.Context, orgaID, az, projectID, eid string, body models.UpdateSecurityGroupBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/security-group/%s", orgaID, az, projectID, eid)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}
	return decodeResponse(resp, nil)
}

// DeleteSecurityGroup deletes a security group
func (c *Client) DeleteSecurityGroup(ctx context.Context, orgaID, az, projectID, eid string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/security-group/%s", orgaID, az, projectID, eid)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	return decodeResponse(resp, nil)
}
