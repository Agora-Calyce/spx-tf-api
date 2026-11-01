package handlers_test

import (
	"bytes"
	"encoding/json"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSSHKeyHandler_GetSSHKey_Mapping(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()

	// Mock SPX API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/ssh/ssh1", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "api-token my-token", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"id":          "ssh-uuid",
			"eid":         "ssh1",
			"productName": "ma-cle",
			"ssh": map[string]interface{}{
				"data": map[string]interface{}{
					"spxSSHKey": "c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFD",
				},
				"status": map[string]interface{}{
					"fingerprint": "SHA256:...",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	configStore.SetEndpoint(server.URL)
	h := handlers.NewSSHKeyHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga1/az1/proj1/ssh-keys/ssh1", nil)
	req.Header.Set("Authorization", "api-token my-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "ssh1")

	// Execute
	if assert.NoError(t, h.GetSSHKey(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var ssh models.SSHKey
		err := json.Unmarshal(rec.Body.Bytes(), &ssh)
		assert.NoError(t, err)
		assert.Equal(t, "ssh1", ssh.ID)
		assert.Equal(t, "ma-cle", ssh.Name)
		assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC", ssh.PublicKey)
		assert.Equal(t, "SHA256:...", ssh.Fingerprint)
	}
}

func TestSSHKeyHandler_CreateSSHKey(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()

	// Mock SPX API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/ssh", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]string{"eid": "ssh1"})
		} else if r.Method == "GET" {
			assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/ssh/ssh1", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"eid":         "ssh1",
				"productName": "ma-cle",
				"ssh": map[string]interface{}{
					"data": map[string]interface{}{
						"spxSSHKey": "c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFD",
					},
					"status": map[string]interface{}{
						"fingerprint": "SHA256:...",
					},
				},
			})
		}
	}))
	defer server.Close()

	configStore.SetEndpoint(server.URL)
	h := handlers.NewSSHKeyHandler(configStore)

	sshReq := models.SSHKeyRequest{
		Name:      "ma-cle",
		PublicKey: "ssh-rsa AAAAB3...",
	}
	body, _ := json.Marshal(sshReq)

	req := httptest.NewRequest(http.MethodPost, "/orga1/az1/proj1/ssh-keys", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "api-token my-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga1", "az1", "proj1")

	// Execute
	if assert.NoError(t, h.CreateSSHKey(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		var ssh models.SSHKey
		err := json.Unmarshal(rec.Body.Bytes(), &ssh)
		assert.NoError(t, err)
		assert.Equal(t, "ssh1", ssh.ID)
		assert.Equal(t, "ma-cle", ssh.Name)
		assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC", ssh.PublicKey)
		assert.Equal(t, "SHA256:...", ssh.Fingerprint)
	}
}

func TestSSHKeyHandler_DeleteSSHKey(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()

	// Mock SPX API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/ssh/ssh1", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	configStore.SetEndpoint(server.URL)
	h := handlers.NewSSHKeyHandler(configStore)

	req := httptest.NewRequest(http.MethodDelete, "/orga1/az1/proj1/ssh-keys/ssh1", nil)
	req.Header.Set("Authorization", "api-token my-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "ssh1")

	// Execute
	if assert.NoError(t, h.DeleteSSHKey(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}
}

func TestSSHKeyHandler_AuthorizationHeader(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("http://localhost:8080")
	h := handlers.NewSSHKeyHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga1/az1/proj1/ssh-keys/ssh1", nil)
	// PAS DE HEADER AUTHORIZATION
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "ssh1")

	// Execute
	err := h.GetSSHKey(c)

	// Verify
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}
