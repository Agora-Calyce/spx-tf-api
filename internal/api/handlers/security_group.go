package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"

	"github.com/labstack/echo/v4"
)

type SecurityGroupHandler struct {
	configStore *store.ConfigStore
}

func NewSecurityGroupHandler(configStore *store.ConfigStore) *SecurityGroupHandler {
	return &SecurityGroupHandler{configStore: configStore}
}

// getSPXClient is a helper to get a client configured from the Authorization header
func (h *SecurityGroupHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
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

// ListSecurityGroups lists all security groups
// @Summary List all security groups
// @Tags security-groups
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Success 200 {array} models.SecurityGroup
// @Router /{orga_id}/{az}/{project_id}/security-groups [get]
func (h *SecurityGroupHandler) ListSecurityGroups(c echo.Context) error {
	orgaID := c.Param("orga_id")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	res, err := client.ListSecurityGroups(c.Request().Context(), orgaID, projectID)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to list security groups: %v", err))
	}

	sgs := make([]models.SecurityGroup, len(res))
	for i, r := range res {
		sgs[i] = mapSecurityGroupFromSPX(r)
	}

	return c.JSON(http.StatusOK, sgs)
}

// GetSecurityGroup returns a security group
// @Summary Get a security group
// @Tags security-groups
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Param eid path string true "Security group ID"
// @Success 200 {object} models.SecurityGroup
// @Router /{orga_id}/{az}/{project_id}/security-groups/{eid} [get]
func (h *SecurityGroupHandler) GetSecurityGroup(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	eid := c.Param("eid")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	res, err := client.GetSecurityGroup(c.Request().Context(), orgaID, az, projectID, eid)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusNotFound), fmt.Sprintf("Security group not found: %v", err))
	}

	return c.JSON(http.StatusOK, mapSecurityGroupFromSPX(*res))
}

// CreateSecurityGroup creates a security group
// @Summary Create a security group
// @Tags security-groups
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Param security_group body models.SecurityGroupRequest true "Security group data"
// @Success 201 {object} models.SecurityGroup
// @Router /{orga_id}/{az}/{project_id}/security-groups [post]
func (h *SecurityGroupHandler) CreateSecurityGroup(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	var req models.SecurityGroupRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	body := spxmodels.CreateSecurityGroupBody{}
	body.General.ProductName = req.Name
	body.General.Description = req.Description
	body.Spec = mapSecurityGroupSpecToSPX(req)

	res, err := client.CreateSecurityGroup(c.Request().Context(), orgaID, az, projectID, body)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to create the security group: %v", err))
	}

	// Fetch the created object to get all the info
	created, err := client.GetSecurityGroup(c.Request().Context(), orgaID, az, projectID, res.EID)
	if err != nil {
		// Fallback if the Get fails right after creation
		return c.JSON(http.StatusCreated, models.SecurityGroup{
			ID:          res.EID,
			Name:        req.Name,
			Description: req.Description,
			Target:      req.Target,
			Ingress:     req.Ingress,
			Egress:      req.Egress,
			SubnetIDs:   nonNilStrings(req.SubnetIDs),
		})
	}

	return c.JSON(http.StatusCreated, mapSecurityGroupFromSPX(*created))
}

// UpdateSecurityGroup updates a security group
// @Summary Update a security group
// @Tags security-groups
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Param eid path string true "Security group ID"
// @Param security_group body models.SecurityGroupRequest true "Security group data"
// @Success 200 {object} models.SecurityGroup
// @Router /{orga_id}/{az}/{project_id}/security-groups/{eid} [post]
func (h *SecurityGroupHandler) UpdateSecurityGroup(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	eid := c.Param("eid")

	var req models.SecurityGroupRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	body := spxmodels.UpdateSecurityGroupBody{}
	body.General.ProductName = req.Name
	body.General.Description = req.Description
	body.Spec = mapSecurityGroupSpecToSPX(req)

	err = client.UpdateSecurityGroup(c.Request().Context(), orgaID, az, projectID, eid, body)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to update the security group: %v", err))
	}

	// Fetch the updated object
	updated, err := client.GetSecurityGroup(c.Request().Context(), orgaID, az, projectID, eid)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to fetch the updated security group: %v", err))
	}

	return c.JSON(http.StatusOK, mapSecurityGroupFromSPX(*updated))
}

