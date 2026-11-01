package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSecurityGroupHandler_Mapping(t *testing.T) {
	// Mock SPX API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/org1/api/spx-ctrl/az1/proj1/security-group/sg1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "uuid-1",
			"eid": "sg1",
			"productName": "test-sg",
			"codeAZ": "az1",
			"gitops": "false",
			"productTypeId": "security-group",
			"securityGroup": {
				"description": "A test security group",
				"subnetEIds": ["subnet-a", "subnet-b"],
				"spec": {
					"policyTypes": ["Ingress", "Egress"],
					"podSelector": {
						"matchLabels": {
							"app": "test",
							"superphenix.net/projectID": "123"
						},
						"matchExpressions": [
							{"key": "env", "operator": "In", "values": ["prod"]},
							{"key": "superphenix.net/ignoreNetworkPolicies", "operator": "Exists"}
						]
					},
					"ingress": [
						{
							"ports": [
								{"port": 80, "protocol": "TCP"},
								{"port": 8000, "endPort": 8100, "protocol": "UDP"},
								{"port": "9090", "protocol": "TCP"}
							],
							"from": [
								{"podSelector": {"matchLabels": {"role": "web"}}},
								{"ipBlock": {"cidr": "10.0.0.0/8", "except": ["10.1.0.0/16"]}}
							]
						}
					],
					"egress": []
				}
			}
		}`))
	}))
	defer server.Close()

	configStore := store.NewConfigStore()
	configStore.SetEndpoint(server.URL)
	handler := handlers.NewSecurityGroupHandler(configStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/org1/az1/proj1/security-groups/sg1", nil)
	req.Header.Set("Authorization", "api-token test-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "eid")
	c.SetParamValues("org1", "az1", "proj1", "sg1")

	if assert.NoError(t, handler.GetSecurityGroup(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		var sg models.SecurityGroup
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sg))

		assert.Equal(t, "sg1", sg.ID)
		assert.Equal(t, "test-sg", sg.Name)
		assert.Equal(t, "A test security group", sg.Description)
		assert.Equal(t, []string{"subnet-a", "subnet-b"}, sg.SubnetIDs)

		// Labels système filtrés, format "clé:valeur"
		assert.Equal(t, []string{"app:test"}, sg.Target.MatchLabels)

		// Expressions système filtrées
		if assert.Len(t, sg.Target.MatchExpressions, 1) {
			assert.Equal(t, "env", sg.Target.MatchExpressions[0].Key)
		}

		if assert.Len(t, sg.Ingress, 1) {
			ports := sg.Ingress[0].Ports
			if assert.Len(t, ports, 3) {
				// Port seul : end_port = port
				assert.Equal(t, models.SgPort{Port: 80, EndPort: 80, Protocol: "TCP"}, ports[0])
				assert.Equal(t, models.SgPort{Port: 8000, EndPort: 8100, Protocol: "UDP"}, ports[1])
				// Port Kubernetes sous forme de chaîne (intstr)
				assert.Equal(t, models.SgPort{Port: 9090, EndPort: 9090, Protocol: "TCP"}, ports[2])
			}
			if assert.Len(t, sg.Ingress[0].From, 2) {
				assert.Equal(t, []string{"role:web"}, sg.Ingress[0].From[0].PodSelector.MatchLabels)
				assert.Nil(t, sg.Ingress[0].From[0].IPBlock)
				assert.Nil(t, sg.Ingress[0].From[1].PodSelector)
				assert.Equal(t, "10.0.0.0/8", sg.Ingress[0].From[1].IPBlock.CIDR)
				assert.Equal(t, []string{"10.1.0.0/16"}, sg.Ingress[0].From[1].IPBlock.Except)
			}
		}

		// Egress déclaré dans policyTypes mais sans règle : tout refuser
		if assert.Len(t, sg.Egress, 1) {
			assert.True(t, sg.Egress[0].DenyAll)
		}
	}
}

func TestSecurityGroupHandler_CreateSecurityGroup(t *testing.T) {
	var sentBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			assert.Equal(t, "/org1/api/spx-ctrl/az1/proj1/security-group", r.URL.Path)
			raw, _ := io.ReadAll(r.Body)
			assert.NoError(t, json.Unmarshal(raw, &sentBody))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"eid": "sg-new"}`))
			return
		}
		// GET après la création
		assert.Equal(t, "/org1/api/spx-ctrl/az1/proj1/security-group/sg-new", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"eid": "sg-new",
			"productName": "new-sg",
			"securityGroup": {
				"description": "New one",
				"subnetEIds": ["subnet-a"],
				"spec": {
					"policyTypes": ["Ingress"],
					"podSelector": {"matchLabels": {"app": "test"}},
					"ingress": [{"ports": [{"port": 443, "protocol": "TCP"}]}]
				}
			}
		}`))
	}))
	defer server.Close()

	configStore := store.NewConfigStore()
	configStore.SetEndpoint(server.URL)
	handler := handlers.NewSecurityGroupHandler(configStore)

	// Format envoyé par le provider Terraform : labels "clé:valeur" et ports en chaînes
	reqBody := `{
		"name": "new-sg",
		"description": "New one",
		"target": {"matchLabels": ["app:test"]},
		"ingress": [{"ports": [{"port": "443", "endPort": "", "protocol": "tcp"}], "from": [{"IPBlock": {"CIDR": "0.0.0.0/0"}}]}],
		"subnetIds": ["subnet-a"]
	}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/org1/az1/proj1/security-groups", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "api-token test-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("org1", "az1", "proj1")

	if assert.NoError(t, handler.CreateSecurityGroup(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Corps envoyé à SPX au format attendu par l'API
		general := sentBody["general"].(map[string]interface{})
		assert.Equal(t, "new-sg", general["productName"])
		spec := sentBody["spec"].(map[string]interface{})
		assert.Equal(t, []interface{}{"subnet-a"}, spec["subnetEIds"])
		target := spec["target"].(map[string]interface{})
		assert.Equal(t, []interface{}{map[string]interface{}{"key": "app", "value": "test"}}, target["matchLabels"])
		ingress := spec["ingress"].([]interface{})[0].(map[string]interface{})
		port := ingress["ports"].([]interface{})[0].(map[string]interface{})
		assert.Equal(t, float64(443), port["port"])
		assert.Equal(t, float64(443), port["endPort"])
		assert.Equal(t, "TCP", port["protocol"])
		peer := ingress["from"].([]interface{})[0].(map[string]interface{})
		assert.Equal(t, "0.0.0.0/0", peer["IPBlock"].(map[string]interface{})["CIDR"])

		var sg models.SecurityGroup
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sg))
		assert.Equal(t, "sg-new", sg.ID)
		assert.Equal(t, "new-sg", sg.Name)
		assert.Equal(t, []string{"app:test"}, sg.Target.MatchLabels)
		assert.Equal(t, []string{"subnet-a"}, sg.SubnetIDs)
		assert.Equal(t, models.PortNumber(443), sg.Ingress[0].Ports[0].Port)
	}
}

