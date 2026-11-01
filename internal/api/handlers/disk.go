package handlers

import (
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

// DiskHandler gère les requêtes liées aux disques.
type DiskHandler struct {
	configStore *store.ConfigStore
	mu          sync.RWMutex
}

// NewDiskHandler crée un nouveau DiskHandler.
func NewDiskHandler(configStore *store.ConfigStore) *DiskHandler {
	return &DiskHandler{
		configStore: configStore,
	}
}

// getSPXClient helper pour obtenir un client configuré à partir du header Authorization
func (h *DiskHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Endpoint API non configuré au démarrage")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Header Authorization manquant")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	if token == "" || token == authHeader {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Format du token invalide (attendu: api-token {token})")
	}

	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Erreur création client API")
	}

	return client, nil
}

// CreateDisk crée un nouveau disque.
// @Summary Créer un disque
// @Description Crée une nouvelle ressource disque en appelant l'API SPX.
// @Tags disk
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param disk body models.DiskRequest true "Disk Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 201 {object} models.Disk
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/disks [post]
func (h *DiskHandler) CreateDisk(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	var req models.DiskRequest
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

	spxReq := &spxmodels.CreateDiskBody{}
	spxReq.General.ProductName = req.Name
	spxReq.General.Storage = fmt.Sprintf("%d", req.Size)
	spxReq.General.StorageClass = req.StorageClass
	spxReq.General.Labels = req.Labels
	if req.Source != nil {
		spxReq.General.Source.Type = req.Source.Type
		spxReq.General.Source.URL = req.Source.URL
		spxReq.General.Source.Clone = req.Source.Clone
		spxReq.General.Source.Snapshot = req.Source.Snapshot
	}

	resp, err := client.CreateDisk(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	disk := &models.Disk{
		ID:           resp.EID,
		Name:         req.Name,
		AZ:           az,
		Size:         req.Size,
		StorageClass: req.StorageClass,
		Source:       req.Source,
		Labels:       req.Labels,
	}

	return c.JSON(http.StatusCreated, disk)
}

// GetDisk récupère un disque par son ID.
// @Summary Récupérer un disque
// @Description Récupère les détails d'un disque en appelant l'API SPX.
// @Tags disk
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param id path string true "Disk ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Disk
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/disks/{id} [get]
func (h *DiskHandler) GetDisk(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.GetDisk(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "Disque non trouvé sur SPX"})
	}

	disk := mapDiskFromSPX(resp, az)

	return c.JSON(http.StatusOK, disk)
}

// UpdateDisk met à jour un disque existant.
// @Summary Mettre à jour un disque
// @Description Met à jour le nom d'un disque en appelant l'API SPX.
// @Tags disk
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param id path string true "Disk ID (Effective ID)"
// @Param disk body models.DiskRequest true "Disk Request Body"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {object} models.Disk
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/disks/{id} [put]
func (h *DiskHandler) UpdateDisk(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	var req models.DiskRequest
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

	spxReq := &spxmodels.UpdateDiskBody{}
	spxReq.General.ProductName = req.Name
	spxReq.General.Storage = fmt.Sprintf("%d", req.Size)
	spxReq.General.Labels = req.Labels

	err = client.UpdateDisk(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	respGet, err := client.GetDisk(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Disque mis à jour mais erreur lors de la récupération: %v", err)})
	}

	disk := mapDiskFromSPX(respGet, az)

	return c.JSON(http.StatusOK, disk)
}

// DeleteDisk supprime un disque.
// @Summary Supprimer un disque
// @Description Supprime un disque en appelant l'API SPX.
// @Tags disk
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param id path string true "Disk ID (Effective ID)"
// @Param Authorization header string true "api-token {token}"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/disks/{id} [delete]
func (h *DiskHandler) DeleteDisk(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteDisk(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListDisks liste tous les disques.
// @Summary Lister les disques
// @Description Renvoie la liste de tous les disques en appelant l'API SPX.
// @Tags disk
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} models.Disk
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/disks [get]
func (h *DiskHandler) ListDisks(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListDisks(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.Disk, 0, len(resp))
	for _, v := range resp {
		list = append(list, mapDiskFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}

// ListStorageClasses liste les StorageClasses.
// @Summary Lister les StorageClasses
// @Description Renvoie la liste de toutes les StorageClasses disponibles pour un projet et une AZ donnés.
// @Tags disk
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
// @Param Authorization header string true "api-token {token}"
// @Success 200 {array} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/storage-classes [get]
func (h *DiskHandler) ListStorageClasses(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListStorageClasses(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	return c.JSON(http.StatusOK, resp)
}

func mapDiskFromSPX(resp *spxmodels.DiskFullResponse, az string) *models.Disk {
	sizeStr := resp.Disk.Spec.Storage.Resources.Requests.Storage
	size, _ := strconv.Atoi(strings.TrimSuffix(sizeStr, "Gi"))

	storageClass := resp.Disk.Spec.Storage.StorageClassName
	if storageClass == "" {
		// L'API SPX ne renvoie plus toujours la StorageClass dans la spec du disque : on la lit sur le PVC
		storageClass = resp.PVC.Spec.StorageClassName
	}
	if parts := strings.Split(storageClass, "."); len(parts) > 0 {
		storageClass = parts[len(parts)-1]
	}

	source := &models.DiskSource{}
	if resp.Disk.Spec.Source.Blank != nil {
		source.Type = "blank"
	} else if resp.Disk.Spec.Source.HTTP != nil {
		source.Type = "http"
		source.URL = resp.Disk.Spec.Source.HTTP.URL
	} else if resp.Disk.Spec.Source.Snapshot != nil {
		source.Type = "snapshot"
		source.Snapshot = resp.Disk.Spec.Source.Snapshot.DiskEID
	} else if resp.Disk.Spec.Source.Clone != nil {
		source.Type = "clone"
		source.Clone = resp.Disk.Spec.Source.Clone.DiskEID
	} else if resp.Disk.Spec.Source.Registry != nil {
		source.Type = "registry"
		source.URL = resp.Disk.Spec.Source.Registry.URL
	}

	return &models.Disk{
		ID:           resp.EID,
		Name:         resp.ProductName,
		AZ:           az,
		Size:         size,
		StorageClass: storageClass,
		Source:       source,
		Labels:       mapDiskLabelsFromSPX(resp),
	}
}

// mapDiskLabelsFromSPX lit les labels utilisateur sur le PVC (mis à jour par l'API SPX),
// à défaut sur le DataVolume (positionnés à la création)
func mapDiskLabelsFromSPX(resp *spxmodels.DiskFullResponse) []string {
	if labels := mapCustomLabelsFromSPX(resp.PVC.Metadata.Labels); len(labels) > 0 {
		return labels
	}
	return mapCustomLabelsFromSPX(resp.Disk.Metadata.Labels)
}
