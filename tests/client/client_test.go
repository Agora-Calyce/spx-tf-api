package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"

	"github.com/stretchr/testify/assert"
)

// L'endpoint SPX peut contenir un chemin (ex. "https://console.mon-superphenix.net/api") :
// il doit préfixer le chemin de chaque appel.
func TestClient_KeepsEndpointPath(t *testing.T) {
	for _, suffix := range []string{"", "/", "/api", "/api/"} {
		t.Run("endpoint"+suffix, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{}`))
			}))
			defer server.Close()

			client, err := spx.NewClient(server.URL+suffix, "my-token")
			assert.NoError(t, err)

			_, err = client.GetSSHKey(context.Background(), "orga1", "az1", "proj1", "ssh1")
			assert.NoError(t, err)

			prefix := ""
			if suffix == "/api" || suffix == "/api/" {
				prefix = "/api"
			}
			assert.Equal(t, prefix+"/orga1/api/spx-ctrl/az1/proj1/ssh/ssh1", gotPath)
		})
	}
}
