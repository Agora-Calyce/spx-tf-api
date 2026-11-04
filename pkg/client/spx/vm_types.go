package spx

import (
	"context"
	"fmt"
	"net/http"
)

// ListVMTypes retrieves the list of VM types for a given project and AZ
func (c *Client) ListVMTypes(ctx context.Context, orgaID, az, projectID string) ([]string, error) {
	path := fmt.Sprintf("/%s/api/spx-ctrl/%s/%s/vm-type", orgaID, az, projectID)
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var vmTypes []string
	if err := decodeResponse(resp, &vmTypes); err != nil {
		return nil, err
	}

	return vmTypes, nil
}
