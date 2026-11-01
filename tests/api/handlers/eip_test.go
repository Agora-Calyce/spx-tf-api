package handlers_test

import (
	"bytes"
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

func TestEipHandler_AuthorizationHeader(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("https://api.example.com")
	h := handlers.NewEipHandler(configStore)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Header manquant",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Header Authorization manquant",
		},
		{
			name:           "Format invalide (pas de api-token)",
			authHeader:     "token123",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Format du token invalide (attendu: api-token {token})",
		},
		{
			name:           "Format invalide (api-token seul)",
			authHeader:     "api-token ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Format du token invalide (attendu: api-token {token})",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(models.EipRequest{
				Name:       "test-eip",
				SubnetEId:  "subnet-123",
				InternalIP: "10.0.0.1",
			})
			req := httptest.NewRequest(http.MethodPost, "/orga-1/az1/proj-1/eips", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("orga_id", "az", "project_id")
			c.SetParamValues("orga-1", "az1", "proj-1")

			err := h.CreateEip(c)

			// Le middleware Echo peut transformer l'erreur en réponse si c'est un HTTPError
			if err != nil {
				he, ok := err.(*echo.HTTPError)
				if ok {
					assert.Equal(t, tt.expectedStatus, he.Code)
					assert.Contains(t, he.Message.(string), tt.expectedError)
				} else {
					t.Fatalf("Attendu echo.HTTPError, got %v", err)
				}
			} else {
				// Si pas d'erreur retournée par le handler, on check le recorder
				assert.Equal(t, tt.expectedStatus, rec.Code)
				var resp map[string]string
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func TestEipHandler_AllEndpoints_Authorization(t *testing.T) {
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("https://api.example.com")
	h := handlers.NewEipHandler(configStore)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "CreateEip",
			method: http.MethodPost,
			path:   "/o/a/p/eips",
			body: models.EipRequest{
				Name:       "eip",
				SubnetEId:  "subnet-1",
				InternalIP: "10.0.0.1",
			},
		},
		{
			name:   "GetEip",
			method: http.MethodGet,
			path:   "/o/a/p/eips/1",
		},
		{
			name:   "UpdateEip",
			method: http.MethodPut,
			path:   "/o/a/p/eips/1",
			body: models.EipRequest{
				Name:       "eip",
				SubnetEId:  "subnet-1",
				InternalIP: "10.0.0.1",
			},
		},
		{
			name:   "DeleteEip",
			method: http.MethodDelete,
			path:   "/o/a/p/eips/1",
		},
		{
			name:   "ListEips",
			method: http.MethodGet,
			path:   "/o/a/p/eips",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.body != nil {
				json.NewEncoder(&buf).Encode(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, &buf)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			// On ne met pas de header Authorization

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if strings.Contains(tt.path, "/eips/") {
				c.SetParamNames("id", "orga_id", "az", "project_id")
				c.SetParamValues("1", "o", "a", "p")
			} else {
				c.SetParamNames("orga_id", "az", "project_id")
				c.SetParamValues("o", "a", "p")
			}

			var err error
			switch tt.name {
			case "CreateEip":
				err = h.CreateEip(c)
			case "GetEip":
				err = h.GetEip(c)
			case "UpdateEip":
				err = h.UpdateEip(c)
			case "DeleteEip":
				err = h.DeleteEip(c)
			case "ListEips":
				err = h.ListEips(c)
			}

			if err != nil {
				he, ok := err.(*echo.HTTPError)
				if ok {
					assert.Equal(t, http.StatusUnauthorized, he.Code)
					assert.Contains(t, he.Message.(string), "Header Authorization manquant")
				}
			}
		})
	}
}

func TestEipHandler_DeleteEip(t *testing.T) {
	// Setup du mock server
	calls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodDelete {
			assert.Contains(t, r.URL.Path, "/eip/eip-789")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodGet {
			assert.Contains(t, r.URL.Path, "/eip/eip-789")
			// Simuler que l'EIP n'est plus là au premier GET
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)
	h.PollingInterval = 10 * time.Millisecond

	req := httptest.NewRequest(http.MethodDelete, "/orga-1/az1/proj-1/eips/eip-789", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-789")

	// Exécution
	err := h.DeleteEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 2, calls, "Doit faire un DELETE puis un GET")
}

func TestEipHandler_DeleteEip_Retry(t *testing.T) {
	// Setup du mock server
	deleteCalls := 0
	getCalls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCalls++
			if deleteCalls == 1 {
				// Premier appel échoue (500)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error": "Simulated failure"}`)
				return
			}
			// Second appel réussit (204)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodGet {
			getCalls++
			// Simule la disparition de l'objet après le DELETE réussi
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer mockServer.Close()

	// Setup Echo
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)
	// On réduit l'intervalle de polling pour le test
	h.PollingInterval = 10 * time.Millisecond

	req := httptest.NewRequest(http.MethodDelete, "/orga-1/az1/proj-1/eips/eip-123", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-123")

	// Exécution
	err := h.DeleteEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 2, deleteCalls, "Doit avoir appelé DELETE 2 fois (initial + retry)")
	assert.True(t, getCalls >= 1, "Doit avoir appelé GET au moins une fois pour le polling")
}
