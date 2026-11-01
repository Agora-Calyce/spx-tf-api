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

func TestLoadBalancerHandler_Authorization(t *testing.T) {
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint("https://api.example.com")
	h := handlers.NewLoadBalancerHandler(configStore)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "CreateLoadBalancer",
			method: http.MethodPost,
			path:   "/o/a/p/load-balancers",
			body: models.LoadBalancerRequest{
				Name: "lb",
				VIP:  "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 80, Protocol: "TCP"},
				},
			},
		},
		{
			name:   "GetLoadBalancer",
			method: http.MethodGet,
			path:   "/o/a/p/load-balancers/1",
		},
		{
			name:   "UpdateLoadBalancer",
			method: http.MethodPut,
			path:   "/o/a/p/load-balancers/1",
			body: models.LoadBalancerRequest{
				Name: "lb",
				VIP:  "198.18.0.1",
				Ports: []models.LoadBalancerPort{
					{Port: 80, TargetPort: 80, Protocol: "TCP"},
				},
			},
		},
		{
			name:   "DeleteLoadBalancer",
			method: http.MethodDelete,
			path:   "/o/a/p/load-balancers/1",
		},
		{
			name:   "ListLoadBalancers",
			method: http.MethodGet,
			path:   "/o/a/p/load-balancers",
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

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("orga_id", "az", "project_id", "id")
			c.SetParamValues("o", "a", "p", "1")

			var err error
			switch tt.name {
			case "CreateLoadBalancer":
				err = h.CreateLoadBalancer(c)
			case "GetLoadBalancer":
				err = h.GetLoadBalancer(c)
			case "UpdateLoadBalancer":
				err = h.UpdateLoadBalancer(c)
			case "DeleteLoadBalancer":
				err = h.DeleteLoadBalancer(c)
			case "ListLoadBalancers":
				err = h.ListLoadBalancers(c)
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