func TestSecurityGroupHandler_CreateSecurityGroup_TooManySubnets(t *testing.T) {
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("http://unused")
	handler := handlers.NewSecurityGroupHandler(configStore)

	reqBody := `{"name": "sg", "target": {"matchLabels": ["app:test"]}, "subnetIds": ["1","2","3","4","5","6","7","8","9","10","11"]}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/org1/az1/proj1/security-groups", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "api-token test-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("org1", "az1", "proj1")

	err := handler.CreateSecurityGroup(c)
	if assert.Error(t, err) {
		he, ok := err.(*echo.HTTPError)
		if assert.True(t, ok) {
			assert.Equal(t, http.StatusBadRequest, he.Code)
		}
	}
}

func TestSecurityGroupHandler_UpdateSecurityGroup(t *testing.T) {
	var updateCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/org1/api/spx-ctrl/az1/proj1/security-group/sg1", r.URL.Path)
		if r.Method == http.MethodPost {
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"eid": "sg1", "productName": "renamed", "securityGroup": {"spec": {"podSelector": {"matchLabels": {"app": "test"}}}}}`))
	}))
	defer server.Close()

	configStore := store.NewConfigStore()
	configStore.SetEndpoint(server.URL)
	handler := handlers.NewSecurityGroupHandler(configStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/org1/az1/proj1/security-groups/sg1", strings.NewReader(`{"name": "renamed", "target": {"matchLabels": ["app:test"]}}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "api-token test-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "eid")
	c.SetParamValues("org1", "az1", "proj1", "sg1")

	if assert.NoError(t, handler.UpdateSecurityGroup(c)) {
		assert.True(t, updateCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		var sg models.SecurityGroup
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sg))
		assert.Equal(t, "renamed", sg.Name)
		assert.Equal(t, []string{}, sg.SubnetIDs)
		assert.Equal(t, []models.IngressRule{}, sg.Ingress)
	}
}

func TestSecurityGroupHandler_DeleteSecurityGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/org1/api/spx-ctrl/az1/proj1/security-group/sg1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configStore := store.NewConfigStore()
	configStore.SetEndpoint(server.URL)
	handler := handlers.NewSecurityGroupHandler(configStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/org1/az1/proj1/security-groups/sg1", nil)
	req.Header.Set("Authorization", "api-token test-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "eid")
	c.SetParamValues("org1", "az1", "proj1", "sg1")

	if assert.NoError(t, handler.DeleteSecurityGroup(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}
}
