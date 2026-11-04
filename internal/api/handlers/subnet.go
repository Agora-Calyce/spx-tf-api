package handlers

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// SubnetHandler handles requests related to Subnets.
type SubnetHandler struct {
	configStore     *store.ConfigStore
	PollingInterval time.Duration
}

// NewSubnetHandler creates a new SubnetHandler.
func NewSubnetHandler(configStore *store.ConfigStore) *SubnetHandler {
	return &SubnetHandler{
		configStore:     configStore,
		PollingInterval: 5 * time.Second,
	}
}

// getSPXClient is a helper to get a client configured from the Authorization header
func (h *SubnetHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "API endpoint not configured at startup")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	if token == "" || token == authHeader {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token format (expected: api-token {token})")
	}

	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error creating API client")
	}

	return client, nil
}

// mapSubnetFromSPX converts an SPX response into a Subnet model for the proxy
func (h *SubnetHandler) mapSubnetFromSPX(resp *spxmodels.SubnetFullResponse, az string) *models.Subnet {
	s := &models.Subnet{
		ID:   resp.EID,
		Name: resp.ProductName,
		AZ:   az,
	}

	// NAT Gateway handling
	if resp.NatGateway.Spec.LanIp != "" {
		s.NatGateway = true
		s.NatGatewayIp = resp.NatGateway.Spec.LanIp
	} else {
		s.NatGateway = false
	}

	spec := resp.Subnet.Spec
	s.VPCID = spec.VPC
	s.Private = spec.Private
	s.Protocol = spec.Protocol

	// Extract IPv4 and IPv6 from cidrBlock (comma-separated)
	if spec.CIDRBlock != "" {
		parts := strings.Split(spec.CIDRBlock, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, ".") {
				s.IPv4 = part
			} else if strings.Contains(part, ":") {
				s.IPv6 = part
			}
		}
	}

	// Extract dns_server from DHCPV4Options
	dhcpV4 := []string{}
	if spec.DHCPV4Options != nil {
		if opts, ok := spec.DHCPV4Options.([]interface{}); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					dhcpV4 = append(dhcpV4, s)
				}
			}
		} else if opt, ok := spec.DHCPV4Options.(string); ok {
			dhcpV4 = append(dhcpV4, opt)
		}
	}
	for _, opt := range dhcpV4 {
		if strings.HasPrefix(opt, "dns_server=") {
			s.DNSV4 = strings.TrimPrefix(opt, "dns_server=")
			break
		}
	}

	// Extract dns_server from DHCPV6Options
	dhcpV6 := []string{}
	if spec.DHCPV6Options != nil {
		if opts, ok := spec.DHCPV6Options.([]interface{}); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					dhcpV6 = append(dhcpV6, s)
				}
			}
		} else if opt, ok := spec.DHCPV6Options.(string); ok {
			dhcpV6 = append(dhcpV6, opt)
		}
	}
	for _, opt := range dhcpV6 {
		if strings.HasPrefix(opt, "dns_server=") {
			s.DNSV6 = strings.TrimPrefix(opt, "dns_server=")
			break
		}
	}

	return s
}

// CreateSubnet creates a new Subnet.
// @Summary Create a Subnet
// @Description Creates a new Subnet resource by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags subnet
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param subnet body models.SubnetRequest true "Subnet Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 201 {object} models.Subnet
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/subnets [post]
func (h *SubnetHandler) CreateSubnet(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	var req models.SubnetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	spxReq := &spxmodels.CreateSubnetBody{}
	spxReq.General.ProductName = req.Name
	spxReq.General.VPCEID = req.VPCID
	spxReq.NatGateway.Enable = req.NatGateway
	spxReq.Network.IPv4 = req.IPv4
	spxReq.Network.IPv6 = req.IPv6
	spxReq.Network.Private = req.Private
	spxReq.Network.Protocol = req.Protocol
	spxReq.Network.DNSV4 = req.DNSV4
	spxReq.Network.DNSV6 = req.DNSV6

	resp, err := client.CreateSubnet(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	// Fetch the created object to get the full info and wait until it is ready
	fullResp, err := h.fetchSubnetFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, resp.EID)
	if err != nil {
		// On timeout, still return the created object without waiting
		return c.JSON(http.StatusCreated, &models.Subnet{ID: resp.EID, Name: req.Name, AZ: az})
	}

	return c.JSON(http.StatusCreated, h.mapSubnetFromSPX(fullResp, az))
}

