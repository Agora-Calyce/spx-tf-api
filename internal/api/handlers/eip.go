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

// EipHandler handles requests related to EIPs.
type EipHandler struct {
	configStore     *store.ConfigStore
	PollingInterval time.Duration
}

// NewEipHandler creates a new EipHandler.
func NewEipHandler(configStore *store.ConfigStore) *EipHandler {
	return &EipHandler{
		configStore:     configStore,
		PollingInterval: 5 * time.Second,
	}
}

// getSPXClient is a helper to get a client configured from the Authorization header
func (h *EipHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
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

// mapEipFromSPX converts an SPX response into an Eip model for the proxy
func (h *EipHandler) mapEipFromSPX(r *spxmodels.EipFullResponse, az string) *models.Eip {
	m := &models.Eip{
		ID:   r.EID,
		Name: r.ProductName,
		AZ:   az,
	}

	if r.Eip != nil {
		if eipMap, ok := r.Eip.(map[string]interface{}); ok {
			if spec, ok := eipMap["spec"].(map[string]interface{}); ok {
				// Get the SubnetID from natGwDp (user request)
				if natGwDp, ok := spec["natGwDp"].(string); ok {
					m.SubnetEId = natGwDp
				}

				// Fallback to subnetEid if natGwDp is missing
				if m.SubnetEId == "" {
					if subnetEId, ok := spec["subnetEid"].(string); ok {
						m.SubnetEId = subnetEId
					}
				}

				// Get InternalIP (legacy fallback)
				if internalIP, ok := spec["internalIP"].(string); ok {
					m.InternalIP = internalIP
				} else if internalIP, ok := spec["internalIp"].(string); ok {
					m.InternalIP = internalIP
				}
			}

			// Status fields
			if status, ok := eipMap["status"].(map[string]interface{}); ok {
				if r.ProductTypeID == "eip" {
					if ip, ok := status["ip"].(string); ok {
						m.ExternalIP = ip
					}
					// nat mapping (fip, snat, dnat or combinations)
					if nat, ok := status["nat"].(string); ok {
						m.Nat = nat
					}
				} else {
					// Legacy fallback to externalIP in status
					if externalIP, ok := status["externalIP"].(string); ok {
						m.ExternalIP = externalIP
					}
				}
			}
		}
	}

	// Get internalIp and externalIP from the fip field (takes priority)
	if r.Fip != nil {
		if fipMap, ok := r.Fip.(map[string]interface{}); ok {
			// Get internalIp from spec (new request)
			if spec, ok := fipMap["spec"].(map[string]interface{}); ok {
				if internalIP, ok := spec["internalIp"].(string); ok && internalIP != "" {
					m.InternalIP = internalIP
				} else if internalIP, ok := spec["internalIP"].(string); ok && internalIP != "" {
					m.InternalIP = internalIP
				}
			}

			// Get from status (fallback and externalIP)
			if status, ok := fipMap["status"].(map[string]interface{}); ok {
				// Fallback if not found in spec
				if m.InternalIP == "" {
					if internalIP, ok := status["internalIp"].(string); ok && internalIP != "" {
						m.InternalIP = internalIP
					} else if internalIP, ok := status["internalIP"].(string); ok && internalIP != "" {
						m.InternalIP = internalIP
					}
				}

				if externalIP, ok := status["externalIP"].(string); ok && externalIP != "" {
					m.ExternalIP = externalIP
				}
			}
		}
	}

	// Get the SNAT (array of CIDRs from .snat[].spec.internalCIDR)
	if r.SNat != nil {
		if snatList, ok := r.SNat.([]interface{}); ok {
			for _, item := range snatList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if spec, ok := itemMap["spec"].(map[string]interface{}); ok {
						if cidr, ok := spec["internalCIDR"].(string); ok {
							m.SNAT = append(m.SNAT, cidr)
						}
					}
				}
			}
		} else if snatMap, ok := r.SNat.(map[string]interface{}); ok {
			// Legacy format fallback
			if spec, ok := snatMap["spec"].(map[string]interface{}); ok {
				if snatList, ok := spec["snat"].([]interface{}); ok {
					for _, s := range snatList {
						if sStr, ok := s.(string); ok {
							m.SNAT = append(m.SNAT, sStr)
						}
					}
				}
			}
		}
	}

	// Get the DNAT (array of objects from .dnat[].spec)
	if r.DNat != nil {
		if dnatList, ok := r.DNat.([]interface{}); ok {
			for _, item := range dnatList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if spec, ok := itemMap["spec"].(map[string]interface{}); ok {
						dnat := models.EipDNAT{
							ExternalPort: fmt.Sprintf("%v", spec["externalPort"]),
							InternalIP:   fmt.Sprintf("%v", spec["internalIp"]),
							InternalPort: fmt.Sprintf("%v", spec["internalPort"]),
							Protocol:     strings.ToLower(fmt.Sprintf("%v", spec["protocol"])),
						}
						// Fallback to internalIP (uppercase P)
						if (spec["internalIp"] == nil || spec["internalIp"] == "") && spec["internalIP"] != nil {
							dnat.InternalIP = fmt.Sprintf("%v", spec["internalIP"])
						}
						m.DNAT = append(m.DNAT, dnat)
					}
				}
			}
		} else if dnatMap, ok := r.DNat.(map[string]interface{}); ok {
			// Legacy format fallback
			if spec, ok := dnatMap["spec"].(map[string]interface{}); ok {
				if dnatList, ok := spec["dnat"].([]interface{}); ok {
					for _, d := range dnatList {
						if dMap, ok := d.(map[string]interface{}); ok {
							dnat := models.EipDNAT{
								ExternalPort: fmt.Sprintf("%v", dMap["externalPort"]),
								InternalIP:   fmt.Sprintf("%v", dMap["internalIP"]),
								InternalPort: fmt.Sprintf("%v", dMap["internalPort"]),
								Protocol:     strings.ToLower(fmt.Sprintf("%v", dMap["protocol"])),
							}
							m.DNAT = append(m.DNAT, dnat)
						}
					}
				}
			}
		}
	}

	return m
}

