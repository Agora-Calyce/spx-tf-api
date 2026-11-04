package handlers_test

import (
	"encoding/json"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSubnetHandler_CreateSubnet_Retry(t *testing.T) {
	calls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			// Successful creation
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"eid": "subnet-123"}`)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/subnet/subnet-123") {
			calls++
			w.WriteHeader(http.StatusOK)

			if calls == 1 {
				// No ready status yet
				fmt.Fprint(w, `{
					"eid": "subnet-123",
					"productName": "test-subnet",
					"subnet": {
						"spec": {
							"vpc": "vpc-123",
							"cidrBlock": "10.0.1.0/24",
							"protocol": "IPv4"
						}
					}
				}`)
			} else {
				// Status present on the second call
				fmt.Fprint(w, `{
					"eid": "subnet-123",
					"productName": "test-subnet",
					"subnet": {
						"spec": {
							"vpc": "vpc-123",
							"cidrBlock": "10.0.1.0/24",
							"protocol": "IPv4"
						},
						"status": {
							"v4availableIPrange": "10.0.1.2-10.0.1.254"
						}
					}
				}`)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewSubnetHandler(configStore)
	h.PollingInterval = 10 * time.Millisecond // Short interval for the test

	reqBody := `{"name": "test-subnet", "vpc_id": "vpc-123", "ipv4": "10.0.1.0/24", "protocol": "IPv4"}`
	req := httptest.NewRequest(http.MethodPost, "/orga-1/az1/proj-1/subnets", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga-1", "az1", "proj-1")

	// Execution
	err := h.CreateSubnet(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, 2, calls, "Devrait avoir fait 2 appels GET (polling)")

	var resp models.Subnet
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "subnet-123", resp.ID)
	assert.Equal(t, "10.0.1.0/24", resp.IPv4)
}

func TestSubnetHandler_GetSubnet_Retry(t *testing.T) {
	calls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls++
		w.WriteHeader(http.StatusOK)

		if calls == 1 {
			// Not ready yet
			fmt.Fprint(w, `{
				"eid": "subnet-get",
				"productName": "get-subnet",
				"subnet": {
					"spec": { "vpc": "vpc-1", "cidrBlock": "10.0.2.0/24", "protocol": "IPv4" }
				}
			}`)
		} else {
			// Ready
			fmt.Fprint(w, `{
				"eid": "subnet-get",
				"productName": "get-subnet",
				"subnet": {
					"spec": { "vpc": "vpc-1", "cidrBlock": "10.0.2.0/24", "protocol": "IPv4" },
					"status": { "v4availableIPrange": "10.0.2.2-10.0.2.254" }
				}
			}`)
		}
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewSubnetHandler(configStore)
	h.PollingInterval = 10 * time.Millisecond

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/subnets/subnet-get", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "subnet-get")

	// Execution
	err := h.GetSubnet(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, calls)
}
