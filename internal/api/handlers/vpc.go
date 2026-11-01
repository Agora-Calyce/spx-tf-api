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

// VPCHandler gère les requêtes liées aux VPC.
type VPCHandler struct {
	vpcs        map[string]*models.VPC
	configStore *store.ConfigStore
	mu          sync.RWMutex
}

// NewVPCHandler crée un nouveau VPCHandler.
func NewVPCHandler(configStore *store.ConfigStore) *VPCHandler {
	return &VPCHandler{
		vpcs:        make(map[string]*models.VPC),
		configStore: configStore,
	}
}

// getSPXClient helper pour obtenir un client configuré à partir du header Authorization
func (h *VPCHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Endpoint API non configuré au démarrage")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Header Authorization manquant")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	if token == "" || token == authHeader { // Si "api-token " n'était pas présent ou rien après
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Format du token invalide (attendu: api-token {token})")
	}

	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Erreur création client API")
	}

	return client, nil
}

// CreateVPC crée un nouveau VPC.
// @Summary Créer un VPC
// @Description Crée une nouvelle ressource VPC en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags vpc
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	var req models.VPCRequest
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

	spxReq := &spxmodels.CreateVPCBody{}
	spxReq.General.ProductName = req.Name

	resp, err := client.CreateVPC(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
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

// GetVPC récupère un VPC par son ID.
// @Summary Récupérer un VPC
// @Description Récupère les détails d'un VPC en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags vpc
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "VPC non trouvé sur SPX"})
	}

	vpc := mapVPCFromSPX(resp, az)

	return c.JSON(http.StatusOK, vpc)
}

// UpdateVPC met à jour un VPC existant.
// @Summary Mettre à jour un VPC
// @Description Met à jour le nom d'un VPC en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags vpc
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	var req models.VPCRequest
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

	spxReq := &spxmodels.UpdateVPCBody{}
	spxReq.General.ProductName = req.Name
	if req.StaticRoutes != nil {
		spxReq.StaticRoutes = mapStaticRoutesToSPX(*req.StaticRoutes)
	} else {
		// L'API SPX remplace toutes les routes statiques à chaque mise à jour :
		// sans static_routes dans la requête, on renvoie les routes actuelles pour ne pas les perdre.
		current, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
		if err != nil {
			return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": fmt.Sprintf("VPC non trouvé sur SPX: %v", err)})
		}
		spxReq.StaticRoutes = mapStaticRoutesToSPX(mapVPCFromSPX(current, az).StaticRoutes)
	}

	err = client.UpdateVPC(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	// Après une mise à jour réussie, on récupère l'objet complet pour le retourner
	respGet, err := client.GetVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		// Si le GET échoue juste après, on retourne quand même un objet minimal ou une erreur
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("VPC mis à jour mais erreur lors de la récupération: %v", err)})
	}

	vpc := mapVPCFromSPX(respGet, az)

	return c.JSON(http.StatusOK, vpc)
}

// DeleteVPC supprime un VPC.
// @Summary Supprimer un VPC
// @Description Supprime un VPC en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags vpc
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteVPC(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
	}

	h.mu.Lock()
	delete(h.vpcs, id)
	h.mu.Unlock()

	return c.NoContent(http.StatusNoContent)
}

// ListVPCs liste tous les VPCs.
// @Summary Lister les VPCs
// @Description Renvoie la liste de tous les VPCs en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags vpc
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListVPCs(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.VPC, 0, len(resp))
	for _, v := range resp {
		list = append(list, mapVPCFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}

// mapVPCFromSPX convertit une réponse SPX en VPC (sans les routes générées par les NAT Gateways)
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