// CreateEip creates a new EIP.
// @Summary Create an EIP
// @Description Creates a new EIP resource by calling the SPX API.
// @Tags eip
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param eip body models.EipRequest true "EIP Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 201 {object} models.Eip
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/eips [post]
func (h *EipHandler) CreateEip(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	if orgaID == "" || az == "" || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az and project_id are required in the URL"})
	}

	var req models.EipRequest
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

	spxReq := &spxmodels.CreateEipBody{}
	spxReq.General.ProductName = req.Name
	spxReq.General.SubnetEId = req.SubnetEId
	spxReq.Spec.InternalIP = req.InternalIP
	spxReq.Spec.SNAT = req.SNAT
	for _, d := range req.DNAT {
		spxReq.Spec.DNAT = append(spxReq.Spec.DNAT, spxmodels.EipDNATBody{
			ExternalPort: d.ExternalPort,
			InternalIP:   d.InternalIP,
			InternalPort: d.InternalPort,
			Protocol:     strings.ToLower(d.Protocol),
		})
	}

	resp, err := client.CreateEip(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	fullResp, err := h.fetchEipFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, resp.EID, false)
	if err != nil {
		return c.JSON(http.StatusCreated, &models.Eip{
			ID:         resp.EID,
			Name:       req.Name,
			AZ:         az,
			SubnetEId:  req.SubnetEId,
			InternalIP: req.InternalIP,
			SNAT:       req.SNAT,
			DNAT:       req.DNAT,
		})
	}

	return c.JSON(http.StatusCreated, h.mapEipFromSPX(fullResp, az))
}

