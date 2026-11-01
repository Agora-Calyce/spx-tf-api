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
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEipHandler_UpdateEip_Refetch(t *testing.T) {
	// Setup du mock server pour l'API cliente
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Premier appel : Update (POST)
		if callCount == 1 {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/eip/eip-123")
			w.WriteHeader(http.StatusOK)
			return
		}
		// Deuxième appel : Get (GET)
		if callCount == 2 {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/eip/eip-123")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"eid": "eip-123", "productName": "new-name-from-get"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Setup de l'API proxy
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	// Corps de la requête de mise à jour
	reqBody, _ := json.Marshal(models.EipRequest{
		Name:       "new-name-req",
		SubnetEId:  "subnet-123",
		InternalIP: "10.0.0.1",
	})
	req := httptest.NewRequest(http.MethodPut, "/orga-1/az1/proj-1/eips/eip-123", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "eip-123")

	// Exécution
	err := h.UpdateEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, callCount, "L'API cliente doit être appelée deux fois (Update puis Get)")

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "eip-123", resp.ID)
	assert.Equal(t, "new-name-from-get", resp.Name, "Le nom doit provenir du Get final, pas de la requête")
}

func TestEipHandler_CreateEip_Refetch(t *testing.T) {
	// Setup du mock server pour l'API cliente
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Premier appel : Create (POST)
		if callCount == 1 {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/eip")
			assert.NotContains(t, r.URL.Path, "/eip/") // Pas d'ID dans le path pour la création
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"eid": "eip-456"}`)
			return
		}
		// Deuxième appel : Get (GET)
		if callCount == 2 {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/eip/eip-456")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"eid": "eip-456", "productName": "full-name-from-get"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Setup de l'API proxy
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewEipHandler(configStore)

	// Corps de la requête de création
	reqBody, _ := json.Marshal(models.EipRequest{
		Name:       "name-req",
		SubnetEId:  "subnet-123",
		InternalIP: "10.0.0.1",
	})
	req := httptest.NewRequest(http.MethodPost, "/orga-1/az1/proj-1/eips", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga-1", "az1", "proj-1")

	// Exécution
	err := h.CreateEip(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, 2, callCount, "L'API cliente doit être appelée deux fois (Create puis Get)")

	var resp models.Eip
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "eip-456", resp.ID)
	assert.Equal(t, "full-name-from-get", resp.Name)
}
