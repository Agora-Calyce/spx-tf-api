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
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Endpoint API non configuré au démarrage")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Header Authorization manquant")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Erreur création client API")
	}

	return client, nil
}

// GetSSHKey récupère une clé SSH
// @Summary Récupérer une clé SSH
// @Description Récupère les détails d'une clé SSH en appelant l'API SPX.
// @Tags ssh
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "Clé SSH non trouvée sur SPX"})
	}

	return c.JSON(http.StatusOK, mapSSHKeyFromSPX(resp))
}

// ListSSHKeys liste toutes les clés SSH
// @Summary Lister les clés SSH
// @Description Renvoie la liste de toutes les clés SSH en appelant l'API SPX.
// @Tags ssh
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.SSHKey, 0, len(resp))
	for _, s := range resp {
		list = append(list, mapSSHKeyFromSPX(&s))
	}

	return c.JSON(http.StatusOK, list)
}

// CreateSSHKey crée une nouvelle clé SSH
// @Summary Créer une clé SSH
// @Description Crée une nouvelle clé SSH en appelant l'API SPX.
// @Tags ssh
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "corps de requête invalide"})
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	// Récupérer pour avoir l'objet complet
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

// DeleteSSHKey supprime une clé SSH
// @Summary Supprimer une clé SSH
// @Description Supprime une clé SSH en appelant l'API SPX.
// @Tags ssh
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

func mapSSHKeyFromSPX(spxSSH *spxmodels.SSHKeyFullResponse) *models.SSHKey {
	ssh := &models.SSHKey{
		ID:          spxSSH.EID,
		Name:        spxSSH.ProductName,
		Fingerprint: spxSSH.SSH.Status.Fingerprint,
	}

	// Récupération de la clé publique depuis .ssh.data.spxSSHKey (décodage base64 requis)
	if spxSSH.SSH.Data.SpxSshKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(spxSSH.SSH.Data.SpxSshKey)
		if err == nil {
			ssh.PublicKey = string(decoded)
		} else {
			// En cas d'erreur de décodage, on garde la valeur brute par précaution
			ssh.PublicKey = spxSSH.SSH.Data.SpxSshKey
		}
	}

	return ssh
}
