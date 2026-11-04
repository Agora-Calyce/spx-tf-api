package spx

import (
	"context"
	"net/http"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
)

// WhoAmI retrieves the authenticated user's information
func (c *Client) WhoAmI(ctx context.Context) (*models.APIUser, error) {
	resp, err := c.do(ctx, http.MethodGet, "/v1/whoami", nil, nil)
	if err != nil {
		return nil, err
	}

	var user models.APIUser
	if err := decodeResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ListAPITokens lists the user's API tokens
func (c *Client) ListAPITokens(ctx context.Context) ([]models.APIToken, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api-token", nil, nil)
	if err != nil {
		return nil, err
	}

	var tokens []models.APIToken
	if err := decodeResponse(resp, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// CreateAPIToken creates a new API token
func (c *Client) CreateAPIToken(ctx context.Context, body *models.CreateTokenBody) (*models.TokenResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api-token", body, nil)
	if err != nil {
		return nil, err
	}

	var token models.TokenResponse
	if err := decodeResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// DeleteAPIToken deletes an API token
func (c *Client) DeleteAPIToken(ctx context.Context, tokenID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api-token/"+tokenID, nil, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}