func (h *SubnetHandler) fetchSubnetFromSPXWithRetry(ctx context.Context, client *spx.Client, orgaID, az, projectID, id string) (*spxmodels.SubnetFullResponse, error) {
	maxRetries := 24 // 2 minutes (24 * 5s)
	interval := h.PollingInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := client.GetSubnet(ctx, orgaID, az, projectID, id)
		if err != nil {
			return nil, err
		}

		ready := false
		if resp.Subnet.Status != nil {
			if statusMap, ok := resp.Subnet.Status.(map[string]interface{}); ok {
				_, v4ok := statusMap["v4availableIPrange"]
				_, v6ok := statusMap["v6availableIPrange"]
				if v4ok || v6ok {
					ready = true
				}
			}
		}

		if ready {
			return resp, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			// Continue
		}
	}

	return nil, fmt.Errorf("timeout: the subnet did not become operational after 2 minutes")
}

// GetSubnet retrieves a Subnet by its ID.
// @Summary Get a Subnet
// @Description Retrieves the details of a Subnet by calling the SPX API.
// @Tags subnet
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "Subnet ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Subnet
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/subnets/{id} [get]
func (h *SubnetHandler) GetSubnet(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := h.fetchSubnetFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "Subnet not found on SPX or not operational"})
	}

	return c.JSON(http.StatusOK, h.mapSubnetFromSPX(resp, az))
}

// UpdateSubnet updates an existing Subnet.
// @Summary Update a Subnet
// @Description Updates a Subnet by calling the SPX API.
// @Tags subnet
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "Subnet ID (Effective ID)"
// @Param subnet body models.SubnetRequest true "Subnet Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Subnet
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/subnets/{id} [put]
func (h *SubnetHandler) UpdateSubnet(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	var req models.SubnetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	spxReq := &spxmodels.UpdateSubnetBody{}
	spxReq.General.ProductName = req.Name
	spxReq.NatGateway.Enable = req.NatGateway
	spxReq.Network.DNSV4 = req.DNSV4
	spxReq.Network.DNSV6 = req.DNSV6
	spxReq.Network.Private = req.Private

	err = client.UpdateSubnet(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	respGet, err := h.fetchSubnetFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Subnet updated but error while fetching it or not operational: %v", err)})
	}

	return c.JSON(http.StatusOK, h.mapSubnetFromSPX(respGet, az))
}

// DeleteSubnet deletes a Subnet and waits for its actual deletion.
// @Summary Delete a Subnet
// @Description Deletes a Subnet by calling the SPX API and waits until it is actually gone.
// @Tags subnet
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "Subnet ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/subnets/{id} [delete]
func (h *SubnetHandler) DeleteSubnet(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteSubnet(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error during deletion: %v", err)})
	}

	// Poll to wait for the actual deletion
	maxRetries := 60 // 60 * 5s = 5 minutes max
	interval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		_, err := client.GetSubnet(c.Request().Context(), orgaID, az, projectID, id)
		if err != nil {
			// Assume an error here (probably 404) means the object is deleted
			return c.NoContent(http.StatusNoContent)
		}

		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(interval):
			// Continue the loop
		}
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "timeout: the subnet was not deleted after 5 minutes"})
}

// ListSubnets lists all Subnets.
// @Summary List Subnets
// @Description Returns the list of all Subnets by calling the SPX API.
// @Tags subnet
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} models.Subnet
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/subnets [get]
func (h *SubnetHandler) ListSubnets(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListSubnets(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	list := make([]*models.Subnet, 0, len(resp))
	for _, v := range resp {
		list = append(list, h.mapSubnetFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}
