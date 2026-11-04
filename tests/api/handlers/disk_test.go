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

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestDiskHandler_GetDisk_Mapping(t *testing.T) {
	// Simulated SPX response data
	spxResponse := `{
		"eid": "disk-123",
		"productName": "mon-disque",
		"codeAZ": "az1",
		"disk": {
			"spec": {
				"storage": {
					"resources": {
						"requests": {
							"storage": "10Gi"
						}
					},
					"storageClassName": "spx-aq01-test02-storage01.default"
				},
				"source": {
					"http": {
						"url": "http://example.com/image.qcow2"
					}
				}
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
	h := handlers.NewDiskHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/disks/disk-123", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-123")

	// Execution
	err := h.GetDisk(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Disk
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping
	assert.Equal(t, "disk-123", resp.ID)
	assert.Equal(t, "mon-disque", resp.Name)
	assert.Equal(t, 10, resp.Size)
	assert.Equal(t, "default", resp.StorageClass)
	assert.NotNil(t, resp.Source)
	assert.Equal(t, "http", resp.Source.Type)
	assert.Equal(t, "http://example.com/image.qcow2", resp.Source.URL)
}

func TestDiskHandler_GetDisk_SnapshotMapping(t *testing.T) {
	// Simulated SPX response data for a snapshot
	spxResponse := `{
		"eid": "disk-456",
		"productName": "disque-snapshot",
		"codeAZ": "az1",
		"disk": {
			"spec": {
				"storage": {
					"resources": {
						"requests": {
							"storage": "20Gi"
						}
					},
					"storageClassName": "default"
				},
				"source": {
					"snapshot": {
						"diskEId": "disk-parent"
					}
				}
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
	h := handlers.NewDiskHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/disks/disk-456", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-456")

	// Execution
	err := h.GetDisk(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Disk
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Check the mapping
	assert.Equal(t, "disk-456", resp.ID)
	assert.Equal(t, "snapshot", resp.Source.Type)
	assert.Equal(t, "disk-parent", resp.Source.Snapshot)
}

func TestDiskHandler_GetDisk_CloneMapping(t *testing.T) {
	// Simulated SPX response data for a clone
	spxResponse := `{
		"eid": "disk-789",
		"productName": "disque-clone",
		"codeAZ": "az1",
		"disk": {
			"spec": {
				"storage": {
					"resources": {"requests": {"storage": "5Gi"}},
					"storageClassName": "standard"
				},
				"source": {
					"clone": {
						"diskEId": "disk-original"
					}
				}
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
	h := handlers.NewDiskHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/disks/disk-789", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-789")

	// Execution
	err := h.GetDisk(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Disk
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Check the mapping
	assert.Equal(t, "clone", resp.Source.Type)
	assert.Equal(t, "disk-original", resp.Source.Clone)
}

func TestDiskHandler_GetDisk_RegistryMapping(t *testing.T) {
	// Simulated SPX response data for registry
	spxResponse := `{
		"eid": "disk-101",
		"productName": "disque-registry",
		"codeAZ": "az1",
		"disk": {
			"spec": {
				"storage": {
					"resources": {"requests": {"storage": "10Gi"}},
					"storageClassName": "standard"
				},
				"source": {
					"registry": {
						"url": "docker://registry.example.com/image:latest"
					}
				}
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
	h := handlers.NewDiskHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/disks/disk-101", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-101")

	// Execution
	err := h.GetDisk(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Disk
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Check the mapping
	assert.Equal(t, "registry", resp.Source.Type)
	assert.Equal(t, "docker://registry.example.com/image:latest", resp.Source.URL)
}

func TestDiskHandler_UpdateDisk(t *testing.T) {
	// Request data
	updateReq := models.DiskRequest{
		Name:         "new-name",
		Size:         20,
		StorageClass: "fast",
		Source: &models.DiskSource{
			Type: "blank",
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			// Check that the body sent to SPX contains all the fields
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			general := body["general"].(map[string]interface{})
			assert.Equal(t, "new-name", general["productName"])
			assert.Equal(t, "20", general["storage"])

			// Check that the other fields are no longer sent
			assert.Nil(t, general["storageClass"])
			assert.Nil(t, general["source"])

			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			// Simulate GetDisk after the update
			spxResponse := `{
				"eid": "disk-123",
				"productName": "new-name",
				"codeAZ": "az1",
				"disk": {
					"spec": {
						"storage": {
							"resources": {"requests": {"storage": "20Gi"}},
							"storageClassName": "fast"
						},
						"source": {"blank": {}}
					}
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, spxResponse)
			return
		}
	}))
	defer mockServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewDiskHandler(configStore)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/orga-1/az1/proj-1/disks/disk-123", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "api-token secret-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-123")

	// Execution
	err := h.UpdateDisk(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.Disk
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "new-name", resp.Name)
	assert.Equal(t, 20, resp.Size)
	assert.Equal(t, "fast", resp.StorageClass)
}

func TestDiskHandler_UpdateDisk_Labels(t *testing.T) {
	var sentLabels interface{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			sentLabels = body["general"].(map[string]interface{})["labels"]
			w.WriteHeader(http.StatusOK)
			return
		}
		// Labels updated on the PVC, the DataVolume keeps the creation labels
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"eid": "disk-123",
			"productName": "disk",
			"disk": {
				"metadata": {"labels": {"user.superphenix.net/env": "old"}},
				"spec": {"storage": {"resources": {"requests": {"storage": "20Gi"}}, "storageClassName": "fast"}, "source": {"blank": {}}}
			},
			"pvc": {
				"metadata": {"labels": {"user.superphenix.net/team": "backend", "user.superphenix.net/env": "prod", "app.kubernetes.io/managed-by": "cdi"}},
				"spec": {"volumeName": "pv-1"}
			},
			"mountStatus": {"isMounted": true, "by": "inst-1"}
		}`)
	}))
	defer mockServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewDiskHandler(configStore)

	body, _ := json.Marshal(models.DiskRequest{
		Name: "disk", Size: 20, StorageClass: "fast",
		Source: &models.DiskSource{Type: "blank"},
		Labels: []string{"user.superphenix.net/env:prod", "user.superphenix.net/team:backend"},
	})
	req := httptest.NewRequest(http.MethodPut, "/orga-1/az1/proj-1/disks/disk-123", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "api-token secret-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-123")

	assert.NoError(t, h.UpdateDisk(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []interface{}{"user.superphenix.net/env:prod", "user.superphenix.net/team:backend"}, sentLabels)

	var disk models.Disk
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &disk))
	// Labels read from the PVC, filtered and sorted
	assert.Equal(t, []string{"user.superphenix.net/env:prod", "user.superphenix.net/team:backend"}, disk.Labels)
}

func TestDiskRequest_Validate_Labels(t *testing.T) {
	req := models.DiskRequest{
		Name: "disk", Size: 10, StorageClass: "fast",
		Source: &models.DiskSource{Type: "blank"},
		Labels: []string{"env:prod"},
	}
	err := req.Validate()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "user.superphenix.net/")
	}
}

// The SPX API no longer returns storageClassName in the disk spec: it must be read from the PVC
func TestDiskHandler_GetDisk_StorageClassFromPVC(t *testing.T) {
	spxResponse := `{
		"eid": "disk-123",
		"productName": "mon-disque",
		"disk": {"spec": {"storage": {"resources": {"requests": {"storage": "2Gi"}}}, "source": {"blank": {}}}},
		"pvc": {"spec": {"storageClassName": "default"}}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, spxResponse)
	}))
	defer mockServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(mockServer.URL)
	h := handlers.NewDiskHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga-1/az1/proj-1/disks/disk-123", nil)
	req.Header.Set("Authorization", "api-token secret-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga-1", "az1", "proj-1", "disk-123")

	if assert.NoError(t, h.GetDisk(c)) {
		var resp models.Disk
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "default", resp.StorageClass)
		assert.Equal(t, 2, resp.Size)
		assert.Equal(t, "blank", resp.Source.Type)
	}
}
