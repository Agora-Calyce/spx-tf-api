package spx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// GetOrganization récupère une organisation par son ID
func (c *Client) GetOrganization(ctx context.Context, orgaID string) (*models.Organization, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/organization/%s", orgaID), nil, nil)
	if err != nil {
		return nil, err
	}

	var orga models.Organization
	if err := decodeResponse(resp, &orga); err != nil {
		return nil, err
	}

	return &orga, nil
}

// CreateProject crée un nouveau projet dans une organisation
func (c *Client) CreateProject(ctx context.Context, orgaID string, project *models.APIProject) (*models.APIProject, error) {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/organization/%s/project", orgaID), project, nil)
	if err != nil {
		return nil, err
	}

	var newProject models.APIProject
	if err := decodeResponse(resp, &newProject); err != nil {
		return nil, err
	}

	return &newProject, nil
}