// DeleteSecurityGroup deletes a security group
// @Summary Delete a security group
// @Tags security-groups
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Param eid path string true "Security group ID"
// @Success 204
// @Router /{orga_id}/{az}/{project_id}/security-groups/{eid} [delete]
func (h *SecurityGroupHandler) DeleteSecurityGroup(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	eid := c.Param("eid")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteSecurityGroup(c.Request().Context(), orgaID, az, projectID, eid)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to delete the security group: %v", err))
	}

	return c.NoContent(http.StatusNoContent)
}

// --- Conversion proxy -> SPX ---

// mapSecurityGroupSpecToSPX converts a validated request to the spec expected by the SPX API
func mapSecurityGroupSpecToSPX(req models.SecurityGroupRequest) spxmodels.SecurityGroupSpecBody {
	spec := spxmodels.SecurityGroupSpecBody{
		Target:     mapLabelSelectorToSPX(req.Target),
		Ingress:    make([]spxmodels.IngressRule, len(req.Ingress)),
		Egress:     make([]spxmodels.EgressRule, len(req.Egress)),
		SubnetEIds: nonNilStrings(req.SubnetIDs),
	}
	for i, r := range req.Ingress {
		spec.Ingress[i] = spxmodels.IngressRule{
			Ports:    mapPortsToSPX(r.Ports),
			From:     mapPeersToSPX(r.From),
			AllowAll: r.AllowAll,
			DenyAll:  r.DenyAll,
		}
	}
	for i, r := range req.Egress {
		spec.Egress[i] = spxmodels.EgressRule{
			Ports:    mapPortsToSPX(r.Ports),
			To:       mapPeersToSPX(r.To),
			AllowAll: r.AllowAll,
			DenyAll:  r.DenyAll,
		}
	}
	return spec
}

func mapLabelSelectorToSPX(ls models.LabelSelector) spxmodels.LabelSelector {
	res := spxmodels.LabelSelector{
		MatchLabels:      []spxmodels.MatchLabel{},
		MatchExpressions: []spxmodels.MatchExpression{},
	}
	for _, l := range ls.MatchLabels {
		// Format already checked by Validate()
		key, value, _ := models.SplitMatchLabel(l)
		res.MatchLabels = append(res.MatchLabels, spxmodels.MatchLabel{Key: key, Value: value})
	}
	for _, exp := range ls.MatchExpressions {
		res.MatchExpressions = append(res.MatchExpressions, spxmodels.MatchExpression{
			Key:      exp.Key,
			Operator: exp.Operator,
			Values:   exp.Values,
		})
	}
	return res
}

func mapPortsToSPX(ports []models.SgPort) []spxmodels.SgPort {
	res := make([]spxmodels.SgPort, len(ports))
	for i, p := range ports {
		res[i] = spxmodels.SgPort{
			Port:     int32(p.Port),
			EndPort:  int32(p.EndPort),
			Protocol: p.Protocol,
		}
	}
	return res
}

func mapPeersToSPX(peers []models.Peer) []spxmodels.Peer {
	res := make([]spxmodels.Peer, len(peers))
	for i, p := range peers {
		peer := spxmodels.Peer{}
		if p.PodSelector != nil {
			peer.PodSelector = mapLabelSelectorToSPX(*p.PodSelector)
		}
		if p.IPBlock != nil {
			peer.IPBlock = spxmodels.IPBlock{CIDR: p.IPBlock.CIDR, Except: p.IPBlock.Except}
		}
		res[i] = peer
	}
	return res
}

// --- Conversion SPX -> proxy ---

func mapSecurityGroupFromSPX(r spxmodels.SecurityGroupFullResponse) models.SecurityGroup {
	sg := models.SecurityGroup{
		ID:          r.EID,
		Name:        r.ProductName,
		Description: r.SecurityGroup.Description,
		Target:      mapLabelSelectorFromSPX(r.SecurityGroup.Spec.PodSelector),
		SubnetIDs:   nonNilStrings(r.SecurityGroup.SubnetEIds),
	}

	// Filtering based on PolicyTypes
	hasIngress := false
	hasEgress := false
	for _, pt := range r.SecurityGroup.Spec.PolicyTypes {
		if strings.ToLower(pt) == "ingress" {
			hasIngress = true
		}
		if strings.ToLower(pt) == "egress" {
			hasEgress = true
		}
	}

	if hasIngress {
		if len(r.SecurityGroup.Spec.Ingress) == 0 {
			sg.Ingress = []models.IngressRule{{DenyAll: true}}
		} else {
			sg.Ingress = mapIngressRulesFromSPX(r.SecurityGroup.Spec.Ingress)
		}
	} else {
		sg.Ingress = []models.IngressRule{}
	}

	if hasEgress {
		if len(r.SecurityGroup.Spec.Egress) == 0 {
			sg.Egress = []models.EgressRule{{DenyAll: true}}
		} else {
			sg.Egress = mapEgressRulesFromSPX(r.SecurityGroup.Spec.Egress)
		}
	} else {
		sg.Egress = []models.EgressRule{}
	}

	return sg
}

