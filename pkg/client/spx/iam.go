package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// ListGroups lists the groups of an organization
func (c *Client) ListGroups(ctx context.Context, orgaID string) ([]models.APIGroup, error) {
	path := fmt.Sprintf("/v1/organization/%s/iam/group", orgaID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var groups []models.APIGroup
	if err := decodeResponse(resp, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroup retrieves a group by its ID
func (c *Client) GetGroup(ctx context.Context, orgaID, groupID string) (*models.APIGroup, error) {
	path := fmt.Sprintf("/v1/organization/%s/iam/group/%s", orgaID, groupID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var group models.APIGroup
	if err := decodeResponse(resp, &group); err != nil {
		return nil, err
	}

	return &group, nil
}

// ListPermissionSets lists the permission sets available in an organization
func (c *Client) ListPermissionSets(ctx context.Context, orgaID string) ([]string, error) {
	path := fmt.Sprintf("/v1/organization/%s/iam/permissionSets", orgaID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var sets []string
	if err := decodeResponse(resp, &sets); err != nil {
		return nil, err
	}

	return sets, nil
}
