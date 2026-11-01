package handlers

import (
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// LoadBalancerHandler gère les requêtes liées aux Load Balancers.
type LoadBalancerHandler struct {
	configStore *store.ConfigStore
}

// NewLoadBalancerHandler crée un nouveau LoadBalancerHandler.
func NewLoadBalancerHandler(configStore *store.ConfigStore) *LoadBalancerHandler {
	return &LoadBalancerHandler{
		configStore: configStore,
	}
}

// getSPXClient helper pour obtenir un client configuré à partir du header Authorization
func (h *LoadBalancerHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
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

// CreateLoadBalancer crée un nouveau Load Balancer.
func (h *LoadBalancerHandler) CreateLoadBalancer(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	var req models.LoadBalancerRequest
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

	spxReq := &spxmodels.CreateLoadBalancerBody{}
	spxReq.General.ProductName = req.Name
	spxReq.Spec.VIP = req.VIP
	for _, s := range req.Selectors {
		spxReq.Spec.Selectors = append(spxReq.Spec.Selectors, fmt.Sprintf("%s:%s", s.Key, s.Value))
	}
	spxReq.Spec.Endpoints = req.Endpoints
	for _, p := range req.Ports {
		spxReq.Spec.Ports = append(spxReq.Spec.Ports, spxmodels.LoadBalancerPort{
			Port:       p.Port,
			TargetPort: p.TargetPort,
			Protocol:   p.Protocol,
		})
	}

	resp, err := client.CreateLoadBalancer(c.Request().Context(), orgaID, az, projectID, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	lb := &models.LoadBalancer{
		ID:        resp.EID,
		Name:      req.Name,
		AZ:        az,
		VIP:       req.VIP,
		Selectors: req.Selectors,
		Endpoints: req.Endpoints,
		Ports:     req.Ports,
	}

	return c.JSON(http.StatusCreated, lb)
}

// GetLoadBalancer récupère un Load Balancer par son ID.
func (h *LoadBalancerHandler) GetLoadBalancer(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.GetLoadBalancer(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "Load Balancer non trouvé sur SPX"})
	}

	return c.JSON(http.StatusOK, h.mapLBFromSPX(resp, az))
}

// UpdateLoadBalancer met à jour un Load Balancer.
func (h *LoadBalancerHandler) UpdateLoadBalancer(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	var req models.LoadBalancerRequest
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

	spxReq := &spxmodels.UpdateLoadBalancerBody{}
	spxReq.General.ProductName = req.Name
	spxReq.Spec.VIP = req.VIP
	for _, s := range req.Selectors {
		spxReq.Spec.Selectors = append(spxReq.Spec.Selectors, fmt.Sprintf("%s:%s", s.Key, s.Value))
	}
	spxReq.Spec.Endpoints = req.Endpoints
	for _, p := range req.Ports {
		spxReq.Spec.Ports = append(spxReq.Spec.Ports, spxmodels.LoadBalancerPort{
			Port:       p.Port,
			TargetPort: p.TargetPort,
			Protocol:   p.Protocol,
		})
	}

	err = client.UpdateLoadBalancer(c.Request().Context(), orgaID, az, projectID, id, spxReq)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	respGet, err := client.GetLoadBalancer(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(http.StatusOK, &models.LoadBalancer{
			ID:        id,
			Name:      req.Name,
			AZ:        az,
			VIP:       req.VIP,
			Selectors: req.Selectors,
			Endpoints: req.Endpoints,
			Ports:     req.Ports,
		})
	}

	return c.JSON(http.StatusOK, h.mapLBFromSPX(respGet, az))
}

// DeleteLoadBalancer supprime un Load Balancer.
func (h *LoadBalancerHandler) DeleteLoadBalancer(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteLoadBalancer(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListLoadBalancers liste tous les Load Balancers.
func (h *LoadBalancerHandler) ListLoadBalancers(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	resp, err := client.ListLoadBalancers(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.LoadBalancer, 0, len(resp))
	for _, l := range resp {
		list = append(list, h.mapLBFromSPX(&l, az))
	}

	return c.JSON(http.StatusOK, list)
}

func (h *LoadBalancerHandler) mapLBFromSPX(spxLB *spxmodels.LoadBalancerFullResponse, az string) *models.LoadBalancer {
	lb := &models.LoadBalancer{
		ID:   spxLB.EID,
		Name: spxLB.ProductName,
		AZ:   az,
		VIP:  spxLB.LoadBalancer.Spec.VIP,
	}

	// Mapping des sélecteurs (parsing de "key:value")
	for _, s := range spxLB.LoadBalancer.Spec.Selector {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) == 2 {
			lb.Selectors = append(lb.Selectors, models.LBSelector{
				Key:   parts[0],
				Value: parts[1],
			})
		}
	}

	// Mapping des endpoints
	lb.Endpoints = spxLB.LoadBalancer.Spec.Endpoints

	// Mapping des ports
	for _, p := range spxLB.LoadBalancer.Spec.Ports {
		lb.Ports = append(lb.Ports, models.LoadBalancerPort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort,
			Protocol:   p.Protocol,
		})
	}

	return lb
}
