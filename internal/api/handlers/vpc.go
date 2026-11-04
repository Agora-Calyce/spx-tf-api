package handlers

import (
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

// VPCHandler handles requests related to VPCs.
type VPCHandler struct {
	vpcs        map[string]*models.VPC
	configStore *store.ConfigStore
	mu          sync.RWMutex
}

// NewVPCHandler creates a new VPCHandler.
func NewVPCHandler(configStore *store.ConfigStore) *VPCHandler {
	return &VPCHandler{
		vpcs:        make(map[string]*models.VPC),
		configStore: configStore,
	}
}

// getSPXClient is a helper to get a client configured from the Authorization header
func (h *VPCHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "API endpoint not configured at startup")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	if token == "" || token == authHeader { // If "api-token " was missing or nothing follows it
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token format (expected: api-token {token})")
	}

	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error creating API client")
	}

	return client, nil
}

// CreateVPC creates a new VPC.
// @Summary Create a VPC
// @Description Creates a new VPC resource by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags vpc
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param vpc body models.VPCRequest true "VPC Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 201 {object} models.VPC
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vpcs [post]
func (h *VPCHandler) CreateVPC(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	var req models.VPCRequest
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

	spxReq := &spxmodels.CreateVPCBody{}
	spxReq.General.ProductName = req.Name

	resp, err := client.CreateVPC(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	vpc := &models.VPC{
		ID:           resp.EID,
		Name:         req.Name,
		AZ:           az,
		StaticRoutes: []models.StaticRoute{},
	}

	h.mu.Lock()
	h.vpcs[vpc.ID] = vpc
	h.mu.Unlock()

	return c.JSON(http.StatusCreated, vpc)
}

// GetVPC retrieves a VPC by its ID.
// @Summary Get a VPC
// @Description Retrieves the details of a VPC by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags vpc
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "VPC ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.VPC
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vpcs/{id} [get]
func (h *VPCHandler) GetVPC(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "VPC not found on SPX"})
	}

	vpc := mapVPCFromSPX(resp, az)

	return c.JSON(http.StatusOK, vpc)
}

// UpdateVPC updates an existing VPC.
// @Summary Update a VPC
// @Description Updates the name of a VPC by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags vpc
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "VPC ID (Effective ID)"
// @Param vpc body models.VPCRequest true "VPC Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.VPC
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vpcs/{id} [put]
func (h *VPCHandler) UpdateVPC(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	var req models.VPCRequest
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

	spxReq := &spxmodels.UpdateVPCBody{}
	spxReq.General.ProductName = req.Name
	if req.StaticRoutes != nil {
		spxReq.StaticRoutes = mapStaticRoutesToSPX(*req.StaticRoutes)
	} else {
		// The SPX API replaces all static routes on every update:
		// without static_routes in the request, the current routes are sent back so they are not lost.
		current, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
		if err != nil {
			return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": fmt.Sprintf("VPC not found on SPX: %v", err)})
		}
		spxReq.StaticRoutes = mapStaticRoutesToSPX(mapVPCFromSPX(current, az).StaticRoutes)
	}

	err = client.UpdateVPC(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	// After a successful update, fetch the full object to return it
	respGet, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		// If the GET fails right after, still return a minimal object or an error
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("VPC updated but error while fetching it: %v", err)})
	}

	vpc := mapVPCFromSPX(respGet, az)

	return c.JSON(http.StatusOK, vpc)
}

// DeleteVPC deletes a VPC.
// @Summary Delete a VPC
// @Description Deletes a VPC by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags vpc
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "VPC ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vpcs/{id} [delete]
func (h *VPCHandler) DeleteVPC(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error during deletion: %v", err)})
	}

	h.mu.Lock()
	delete(h.vpcs, id)
	h.mu.Unlock()

	return c.NoContent(http.StatusNoContent)
}

// ListVPCs lists all VPCs.
// @Summary List VPCs
// @Description Returns the list of all VPCs by calling the SPX API. OrgaID, ProjectID and AZ must be provided in the URL.
// @Tags vpc
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} models.VPC
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vpcs [get]
func (h *VPCHandler) ListVPCs(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListVPCs(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	list := make([]*models.VPC, 0, len(resp))
	for _, v := range resp {
		list = append(list, mapVPCFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}

// mapVPCFromSPX converts an SPX response into a VPC (without the routes generated by NAT Gateways)
func mapVPCFromSPX(resp *spxmodels.VPCFullResponse, az string) *models.VPC {
	vpc := &models.VPC{
		ID:           resp.EID,
		Name:         resp.ProductName,
		AZ:           az,
		StaticRoutes: []models.StaticRoute{},
	}
	for _, sr := range resp.VPC.Spec.StaticRoutes {
		if sr.AutoGenerated {
			continue
		}
		vpc.StaticRoutes = append(vpc.StaticRoutes, models.StaticRoute{
			Policy:     sr.Policy,
			CIDR:       sr.CIDR,
			NextHopIP:  sr.NextHopIP,
			RouteTable: sr.RouteTable,
		})
	}
	return vpc
}

func mapStaticRoutesToSPX(routes []models.StaticRoute) []spxmodels.StaticRoute {
	res := make([]spxmodels.StaticRoute, 0, len(routes))
	for _, sr := range routes {
		policy := sr.Policy
		if policy == "" {
			policy = "policyDst"
		}
		res = append(res, spxmodels.StaticRoute{
			Policy:     policy,
			CIDR:       sr.CIDR,
			NextHopIP:  sr.NextHopIP,
			RouteTable: sr.RouteTable,
		})
	}
	return res
}
