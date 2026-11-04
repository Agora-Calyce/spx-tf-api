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
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestInstanceHandler_Mapping(t *testing.T) {
	// SPX server mock
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/instance/inst1", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"eid":         "inst1",
			"productName": "ma-vm",
			"vm": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"user.superphenix.net/env":  "prod",
						"user.superphenix.net/team": "backend",
						"system/internal":           "should-be-filtered",
					},
				},
				"spec": map[string]interface{}{
					"runStrategy": "Always",
					"preference": map[string]interface{}{
						"name": "linux",
					},
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"domain": map[string]interface{}{
								"cpu": map[string]interface{}{
									"cores": 2,
								},
								"memory": map[string]interface{}{
									"guest": "4Gi",
								},
								"devices": map[string]interface{}{
									"disks": []map[string]interface{}{
										{
											"name": "disk1",
											"disk": map[string]interface{}{
												"bus": "scsi",
											},
										},
										{
											"name": "cloud-init",
											"disk": map[string]interface{}{
												"bus": "virtio",
											},
										},
									},
									"interfaces": []map[string]interface{}{
										{
											"model": "virtio",
										},
									},
								},
							},
							"networks": []map[string]interface{}{
								{
									"multus": map[string]interface{}{
										"networkName": "orga1/sub1",
									},
								},
							},
							"accessCredentials": []map[string]interface{}{
								{
									"sshPublicKey": map[string]interface{}{
										"source": map[string]interface{}{
											"secret": map[string]interface{}{
												"secretName": "ssh-from-cred",
											},
										},
									},
								},
							},
							"architecture": "amd64",
						},
					},
					"disks": []map[string]interface{}{
						{
							"order": 0,
							"bus":   "virtio",
							"eid":   "disk1",
							"cdrom": false,
						},
						{
							"order": 1,
							"bus":   "virtio",
							"eid":   "cloud-init",
							"cdrom": false,
						},
					},
					"cloudInit": map[string]interface{}{
						"custom": false,
						"bus":    "virtio",
						"config": "hostname: test",
					},
					"sshKeys": []string{"ssh1"},
				},
				"status": map[string]interface{}{
					"ready": true,
				},
			},
			"vmi": map[string]interface{}{
				"status": map[string]interface{}{
					"interfaces": []map[string]interface{}{
						{
							"ipAddress": "10.0.0.10",
							"name":      "eth0",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer spxServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga1/az1/proj1/instances/inst1", nil)
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "inst1")

	// Test
	if assert.NoError(t, h.GetInstance(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var inst models.Instance
		err := json.Unmarshal(rec.Body.Bytes(), &inst)
		assert.NoError(t, err)
		assert.Equal(t, "inst1", inst.ID)
		assert.Equal(t, "ma-vm", inst.Name)
		assert.Equal(t, "Always", inst.RunStrategy)
		assert.Equal(t, "linux", inst.VMType)
		assert.Equal(t, 2, inst.CPU)
		assert.Equal(t, 4, inst.Memory)
		assert.Equal(t, "amd64", inst.Architecture)
		assert.Equal(t, "10.0.0.10", inst.IPAddress)
		assert.Contains(t, inst.Labels, "user.superphenix.net/env:prod")
		assert.Contains(t, inst.Labels, "user.superphenix.net/team:backend")
		assert.Len(t, inst.Labels, 2)
		assert.True(t, inst.Ready)
		assert.Len(t, inst.Network, 1)
		assert.Equal(t, "sub1", inst.Network[0].SubnetID)
		assert.Equal(t, 0, inst.Network[0].Order)
		assert.Equal(t, "virtio", inst.Network[0].Model)
		assert.Len(t, inst.Disks, 1)
		assert.Equal(t, "disk1", inst.Disks[0].DiskID)
		assert.Equal(t, "scsi", inst.Disks[0].Bus)
		assert.Equal(t, 0, inst.Disks[0].Order)
		assert.Equal(t, "ssh-from-cred", inst.SSHKeys[0])
		assert.Len(t, inst.SSHKeys, 1)
	}
}

func TestInstanceHandler_ListVMTypes(t *testing.T) {
	// SPX server mock
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/vm-type", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := []string{"linux", "windows", "custom"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer spxServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga1/az1/proj1/vm-types", nil)
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga1", "az1", "proj1")

	// Test
	if assert.NoError(t, h.ListVMTypes(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var vmTypes []string
		err := json.Unmarshal(rec.Body.Bytes(), &vmTypes)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"linux", "windows", "custom"}, vmTypes)
	}
}

func TestInstanceHandler_CreateInstance(t *testing.T) {
	// SPX server mock
	callCount := 0
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/instance", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]interface{}{"eid": "new-inst"})
		} else if r.Method == "GET" {
			assert.Equal(t, "/orga1/api/spx-ctrl/az1/proj1/instance/new-inst", r.URL.Path)
			callCount++
			ready := callCount > 1
			response := map[string]interface{}{
				"eid":         "new-inst",
				"productName": "ma-vm",
				"vm": map[string]interface{}{
					"spec": map[string]interface{}{
						"runStrategy": "Always",
						"compute":     map[string]interface{}{"cpu": 1, "memory": 2},
					},
					"status": map[string]interface{}{
						"ready": ready,
					},
				},
				"vmi": map[string]interface{}{"status": map[string]interface{}{}},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer spxServer.Close()

	// Setup
	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)
	h.PollingInterval = 1 * time.Millisecond

	instReq := models.InstanceRequest{
		Name:        "ma-vm",
		RunStrategy: "Always",
		CPU:         1,
		Memory:      2,
		Network: []models.InstanceNetwork{
			{SubnetID: "sub1"},
		},
	}
	body, _ := json.Marshal(instReq)
	req := httptest.NewRequest(http.MethodPost, "/orga1/az1/proj1/instances", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga1", "az1", "proj1")

	// Test
	if assert.NoError(t, h.CreateInstance(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		var inst models.Instance
		json.Unmarshal(rec.Body.Bytes(), &inst)
		assert.Equal(t, "new-inst", inst.ID)
		assert.True(t, inst.Ready)
		assert.GreaterOrEqual(t, callCount, 2)
	}
}

func TestInstanceHandler_DeleteInstance(t *testing.T) {
	callCount := 0
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
		} else if r.Method == "GET" {
			callCount++
			if callCount <= 2 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"eid": "inst1"})
			} else {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"error": "not found"}`)
			}
		}
	}))
	defer spxServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)
	h.PollingInterval = 1 * time.Millisecond

	req := httptest.NewRequest(http.MethodDelete, "/orga1/az1/proj1/instances/inst1", nil)
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "inst1")

	if assert.NoError(t, h.DeleteInstance(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.GreaterOrEqual(t, callCount, 3)
	}
}

func TestInstanceHandler_UpstreamFields(t *testing.T) {
	var sentBody map[string]interface{}
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&sentBody)
			json.NewEncoder(w).Encode(map[string]interface{}{"eid": "new-inst"})
			return
		}
		fmt.Fprint(w, `{
			"eid": "new-inst",
			"productName": "ma-vm",
			"containerDisks": ["virtio-win"],
			"vm": {
				"spec": {
					"runStrategy": "Always",
					"template": {"spec": {
						"domain": {
							"devices": {
								"interfaces": [{"model": "virtio", "macAddress": "02:00:00:aa:bb:cc", "state": "down"}],
								"tpm": {"enabled": true, "persistent": true}
							},
							"firmware": {"bootloader": {"efi": {"secureBoot": false}}}
						},
						"networks": [{"multus": {"networkName": "proj1/sub1"}}]
					}}
				},
				"status": {"ready": true}
			},
			"vmi": {"status": {}}
		}`)
	}))
	defer spxServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)
	h.PollingInterval = 1 * time.Millisecond

	disabled := false
	secureBoot := false
	tpmOn := true
	containerDisks := []string{"virtio-win"}
	body, _ := json.Marshal(models.InstanceRequest{
		Name: "ma-vm", RunStrategy: "Always", CPU: 1, Memory: 2,
		Network: []models.InstanceNetwork{
			{SubnetID: "sub1", Model: "virtio", Enabled: &disabled, MACAddress: "02:00:00:aa:bb:cc"},
		},
		ContainerDisks: &containerDisks,
		Advanced: &models.InstanceAdvanced{
			TPM: models.InstanceTPM{Enabled: &tpmOn, Persistent: &tpmOn},
			EFI: models.InstanceEFI{SecureBoot: &secureBoot},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/orga1/az1/proj1/instances", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id")
	c.SetParamValues("orga1", "az1", "proj1")

	if !assert.NoError(t, h.CreateInstance(c)) {
		return
	}
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Body sent to SPX
	assert.Equal(t, []interface{}{"virtio-win"}, sentBody["containerDisks"])
	nw := sentBody["network"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, false, nw["enabled"])
	assert.Equal(t, "02:00:00:aa:bb:cc", nw["macAddress"])
	adv := sentBody["advanced"].(map[string]interface{})
	tpm := adv["devices"].(map[string]interface{})["tpm"].(map[string]interface{})
	assert.Equal(t, true, tpm["enabled"])
	assert.Equal(t, true, tpm["persistent"])
	efi := adv["firmware"].(map[string]interface{})["bootloader"].(map[string]interface{})["efi"].(map[string]interface{})
	assert.Equal(t, false, efi["secureBoot"])
	_, hasEnabled := efi["enabled"]
	assert.False(t, hasEnabled, "a nil (inherited) field must not be sent")

	// Read back
	var inst models.Instance
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &inst))
	assert.Equal(t, []string{"virtio-win"}, inst.ContainerDisks)
	if assert.Len(t, inst.Network, 1) {
		assert.Equal(t, "02:00:00:aa:bb:cc", inst.Network[0].MACAddress)
		if assert.NotNil(t, inst.Network[0].Enabled) {
			assert.False(t, *inst.Network[0].Enabled)
		}
	}
	if assert.NotNil(t, inst.Advanced) {
		assert.True(t, *inst.Advanced.TPM.Enabled)
		assert.True(t, *inst.Advanced.EFI.Enabled, "EFI block present => EFI enabled")
		assert.False(t, *inst.Advanced.EFI.SecureBoot)
		assert.Nil(t, inst.Advanced.EFI.Persistent)
	}
}

// Without container_disks or advanced, nothing must be sent (SPX default behavior)
func TestInstanceHandler_UpstreamFields_OmittedByDefault(t *testing.T) {
	var rawBody map[string]interface{}
	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&rawBody)
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Fprint(w, `{"eid": "inst1", "productName": "ma-vm", "vm": {"spec": {"runStrategy": "Always"}, "status": {"ready": true}}, "vmi": {"status": {}}}`)
	}))
	defer spxServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)
	h.PollingInterval = 1 * time.Millisecond

	body, _ := json.Marshal(models.InstanceRequest{
		Name: "ma-vm", RunStrategy: "Always", CPU: 1, Memory: 2,
		Network: []models.InstanceNetwork{{SubnetID: "sub1"}},
	})
	req := httptest.NewRequest(http.MethodPut, "/orga1/az1/proj1/instances/inst1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "inst1")

	if assert.NoError(t, h.UpdateInstance(c)) {
		_, hasCD := rawBody["containerDisks"]
		assert.False(t, hasCD, "containerDisks nil => mounts kept on the SPX side")
		_, hasAdv := rawBody["advanced"]
		assert.False(t, hasAdv)

		var inst models.Instance
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &inst))
		assert.Equal(t, []string{}, inst.ContainerDisks)
		assert.Nil(t, inst.Advanced)
	}
}

func TestInstanceRequest_Validate_MAC(t *testing.T) {
	req := models.InstanceRequest{
		Name: "vm", CPU: 1, Memory: 1,
		Network: []models.InstanceNetwork{{SubnetID: "sub1", MACAddress: "not-a-mac"}},
	}
	assert.Error(t, req.Validate())
	req.Network[0].MACAddress = "02:00:00:aa:bb:cc"
	assert.NoError(t, req.Validate())
}

// The SPX API no longer returns vm.spec.disks: disks are rebuilt from the KubeVirt template
func TestInstanceHandler_DisksFromTemplate(t *testing.T) {
	spxResponse := `{
		"eid": "inst1",
		"productName": "ma-vm",
		"vm": {"spec": {"template": {"spec": {"domain": {"devices": {"disks": [
			{"name": "spx-disk-1", "disk": {"bus": "virtio"}},
			{"name": "spx-iso-1", "cdrom": {"bus": "sata"}},
			{"name": "cloud-init"}
		]}}}}}}
	}`

	spxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, spxResponse)
	}))
	defer spxServer.Close()

	e := echo.New()
	configStore := store.NewConfigStore()
	configStore.SetEndpoint(spxServer.URL)
	h := handlers.NewInstanceHandler(configStore)

	req := httptest.NewRequest(http.MethodGet, "/orga1/az1/proj1/instances/inst1", nil)
	req.Header.Set("Authorization", "api-token valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orga_id", "az", "project_id", "id")
	c.SetParamValues("orga1", "az1", "proj1", "inst1")

	if assert.NoError(t, h.GetInstance(c)) {
		var inst models.Instance
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &inst))
		assert.Equal(t, []models.InstanceDisk{
			{Order: 0, Bus: "virtio", DiskID: "spx-disk-1", Cdrom: false},
			{Order: 1, Bus: "sata", DiskID: "spx-iso-1", Cdrom: true},
		}, inst.Disks)
	}
}
