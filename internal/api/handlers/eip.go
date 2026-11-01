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

// EipHandler gère les requêtes liées aux EIP.
type EipHandler struct {
	configStore     *store.ConfigStore
	PollingInterval time.Duration
}

// NewEipHandler crée un nouveau EipHandler.
func NewEipHandler(configStore *store.ConfigStore) *EipHandler {
	return &EipHandler{
		configStore:     configStore,
		PollingInterval: 5 * time.Second,
	}
}

// getSPXClient helper pour obtenir un client configuré à partir du header Authorization
func (h *EipHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
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

// mapEipFromSPX convertit une réponse SPX en modèle Eip pour le proxy
func (h *EipHandler) mapEipFromSPX(r *spxmodels.EipFullResponse, az string) *models.Eip {
	m := &models.Eip{
		ID:   r.EID,
		Name: r.ProductName,
		AZ:   az,
	}

	if r.Eip != nil {
		if eipMap, ok := r.Eip.(map[string]interface{}); ok {
			if spec, ok := eipMap["spec"].(map[string]interface{}); ok {
				// Récupération du SubnetID depuis natGwDp (demande utilisateur)
				if natGwDp, ok := spec["natGwDp"].(string); ok {
					m.SubnetEId = natGwDp
				}

				// Fallback sur subnetEid si natGwDp est absent
				if m.SubnetEId == "" {
					if subnetEId, ok := spec["subnetEid"].(string); ok {
						m.SubnetEId = subnetEId
					}
				}

				// Récupération de l'InternalIP (fallback historique)
				if internalIP, ok := spec["internalIP"].(string); ok {
					m.InternalIP = internalIP
				} else if internalIP, ok := spec["internalIp"].(string); ok {
					m.InternalIP = internalIP
				}
			}

			// Champs de statut
			if status, ok := eipMap["status"].(map[string]interface{}); ok {
				if r.ProductTypeID == "eip" {
					if ip, ok := status["ip"].(string); ok {
						m.ExternalIP = ip
					}
					// Mapping de nat (fip, snat, dnat ou combinaisons)
					if nat, ok := status["nat"].(string); ok {
						m.Nat = nat
					}
				} else {
					// Fallback historique sur externalIP dans status
					if externalIP, ok := status["externalIP"].(string); ok {
						m.ExternalIP = externalIP
					}
				}
			}
		}
	}

	// Récupération de l'internalIp et externalIP depuis le champ fip (prioritaire)
	if r.Fip != nil {
		if fipMap, ok := r.Fip.(map[string]interface{}); ok {
			// Récupération de l'internalIp depuis spec (Nouvelle demande)
			if spec, ok := fipMap["spec"].(map[string]interface{}); ok {
				if internalIP, ok := spec["internalIp"].(string); ok && internalIP != "" {
					m.InternalIP = internalIP
				} else if internalIP, ok := spec["internalIP"].(string); ok && internalIP != "" {
					m.InternalIP = internalIP
				}
			}

			// Récupération depuis status (Fallback et externalIP)
			if status, ok := fipMap["status"].(map[string]interface{}); ok {
				// Fallback si non trouvé dans spec
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

	// Récupération du SNAT (Tableau de CIDRs depuis .snat[].spec.internalCIDR)
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
			// Fallback ancien format
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

	// Récupération du DNAT (Tableau d'objets depuis .dnat[].spec)
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
						// Fallback sur internalIP (majuscule P)
						if (spec["internalIp"] == nil || spec["internalIp"] == "") && spec["internalIP"] != nil {
							dnat.InternalIP = fmt.Sprintf("%v", spec["internalIP"])
						}
						m.DNAT = append(m.DNAT, dnat)
					}
				}
			}
		} else if dnatMap, ok := r.DNat.(map[string]interface{}); ok {
			// Fallback ancien format
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

// CreateEip crée un nouveau EIP.
// @Summary Créer un EIP
// @Description Crée une nouvelle ressource EIP en appelant l'API SPX.
// @Tags eip
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "orga_id, az et project_id sont requis dans l'URL"})
	}

	var req models.EipRequest
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
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
	maxRetries := 24 // 24 * intervalle
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
				// Si on nous demande d'ignorer le premier état ready (cas de l'update)
				if i == 0 && skipFirstReady {
					// On continue la boucle pour attendre le prochain cycle
				} else {
					return resp, nil
				}
			}
			// Pas encore prêt ou premier ready ignoré, on attend
		} else {
			// Pas un type "eip", pas de retry spécifique demandé
			return resp, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			// Continuer la boucle
		}
	}

	return nil, fmt.Errorf("timeout : les données EIP ne sont pas devenues disponibles après 2 minutes")
}

// GetEip récupère un EIP par son ID.
// @Summary Récupérer un EIP
// @Description Récupère les détails d'un EIP en appelant l'API SPX.
// @Tags eip
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusNotFound), map[string]string{"error": "EIP non trouvé sur SPX"})
	}

	return c.JSON(http.StatusOK, h.mapEipFromSPX(resp, az))
}

// UpdateEip met à jour un EIP existant.
// @Summary Mettre à jour un EIP
// @Description Met à jour un EIP en appelant l'API SPX.
// @Tags eip
// @Accept json
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "corps de requête invalide"})
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	respGet, err := h.fetchEipFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id, true)
	if err != nil {
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("EIP mis à jour mais erreur lors de la récupération: %v", err)})
	}

	return c.JSON(http.StatusOK, h.mapEipFromSPX(respGet, az))
}

// DeleteEip supprime un EIP et attend sa suppression effective.
// @Summary Supprimer un EIP
// @Description Supprime un EIP en appelant l'API SPX et attend sa disparition réelle.
// @Tags eip
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		// Retry unique après 1s si le premier delete échoue (demande utilisateur)
		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(1 * time.Second):
			err = client.DeleteEip(c.Request().Context(), orgaID, az, projectID, id)
			if err != nil {
				return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX lors de la suppression: %v", err)})
			}
		}
	}

	// Polling pour attendre la suppression réelle
	maxRetries := 60 // 60 * intervalle
	interval := h.PollingInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for i := 0; i < maxRetries; i++ {
		_, err := client.GetEip(c.Request().Context(), orgaID, az, projectID, id)
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

	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Timeout : l'EIP n'a pas été supprimé après 5 minutes"})
}

// ListEips liste tous les EIPs.
// @Summary Lister les EIPs
// @Description Renvoie la liste de tous les EIPs en appelant l'API SPX.
// @Tags eip
// @Produce json
// @Param orga_id path string true "ID de l'organisation"
// @Param az path string true "Zone de disponibilité"
// @Param project_id path string true "ID du projet"
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
		return c.JSON(spxStatus(err, http.StatusInternalServerError), map[string]string{"error": fmt.Sprintf("Erreur API SPX: %v", err)})
	}

	list := make([]*models.Eip, 0, len(resp))
	for _, v := range resp {
		list = append(list, h.mapEipFromSPX(&v, az))
	}

	return c.JSON(http.StatusOK, list)
}
