package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockSPXStatus démarre une API SPX qui répond toujours avec le statut donné.
func mockSPXStatus(t *testing.T, status int) *store.ConfigStore {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(http.StatusText(status)))
	}))
	t.Cleanup(server.Close)
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(server.URL)
	return configStore
}

// httpStatus renvoie le statut final d'un handler, qu'il écrive la réponse ou renvoie une echo.HTTPError.
func httpStatus(t *testing.T, rec *httptest.ResponseRecorder, err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	assert.NoError(t, err)
	return rec.Code
}

func newContext(path string, names, values []string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "api-token my-token")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames(names...)
	c.SetParamValues(values...)
	return c, rec
}

func TestSPXErrors_4xxForwarded(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			configStore := mockSPXStatus(t, status)

			// Liste : auparavant 500
			c, rec := newContext("/orga1/az1/proj1/vm-types", []string{"orga_id", "az", "project_id"}, []string{"orga1", "az1", "proj1"})
			err := handlers.NewInstanceHandler(configStore).ListVMTypes(c)
			assert.Equal(t, status, httpStatus(t, rec, err))

			// Lecture : auparavant 404, ce qui faisait sortir la ressource du state Terraform
			c, rec = newContext("/orga1/az1/proj1/vpcs/vpc1", []string{"orga_id", "az", "project_id", "id"}, []string{"orga1", "az1", "proj1", "vpc1"})
			err = handlers.NewVPCHandler(configStore).GetVPC(c)
			assert.Equal(t, status, httpStatus(t, rec, err))
		})
	}
}

func TestSPXErrors_5xxKeepsFallback(t *testing.T) {
	configStore := mockSPXStatus(t, http.StatusBadGateway)

	c, rec := newContext("/orga1/az1/proj1/vm-types", []string{"orga_id", "az", "project_id"}, []string{"orga1", "az1", "proj1"})
	err := handlers.NewInstanceHandler(configStore).ListVMTypes(c)
	assert.Equal(t, http.StatusInternalServerError, httpStatus(t, rec, err))

	c, rec = newContext("/orga1/az1/proj1/vpcs/vpc1", []string{"orga_id", "az", "project_id", "id"}, []string{"orga1", "az1", "proj1", "vpc1"})
	err = handlers.NewVPCHandler(configStore).GetVPC(c)
	assert.Equal(t, http.StatusNotFound, httpStatus(t, rec, err))
}
