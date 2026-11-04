package handlers_test

import (
	"bytes"
	"encoding/json"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestVPCHandler_AuthorizationHeader(t *testing.T) {
	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("https://api.example.com")
	h := handlers.NewVPCHandler(configStore)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Missing Authorization header",
		},
		{
			name:           "Invalid format (no api-token)",
			authHeader:     "token123",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token format (expected: api-token {token})",
		},
		{
			name:           "Invalid format (api-token alone)",
			authHeader:     "api-token ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token format (expected: api-token {token})",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(models.VPCRequest{
				Name: "test-vpc",
			})
			req := httptest.NewRequest(http.MethodPost, "/orga-1/az1/proj-1/vpcs", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("orga_id", "az", "project_id")
			c.SetParamValues("orga-1", "az1", "proj-1")

			err := h.CreateVPC(c)

			// The Echo middleware may turn the error into a response if it is an HTTPError
			if err != nil {
				he, ok := err.(*echo.HTTPError)
				if ok {
					assert.Equal(t, tt.expectedStatus, he.Code)
					assert.Contains(t, he.Message.(string), tt.expectedError)
				} else {
					t.Fatalf("Expected echo.HTTPError, got %v", err)
				}
			} else {
				// If the handler returned no error, check the recorder
				assert.Equal(t, tt.expectedStatus, rec.Code)
				var resp map[string]string
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func TestVPCHandler_AllEndpoints_Authorization(t *testing.T) {
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("https://api.example.com")
	h := handlers.NewVPCHandler(configStore)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
		params map[string]string
	}{
		{
			name:   "CreateVPC",
			method: http.MethodPost,
			path:   "/o/a/p/vpcs",
			body: models.VPCRequest{
				Name: "vpc",
			},
		},
		{
			name:   "GetVPC",
			method: http.MethodGet,
			path:   "/o/a/p/vpcs/1",
		},
		{
			name:   "UpdateVPC",
			method: http.MethodPut,
			path:   "/o/a/p/vpcs/1",
			body: models.VPCRequest{
				Name: "vpc",
			},
		},
		{
			name:   "DeleteVPC",
			method: http.MethodDelete,
			path:   "/o/a/p/vpcs/1",
		},
		{
			name:   "ListVPCs",
			method: http.MethodGet,
			path:   "/o/a/p/vpcs",
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
			// No Authorization header

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if strings.Contains(tt.path, "/vpcs/") {
				c.SetParamNames("id", "orga_id", "az", "project_id")
				c.SetParamValues("1", "o", "a", "p")
			} else {
				c.SetParamNames("orga_id", "az", "project_id")
				c.SetParamValues("o", "a", "p")
			}
			if tt.params != nil {
				q := req.URL.Query()
				for k, v := range tt.params {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			var err error
			switch tt.name {
			case "CreateVPC":
				err = h.CreateVPC(c)
			case "GetVPC":
				err = h.GetVPC(c)
			case "UpdateVPC":
				err = h.UpdateVPC(c)
			case "DeleteVPC":
				err = h.DeleteVPC(c)
			case "ListVPCs":
				err = h.ListVPCs(c)
			}

			if err != nil {
				he, ok := err.(*echo.HTTPError)
				if ok {
					assert.Equal(t, http.StatusUnauthorized, he.Code)
					assert.Contains(t, he.Message.(string), "Missing Authorization header")
				}
			}
		})
	}
}
