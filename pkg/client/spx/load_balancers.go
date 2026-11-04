package spx

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
)

// ListLoadBalancers lists all Load Balancers of a project in a zone.
func (c *Client) ListLoadBalancers(ctx context.Context, orgaID, az, projectID string) ([]models.LoadBalancerFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/load-balancer", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var lbs []models.LoadBalancerFullResponse
	if err := decodeResponse(resp, &lbs); err != nil {
		return nil, err
	}

	return lbs, nil
}

// GetLoadBalancer retrieves the details of a Load Balancer.
func (c *Client) GetLoadBalancer(ctx context.Context, orgaID, az, projectID, id string) (*models.LoadBalancerFullResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/load-balancer/%s", orgaID, az, projectID, id)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var lb models.LoadBalancerFullResponse
	if err := decodeResponse(resp, &lb); err != nil {
		return nil, err
	}

	return &lb, nil
}

// CreateLoadBalancer creates a new Load Balancer.
func (c *Client) CreateLoadBalancer(ctx context.Context, orgaID, az, projectID string, body *models.CreateLoadBalancerBody) (*models.CreateResponse, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/load-balancer", orgaID, az, projectID)
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

// UpdateLoadBalancer updates a Load Balancer.
func (c *Client) UpdateLoadBalancer(ctx context.Context, orgaID, az, projectID, id string, body *models.UpdateLoadBalancerBody) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/load-balancer/%s", orgaID, az, projectID, id)
	resp, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// DeleteLoadBalancer deletes a Load Balancer.
func (c *Client) DeleteLoadBalancer(ctx context.Context, orgaID, az, projectID, id string) error {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/load-balancer/%s", orgaID, az, projectID, id)
	resp, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