func mapLabelSelectorFromSPX(spxSelector spxmodels.SPXLabelSelector) models.LabelSelector {
	res := models.LabelSelector{
		MatchLabels:      []string{},
		MatchExpressions: []models.MatchExpression{},
	}

	for k, v := range spxSelector.MatchLabels {
		if k == "superphenix.net/projectID" {
			continue
		}
		res.MatchLabels = append(res.MatchLabels, fmt.Sprintf("%s:%s", k, v))
	}
	// Stable order (Go maps are unordered)
	sort.Strings(res.MatchLabels)

	for _, exp := range spxSelector.MatchExpressions {
		if exp.Key == "superphenix.net/ignoreNetworkPolicies" {
			continue
		}
		res.MatchExpressions = append(res.MatchExpressions, models.MatchExpression{
			Key:      exp.Key,
			Operator: exp.Operator,
			Values:   exp.Values,
		})
	}

	return res
}

func mapPortsFromSPX(spxPorts []spxmodels.SPXSgPort) []models.SgPort {
	res := make([]models.SgPort, len(spxPorts))
	for i, p := range spxPorts {
		port := models.SgPort{Protocol: p.Protocol}
		if p.Port != nil {
			port.Port = models.PortNumber(p.Port.Int32())
		}
		if p.EndPort != nil {
			port.EndPort = models.PortNumber(*p.EndPort)
		} else {
			port.EndPort = port.Port
		}
		res[i] = port
	}
	return res
}

func mapPeerFromSPX(p spxmodels.SPXPeer) models.Peer {
	ls := mapLabelSelectorFromSPX(p.PodSelector)
	peer := models.Peer{}
	if len(ls.MatchLabels) > 0 || len(ls.MatchExpressions) > 0 {
		peer.PodSelector = &ls
	}
	if p.IPBlock.CIDR != "" {
		peer.IPBlock = &models.IPBlock{CIDR: p.IPBlock.CIDR, Except: p.IPBlock.Except}
	}
	return peer
}

func mapIngressRulesFromSPX(spxRules []spxmodels.SPXIngressRule) []models.IngressRule {
	res := make([]models.IngressRule, len(spxRules))
	for i, r := range spxRules {
		allowAll := r.AllowAll
		// An empty rule ({}) in SPX means "Allow all"
		if len(r.Ports) == 0 && len(r.From) == 0 {
			allowAll = true
		}

		res[i] = models.IngressRule{
			Ports:    mapPortsFromSPX(r.Ports),
			AllowAll: allowAll,
			DenyAll:  r.DenyAll,
			From:     make([]models.Peer, len(r.From)),
		}
		for j, p := range r.From {
			res[i].From[j] = mapPeerFromSPX(p)
		}
	}
	return res
}

func mapEgressRulesFromSPX(spxRules []spxmodels.SPXEgressRule) []models.EgressRule {
	res := make([]models.EgressRule, len(spxRules))
	for i, r := range spxRules {
		allowAll := r.AllowAll
		// An empty rule ({}) in SPX means "Allow all"
		if len(r.Ports) == 0 && len(r.To) == 0 {
			allowAll = true
		}

		res[i] = models.EgressRule{
			Ports:    mapPortsFromSPX(r.Ports),
			AllowAll: allowAll,
			DenyAll:  r.DenyAll,
			To:       make([]models.Peer, len(r.To)),
		}
		for j, p := range r.To {
			res[i].To[j] = mapPeerFromSPX(p)
		}
	}
	return res
}

// nonNilStrings returns an empty slice rather than nil (serialized as [] and not null)
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
