package handlers

import (
	"encoding/base64"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type SSHKeyHandler struct {
	configStore *store.ConfigStore
}

func NewSSHKeyHandler(configStore *store.ConfigStore) *SSHKeyHandler {
	return &SSHKeyHandler{configStore: configStore}
}

func (h *SSHKeyHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "API endpoint not configured at startup")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error creating API client")
	}

	return client, nil
}

// GetSSHKey retrieves an SSH key
// @Summary Get an SSH key
// @Description Retrieves the details of an SSH key by calling the SPX API.
// @Tags ssh
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "SSH Key ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.SSHKey
// @Failure 404 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/ssh-keys/{id} [get]
func (h *SSHKeyHandler) GetSSHKey(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.GetSSHKey(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "SSH key not found on SPX"})
	}

	return c.JSON(http.StatusOK, mapSSHKeyFromSPX(resp))
}

// ListSSHKeys lists all SSH keys
// @Summary List SSH keys
// @Description Returns the list of all SSH keys by calling the SPX API.
// @Tags ssh
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} models.SSHKey
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/ssh-keys [get]
func (h *SSHKeyHandler) ListSSHKeys(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListSSHKeys(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	list := make([]*models.SSHKey, 0, len(resp))
	for _, s := range resp {
		list = append(list, mapSSHKeyFromSPX(&s))
	}

	return c.JSON(http.StatusOK, list)
}

// CreateSSHKey creates a new SSH key
// @Summary Create an SSH key
// @Description Creates a new SSH key by calling the SPX API.
// @Tags ssh
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param ssh body models.SSHKeyRequest true "SSH Key Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 201 {object} models.SSHKey
// @Router /{orga_id}/{az}/{project_id}/ssh-keys [post]
func (h *SSHKeyHandler) CreateSSHKey(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	var req models.SSHKeyRequest
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

	spxReq := spxmodels.CreateSSHKeyBody{}
	spxReq.General.ProductName = req.Name
	spxReq.General.PublicKey = req.PublicKey

	resp, err := client.CreateSSHKey(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error: %v", err)})
	}

	// Fetch to get the full object
	respGet, err := client.GetSSHKey(c.Request().Context(), orgaID, az, projectID, resp.EID)
	if err != nil || respGet == nil {
		return c.JSON(http.StatusCreated, &models.SSHKey{
			ID:        resp.EID,
			Name:      req.Name,
			PublicKey: req.PublicKey,
		})
	}

	return c.JSON(http.StatusCreated, mapSSHKeyFromSPX(respGet))
}

// DeleteSSHKey deletes an SSH key
// @Summary Delete an SSH key
// @Description Deletes an SSH key by calling the SPX API.
// @Tags ssh
// @Param orga_id path string true "Organization ID"
// @Param az path string true "Availability zone"
// @Param project_id path string true "Project ID"
// @Param id path string true "SSH Key ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 204 "No Content"
// @Router /{orga_id}/{az}/{project_id}/ssh-keys/{id} [delete]
func (h *SSHKeyHandler) DeleteSSHKey(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteSSHKey(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("SPX API error during deletion: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

func mapSSHKeyFromSPX(spxSSH *spxmodels.SSHKeyFullResponse) *models.SSHKey {
	ssh := &models.SSHKey{
		ID:          spxSSH.EID,
		Name:        spxSSH.ProductName,
		Fingerprint: spxSSH.SSH.Status.Fingerprint,
	}

	// Get the public key from .ssh.data.spxSSHKey (base64 decoding required)
	if spxSSH.SSH.Data.SpxSshKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(spxSSH.SSH.Data.SpxSshKey)
		if err == nil {
			ssh.PublicKey = string(decoded)
		} else {
			// On a decoding error, keep the raw value as a precaution
			ssh.PublicKey = spxSSH.SSH.Data.SpxSshKey
		}
	}

	return ssh
}
