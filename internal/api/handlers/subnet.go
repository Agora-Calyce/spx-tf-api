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

// SubnetHandler gère les requêtes liées aux Subnets.
type SubnetHandler struct {
	configStore     *store.ConfigStore
	PollingInterval time.Duration
}

// NewSubnetHandler crée un nouveau SubnetHandler.
func NewSubnetHandler(configStore *store.ConfigStore) *SubnetHandler {
	return &SubnetHandler{
		configStore:     configStore,
		PollingInterval: 5 * time.Second,
	}
}

// getSPXClient helper pour obtenir un client configuré à partir du header Authorization
func (h *SubnetHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
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

// mapSubnetFromSPX convertit une réponse SPX en modèle Subnet pour le proxy
func (h *SubnetHandler) mapSubnetFromSPX(resp *spxmodels.SubnetFullResponse, az string) *models.Subnet {
	s := &models.Subnet{
		ID:   resp.EID,
		Name: resp.ProductName,
		AZ:   az,
	}

	// Gestion de la NAT Gateway
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

	// Extraire IPv4 et IPv6 depuis cidrBlock (séparés par une virgule)
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

	// Extraire dns_server depuis DHCPV4Options
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

	// Extraire dns_server depuis DHCPV6Options
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

// CreateSubnet crée un nouveau Subnet.
// @Summary Créer un Subnet
// @Description Crée une nouvelle ressource Subnet en appelant l'API SPX. OrgaID, ProjectID et AZ doivent être fournis dans l'URL.
// @Tags subnet
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	var req models.SubnetRequest
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	// Récupérer l'objet créé pour avoir les infos complètes et attendre qu'il soit prêt
	fullResp, err := h.fetchSubnetFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, resp.EID)
	if err != nil {
		// En cas de timeout, on renvoie quand même l'objet créé mais sans attendre
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
			// Continuer
		}
	}

	return nil, fmt.Errorf("timeout : le subnet n'est pas devenu opérationnel après 2 minutes")
}

// GetSubnet récupère un Subnet par son ID.
// @Summary Récupérer un Subnet
// @Description Récupère les détails d'un Subnet en appelant l'API SPX.
// @Tags subnet
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "Subnet non trouvé sur SPX ou non opérationnel"})
	}

	return c.JSON(http.StatusOK, h.mapSubnetFromSPX(resp, az))
}

// UpdateSubnet met à jour un Subnet existant.
// @Summary Mettre à jour un Subnet
// @Description Met à jour un Subnet en appelant l'API SPX.
// @Tags subnet
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "corps de requête invalide"})
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	respGet, err := h.fetchSubnetFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Subnet mis à jour mais erreur lors de la récupération ou non opérationnel: %v", err)})
	}

	return c.JSON(http.StatusOK, h.mapSubnetFromSPX(respGet, az))
}

// DeleteSubnet supprime un Subnet et attend sa suppression effective.
// @Summary Supprimer un Subnet
// @Description Supprime un Subnet en appelant l'API SPX et attend sa disparition réelle.
// @Tags subnet
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
	}

	// Polling pour attendre la suppression réelle
	maxRetries := 60 // 60 * 5s = 5 minutes max
	interval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		_, err := client.GetSubnet(c.Request().Context(), orgaID, az, projectID, id)
		if err != nil {
			// On suppose qu'une erreur ici (probablement 404) signifie que l'objet est supprimé
			return c.NoContent(http.StatusNoContent)
		}

		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(interval):
			// Continuer la boucle
		}
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Timeout : le subnet n'a pas été supprimé après 5 minutes"})
}

// ListSubnets liste tous les Subnets.
// @Summary Lister les Subnets
// @Description Renvoie la liste de tous les Subnets en appelant l'API SPX.
// @Tags subnet
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.Subnet, 0, len(resp))
	for _, v := range resp {
		list = append(list, h.mapSubnetFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}