func (h *EipHandler) fetchEipFromSPXWithRetry(ctx context.Context, client *spx.Client, orgaID, az, projectID, id string, skipFirstReady bool) (*spxmodels.EipFullResponse, error) {
	maxRetries := 24 // 24 * interval
	interval := h.PollingInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := client.GetEip(ctx, orgaID, az, projectID, id)
		if err != nil {
			return nil, err
		}

		if resp.ProductTypeID == "eip" {
			ready := false
			if resp.Eip != nil {
				if eipMap, ok := resp.Eip.(map[string]interface{}); ok {
					if status, ok := eipMap["status"].(map[string]interface{}); ok {
						if isReady, ok := status["ready"].(bool); ok && isReady {
							ready = true
						}
					}
				}
			}

			if ready {
				// If asked to ignore the first ready state (update case)
				if i == 0 && skipFirstReady {
					// Continue the loop to wait for the next cycle
				} else {
					return resp, nil
				}
			}
			// Not ready yet or first ready ignored, keep waiting
		} else {
			// Not an "eip" type, no specific retry requested
			return resp, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			// Continue the loop
		}
	}

	return nil, fmt.Errorf("timeout: EIP data did not become available after 2 minutes")
}

// GetEip retrieves an EIP by its ID.
// @Summary Get an EIP
// @Description Retrieves the details of an EIP by calling the SPX API.
// @Tags eip
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "EIP ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Eip
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/eips/{id} [get]
func (h *EipHandler) GetEip(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := h.fetchEipFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id, false)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "EIP not found on SPX"})
	}

	return c.JSON(http.StatusOK, h.mapEipFromSPX(resp, az))
}

// UpdateEip updates an existing EIP.
// @Summary Update an EIP
// @Description Updates an EIP by calling the SPX API.
// @Tags eip
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "EIP ID (Effective ID)"
// @Param eip body models.EipRequest true "EIP Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Eip
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/eips/{id} [put]
func (h *EipHandler) UpdateEip(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	var req models.EipRequest
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

	spxReq := &spxmodels.UpdateEipBody{}
	spxReq.General.ProductName = req.Name
	spxReq.Spec.InternalIP = req.InternalIP
	spxReq.Spec.SNAT = req.SNAT
	for _, d := range req.DNAT {
		spxReq.Spec.DNAT = append(spxReq.Spec.DNAT, spxmodels.EipDNATBody{
			ExternalPort: d.ExternalPort,
			InternalIP:   d.InternalIP,
			InternalPort: d.InternalPort,
			Protocol:     strings.ToLower(d.Protocol),
		})
	}

	err = client.UpdateEip(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	respGet, err := h.fetchEipFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id, true)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("EIP updated but error while fetching it: %v", err)})
	}

	return c.JSON(http.StatusOK, h.mapEipFromSPX(respGet, az))
}

// DeleteEip deletes an EIP and waits for its actual deletion.
// @Summary Delete an EIP
// @Description Deletes an EIP by calling the SPX API and waits until it is actually gone.
// @Tags eip
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "EIP ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/eips/{id} [delete]
func (h *EipHandler) DeleteEip(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteEip(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		// Single retry after 1s if the first delete fails (user request)
		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(1 * time.Second):
			err = client.DeleteEip(c.Request().Context(), orgaID, az, projectID, id)
			if err != nil {
				return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error during deletion: %v", err)})
			}
		}
	}

	// Poll to wait for the actual deletion
	maxRetries := 60 // 60 * interval
	interval := h.PollingInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for i := 0; i < maxRetries; i++ {
		_, err := client.GetEip(c.Request().Context(), orgaID, az, projectID, id)
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

	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "timeout: the EIP was not deleted after 5 minutes"})
}

// ListEips lists all EIPs.
// @Summary List EIPs
// @Description Returns the list of all EIPs by calling the SPX API.
// @Tags eip
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} models.Eip
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/eips [get]
func (h *EipHandler) ListEips(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListEips(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	list := make([]*models.Eip, 0, len(resp))
	for _, v := range resp {
		list = append(list, h.mapEipFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}
