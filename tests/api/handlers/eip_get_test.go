package handlers_test

import (
	"encoding/json"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEipHandler_GetEip_Mapping(t *testing.T) {
	// Simulated SPX response data
	spxResponse := `{
		"eid": "eip-789",
		"productName": "eip-complex",
		"eip": {
			"spec": {
				"internalIP": "10.0.0.10",
				"subnetEid": "subnet-abc"
			}
		},
		"fip": {
			"status": {
				"externalIP": "8.8.8.8"
			}
		},
		"snat": {
			"spec": {
				"snat": ["10.0.0.1", "10.0.0.2"]
			}
		},
		"dnat": {
			"spec": {
				"dnat": [
					{
						"externalPort": "80",
						"internalIP": "10.0.0.1",
						"internalPort": "8080",
						"protocol": "TCP"
					}
				]
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/eip/eip-789")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-789", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-789")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping
	assert.Equal(t, "eip-789", resp.ID)
	assert.Equal(t, "eip-complex", resp.Name)
	assert.Equal(t, "10.0.0.10", resp.InternalIP)
	assert.Equal(t, "8.8.8.8", resp.ExternalIP)
	assert.Equal(t, "subnet-abc", resp.SubnetEId)
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, resp.SNAT)
	assert.Len(t, resp.DNAT, 1)
	assert.Equal(t, "80", resp.DNAT[0].ExternalPort)
	assert.Equal(t, "10.0.0.1", resp.DNAT[0].InternalIP)
	assert.Equal(t, "8080", resp.DNAT[0].InternalPort)
	assert.Equal(t, "tcp", resp.DNAT[0].Protocol)
}

func TestEipHandler_GetEip_NewMapping(t *testing.T) {
	// Simulated SPX response data with the new format for snat and dnat
	spxResponse := `{
		"eid": "eip-789",
		"productName": "eip-complex-v2",
		"eip": {
			"spec": {
				"internalIP": "10.0.0.10",
				"subnetEid": "subnet-abc"
			}
		},
		"snat": [
			{
				"spec": {
					"internalCIDR": "10.11.12.0/24"
				}
			}
		],
		"dnat": [
			{
				"spec": {
					"eip": "spx-ffb93e61-1fc3-566d-b11e-be830a9468af",
					"externalPort": "80",
					"internalIp": "10.11.12.10",
					"internalPort": "80",
					"protocol": "TCP"
				}
			}
		]
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-789", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-789")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping
	assert.Equal(t, "eip-789", resp.ID)
	assert.Equal(t, "10.11.12.0/24", resp.SNAT[0])
	assert.Len(t, resp.DNAT, 1)
	assert.Equal(t, "80", resp.DNAT[0].ExternalPort)
	assert.Equal(t, "10.11.12.10", resp.DNAT[0].InternalIP)
	assert.Equal(t, "80", resp.DNAT[0].InternalPort)
	assert.Equal(t, "tcp", resp.DNAT[0].Protocol)
}

func TestEipHandler_GetEip_FallbackMapping(t *testing.T) {
	// Simulated SPX response data without fip but with eip.status.externalIP
	spxResponse := `{
		"eid": "eip-001",
		"productName": "eip-fallback",
		"eip": {
			"spec": {
				"internalIP": "10.0.0.5",
				"subnetEid": "subnet-123"
			},
			"status": {
				"externalIP": "1.1.1.1"
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-001", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-001")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping (fallback to eip.status)
	assert.Equal(t, "eip-001", resp.ID)
	assert.Equal(t, "1.1.1.1", resp.ExternalIP)
}

func TestEipHandler_GetEip_FipSpecInternalIPMapping(t *testing.T) {
	// Simulated SPX response data with internalIp in fip.spec
	spxResponse := `{
		"eid": "eip-123",
		"productName": "test-eip",
		"fip": {
			"spec": {
				"internalIp": "10.20.30.40"
			},
			"status": {
				"externalIP": "1.2.3.4"
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-123", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-123")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping
	assert.Equal(t, "10.20.30.40", resp.InternalIP, "Must read internal_ip from fip.spec.internalIp")
	assert.Equal(t, "1.2.3.4", resp.ExternalIP)
}

func TestEipHandler_GetEip_ProductTypeEipMapping(t *testing.T) {
	// Simulated SPX response data with productTypeId == "eip"
	spxResponse := `{
		"eid": "eip-123",
		"productName": "mon-eip-type-eip",
		"productTypeId": "eip",
		"eip": {
			"spec": {
				"internalIP": "10.0.0.5",
				"natGwDp": "spx-id-dp",
				"subnetEid": "subnet-123"
			},
			"status": {
				"ready": true,
				"ip": "2.2.2.2",
				"nat": "fip"
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-123", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-123")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping specific to the "eip" type
	assert.Equal(t, "eip-123", resp.ID)
	assert.Equal(t, "10.0.0.5", resp.InternalIP)
	assert.Equal(t, "spx-id-dp", resp.SubnetEId)
	assert.Equal(t, "2.2.2.2", resp.ExternalIP)
	assert.Equal(t, "fip", resp.Nat)
}

func TestEipHandler_GetEip_RetryLogic(t *testing.T) {
	// First call without status, second with status
	calls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if calls == 1 {
			// No ready status on the first call
			fmt.Fprint(w, `{
				"eid": "eip-retry",
				"productName": "eip-retry",
				"productTypeId": "eip",
				"eip": {
					"spec": { "natGwDp": "10.0.0.5" },
					"status": { "ready": false }
				}
			}`)
		} else {
			// Ready status on the second call
			fmt.Fprint(w, `{
				"eid": "eip-retry",
				"productName": "eip-retry",
				"productTypeId": "eip",
				"eip": {
					"spec": { "natGwDp": "10.0.0.5" },
					"status": {
						"ready": true,
						"ip": "3.3.3.3",
						"nat": "snat"
					}
				}
			}`)
		}
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)
	h.PollingInterval = 1 * time.Millisecond // Short interval for the test

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-retry", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-retry")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, calls, "Should have made 2 attempts")

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "3.3.3.3", resp.ExternalIP)
}

func TestEipHandler_GetEip_FipInternalIpMapping(t *testing.T) {
	// Simulated SPX response data with fip.status.internalIp
	spxResponse := `{
		"eid": "eip-fip",
		"productName": "eip-fip",
		"fip": {
			"status": {
				"internalIp": "10.20.30.40",
				"externalIP": "5.5.5.5"
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-fip", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-fip")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping (from fip.status)
	assert.Equal(t, "10.20.30.40", resp.InternalIP)
	assert.Equal(t, "5.5.5.5", resp.ExternalIP)
}

func TestEipHandler_GetEip_MultiNatMapping(t *testing.T) {
	// Simulated SPX response data with multi-value nat
	spxResponse := `{
		"eid": "eip-multi",
		"productTypeId": "eip",
		"eip": {
			"status": {
				"ready": true,
				"nat": "snat,dnat"
			}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/eips/eip-multi", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-multi")

	// Execution
	err := h.GetEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, "snat,dnat", resp.Nat)
}
