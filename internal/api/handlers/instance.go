package handlers

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/models"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"
	"github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"
	spxmodels "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx/models"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// InstanceHandler handles requests related to instances.
type InstanceHandler struct {
	configStore     *store.ConfigStore
	PollingInterval time.Duration
}

// NewInstanceHandler creates a new InstanceHandler.
func NewInstanceHandler(configStore *store.ConfigStore) *InstanceHandler {
	return &InstanceHandler{
		configStore:     configStore,
		PollingInterval: 5 * time.Second,
	}
}

func (h *InstanceHandler) getSPXClient(c echo.Context) (*spx.Client, error) {
	endpoint := h.configStore.GetEndpoint()
	if endpoint == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "API endpoint not configured at startup")
	}

	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "api-token ")
	if token == "" || token == authHeader {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token format (expected: api-token {token})")
	}

	client, err := spx.NewClient(endpoint, token)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error creating API client")
	}

	return client, nil
}

func (h *InstanceHandler) mapInstanceFromSPX(r *spxmodels.InstanceFullResponse) *models.Instance {
	inst := &models.Instance{
		ID:           r.EID,
		Name:         r.ProductName,
		RunStrategy:  r.VM.Spec.RunStrategy,
		VMType:       r.VM.Spec.Preference.Name,
		CPU:          r.VM.Spec.Template.Spec.Domain.CPU.Cores,
		Ready:        r.VM.Status.Ready,
		Architecture: r.VM.Spec.Template.Spec.Architecture,
		CloudInit: models.InstanceCloudInit{
			Custom: r.VM.Spec.CloudInit.Custom,
			Bus:    r.VM.Spec.CloudInit.Bus,
			Config: r.VM.Spec.CloudInit.Config,
		},
	}

	// Mapping SSH Keys
	if len(r.VM.Spec.Template.Spec.AccessCredentials) > 0 {
		var sshKeys []string
		for _, ac := range r.VM.Spec.Template.Spec.AccessCredentials {
			if ac.SSHPublicKey.Source.Secret.SecretName != "" {
				sshKeys = append(sshKeys, ac.SSHPublicKey.Source.Secret.SecretName)
			}
		}
		inst.SSHKeys = sshKeys
	} else {
		inst.SSHKeys = r.VM.Spec.SSHKeys
	}

	// Parsing Memory
	memStr := r.VM.Spec.Template.Spec.Domain.Memory.Guest
	if strings.HasSuffix(memStr, "Gi") {
		memInt, _ := strconv.Atoi(strings.TrimSuffix(memStr, "Gi"))
		inst.Memory = memInt
	} else {
		inst.Memory = r.VM.Spec.Compute.Memory
	}

	// Mapping Network
	for i, nw := range r.VM.Spec.Template.Spec.Networks {
		parts := strings.Split(nw.Multus.NetworkName, "/")
		subnetID := ""
		if len(parts) >= 2 {
			subnetID = parts[1]
		} else if len(parts) == 1 {
			subnetID = parts[0]
		}

		model := ""
		macAddress := ""
		var enabled *bool
		if i < len(r.VM.Spec.Template.Spec.Domain.Devices.Interfaces) {
			iface := r.VM.Spec.Template.Spec.Domain.Devices.Interfaces[i]
			model = iface.Model
			macAddress = iface.MacAddress
			if iface.State != "" {
				up := iface.State != "down"
				enabled = &up
			}
		}

		inst.Network = append(inst.Network, models.InstanceNetwork{
			SubnetID:   subnetID,
			Order:      i,
			Model:      model,
			Enabled:    enabled,
			MACAddress: macAddress,
		})
	}

	// Mapping Disks
	spxDisks := make(map[string]string)
	devicesDisks := r.VM.Spec.Template.Spec.Domain.Devices.Disks
	for _, ad := range devicesDisks {
		// Skip the cloud-init disk
		if ad.Name == "cloud-init" {
			continue
		}
		spxDisks[ad.Name] = ad.Disk.Bus
	}

	if len(r.VM.Spec.Disks) > 0 {
		for i, d := range r.VM.Spec.Disks {
			// Skip the cloud-init disk
			if d.EID == "cloud-init" {
				continue
			}

			bus := d.Bus
			if b, ok := spxDisks[d.EID]; ok {
				bus = b
			}
			inst.Disks = append(inst.Disks, models.InstanceDisk{
				Order:  i,
				Bus:    bus,
				DiskID: d.EID,
				Cdrom:  d.Cdrom,
			})
		}
	} else {
		// The SPX API no longer returns the spx disk list: it is rebuilt from the KubeVirt template
		// (the device name is the disk EID, the order is that of the devices)
		order := 0
		for _, ad := range devicesDisks {
			if ad.Name == "cloud-init" {
				continue
			}
			disk := models.InstanceDisk{
				Order:  order,
				Bus:    ad.Disk.Bus,
				DiskID: ad.Name,
			}
			if ad.CDRom != nil {
				disk.Bus = ad.CDRom.Bus
				disk.Cdrom = true
			}
			inst.Disks = append(inst.Disks, disk)
			order++
		}
	}

	// Mapping Labels
	inst.Labels = mapCustomLabelsFromSPX(r.VM.Metadata.Labels)

	// Mounted container disks
	inst.ContainerDisks = r.ContainerDisks
	if inst.ContainerDisks == nil {
		inst.ContainerDisks = []string{}
	}

	// Advanced options (overrides present in the VM spec)
	inst.Advanced = mapInstanceAdvancedFromSPX(r)

	// Get the primary IP from VMI
	if len(r.VMI.Status.Interfaces) > 0 {
		inst.IPAddress = r.VMI.Status.Interfaces[0].IPAddress
		if inst.IPAddress == "" && len(r.VMI.Status.Interfaces[0].IPAddresses) > 0 {
			inst.IPAddress = r.VMI.Status.Interfaces[0].IPAddresses[0]
		}
	}

	return inst
}

// mapInstanceAdvancedToSPX converts the proxy advanced options to the SPX format (nil = no override)
func mapInstanceAdvancedToSPX(a *models.InstanceAdvanced) *spxmodels.InstanceAdvancedOptions {
	if a == nil {
		return nil
	}
	adv := &spxmodels.InstanceAdvancedOptions{}
	adv.Devices.Tpm = spxmodels.TPMOptions{
		Enabled:    a.TPM.Enabled,
		Persistent: a.TPM.Persistent,
	}
	adv.Firmware.Bootloader.Efi = spxmodels.EFIOptions{
		Enabled:    a.EFI.Enabled,
		SecureBoot: a.EFI.SecureBoot,
		Persistent: a.EFI.Persistent,
	}
	return adv
}

// mapInstanceAdvancedFromSPX rebuilds the advanced options from the VM spec overrides.
// Returns nil if no TPM/EFI override is present (everything is inherited from the vm_type).
func mapInstanceAdvancedFromSPX(r *spxmodels.InstanceFullResponse) *models.InstanceAdvanced {
	domain := r.VM.Spec.Template.Spec.Domain
	adv := &models.InstanceAdvanced{}
	found := false

	if tpm := domain.Devices.TPM; tpm != nil {
		found = true
		adv.TPM = models.InstanceTPM{Enabled: tpm.Enabled, Persistent: tpm.Persistent}
	}

	if domain.Firmware != nil && domain.Firmware.Bootloader != nil {
		bl := domain.Firmware.Bootloader
		if bl.EFI != nil {
			found = true
			enabled := true
			adv.EFI = models.InstanceEFI{Enabled: &enabled, SecureBoot: bl.EFI.SecureBoot, Persistent: bl.EFI.Persistent}
		} else if bl.BIOS != nil {
			found = true
			disabled := false
			adv.EFI = models.InstanceEFI{Enabled: &disabled}
		}
	}

	if !found {
		return nil
	}
	return adv
}

// mapCustomLabelsFromSPX extracts the user labels ("key:value", sorted) from a Kubernetes label map
func mapCustomLabelsFromSPX(spxLabels map[string]string) []string {
	var labels []string
	for k, v := range spxLabels {
		if strings.HasPrefix(k, models.CustomLabelPrefix) {
			labels = append(labels, fmt.Sprintf("%s:%s", k, v))
		}
	}
	sort.Strings(labels)
	return labels
}

func (h *InstanceHandler) mapInstanceRequestToSPX(r *models.InstanceRequest) *spxmodels.CreateInstanceBody {
	body := &spxmodels.CreateInstanceBody{}
	body.General.ProductName = r.Name
	body.General.RunStrategy = r.RunStrategy
	body.General.VMType = r.VMType
	body.General.Labels = r.Labels
	body.Compute.CPU = r.CPU
	body.Compute.Memory = r.Memory
	body.SSHKeys = r.SSHKeys
	body.CloudInit = spxmodels.SPXInstanceCloudInit{
		Custom: r.CloudInit.Custom,
		Bus:    r.CloudInit.Bus,
		Config: r.CloudInit.Config,
	}

	body.ContainerDisks = r.ContainerDisks
	body.Advanced = mapInstanceAdvancedToSPX(r.Advanced)

	for _, nw := range r.Network {
		body.Network = append(body.Network, spxmodels.SPXInstanceNetwork{
			Order:      nw.Order,
			SubnetEID:  nw.SubnetID,
			Model:      nw.Model,
			Enabled:    nw.Enabled,
			IPv4:       nw.IPv4,
			IPv6:       nw.IPv6,
			MACAddress: nw.MACAddress,
		})
	}

	for _, d := range r.Disks {
		body.Disks = append(body.Disks, spxmodels.SPXInstanceDisk{
			Order: d.Order,
			Bus:   d.Bus,
			EID:   d.DiskID,
			Cdrom: d.Cdrom,
		})
	}

	return body
}

func (h *InstanceHandler) fetchInstanceFromSPXWithRetry(ctx context.Context, client *spx.Client, orgaID, az, projectID, effectiveID string, skipFirstReady bool) (*spxmodels.InstanceFullResponse, error) {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(h.PollingInterval)
	defer ticker.Stop()

	firstCheck := true
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout after waiting 5 minutes for the ready state of instance %s", effectiveID)
		case <-ticker.C:
			inst, err := client.GetInstance(ctx, orgaID, az, projectID, effectiveID)
			if err != nil {
				return nil, err
			}

			if inst.VM.Status.Ready {
				if firstCheck && skipFirstReady {
					firstCheck = false
					continue
				}
				return inst, nil
			}
			firstCheck = false
		}
	}
}

// ListInstances handles retrieving all instances.
func (h *InstanceHandler) ListInstances(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	spxInstances, err := client.ListInstances(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Error fetching instances: %v", err))
	}

	instances := make([]*models.Instance, 0, len(spxInstances))
	for _, spxInst := range spxInstances {
		instances = append(instances, h.mapInstanceFromSPX(&spxInst))
	}

	return c.JSON(http.StatusOK, instances)
}

// GetInstance handles retrieving a specific instance.
func (h *InstanceHandler) GetInstance(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	spxInst, err := client.GetInstance(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		if spx.StatusCode(err) == http.StatusNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
		}
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Error fetching the instance: %v", err))
	}

	return c.JSON(http.StatusOK, h.mapInstanceFromSPX(spxInst))
}

// CreateInstance handles the creation of a new instance.
func (h *InstanceHandler) CreateInstance(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	req := new(models.InstanceRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	body := h.mapInstanceRequestToSPX(req)
	createResp, err := client.CreateInstance(c.Request().Context(), orgaID, az, projectID, body)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to create the instance: %v", err))
	}

	// Wait for the Ready status
	spxInst, err := h.fetchInstanceFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, createResp.EID, false)
	if err != nil {
		// Still return an object with the ID if polling fails
		return c.JSON(http.StatusCreated, &models.Instance{
			ID:   createResp.EID,
			Name: req.Name,
		})
	}

	return c.JSON(http.StatusCreated, h.mapInstanceFromSPX(spxInst))
}

// UpdateInstance handles updating an instance.
func (h *InstanceHandler) UpdateInstance(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	req := new(models.InstanceRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	// CreateInstanceBody is used for the update because SPX seems to use the same format
	body := h.mapInstanceRequestToSPX(req)
	// spxmodels.CreateInstanceBody must be converted to spxmodels.UpdateInstanceBody
	updateBody := &spxmodels.UpdateInstanceBody{}
	updateBody.General.ProductName = body.General.ProductName
	updateBody.General.RunStrategy = body.General.RunStrategy
	updateBody.General.VMType = body.General.VMType
	updateBody.General.Labels = body.General.Labels
	updateBody.Compute.CPU = body.Compute.CPU
	updateBody.Compute.Memory = body.Compute.Memory
	updateBody.Network = body.Network
	updateBody.Disks = body.Disks
	updateBody.CloudInit = body.CloudInit
	updateBody.SSHKeys = body.SSHKeys
	updateBody.ContainerDisks = body.ContainerDisks
	updateBody.Advanced = body.Advanced

	err = client.UpdateInstance(c.Request().Context(), orgaID, az, projectID, id, updateBody)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to update the instance: %v", err))
	}

	// Wait for the Ready status (ignoring the first ready state if this is an asynchronous update)
	spxInst, err := h.fetchInstanceFromSPXWithRetry(c.Request().Context(), client, orgaID, az, projectID, id, true)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Error after update (ready timeout): %v", err))
	}

	return c.JSON(http.StatusOK, h.mapInstanceFromSPX(spxInst))
}

// DeleteInstance handles the deletion of an instance.
func (h *InstanceHandler) DeleteInstance(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")
	id := c.Param("id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	err = client.DeleteInstance(c.Request().Context(), orgaID, az, projectID, id)
	if err != nil {
		// Single retry after 1s
		time.Sleep(1 * time.Second)
		err = client.DeleteInstance(c.Request().Context(), orgaID, az, projectID, id)
		if err != nil {
			return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Unable to delete the instance: %v", err))
		}
	}

	// Wait for the actual deletion
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(h.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-timeout:
			return echo.NewHTTPError(http.StatusInternalServerError, "Timeout waiting for the instance deletion")
		case <-ticker.C:
			_, err := client.GetInstance(c.Request().Context(), orgaID, az, projectID, id)
			if err != nil {
				if spx.StatusCode(err) == http.StatusNotFound {
					return c.NoContent(http.StatusNoContent)
				}
				return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Error while polling for deletion: %v", err))
			}
		}
	}
}

// ListVMTypes returns the list of all VM types available for a given project and AZ.
// @Summary List VM types
// @Description Returns the list of all VM types available for a given project and AZ.
// @Tags instances
// @Accept json
// @Produce json
// @Param orga_id path string true "Organization ID"
// @Param az path string true "AZ code"
// @Param project_id path string true "Project ID"
// @Success 200 {array} string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{orga_id}/{az}/{project_id}/vm-types [get]
func (h *InstanceHandler) ListVMTypes(c echo.Context) error {
	orgaID := c.Param("orga_id")
	az := c.Param("az")
	projectID := c.Param("project_id")

	client, err := h.getSPXClient(c)
	if err != nil {
		return err
	}

	vmTypes, err := client.ListVMTypes(c.Request().Context(), orgaID, az, projectID)
	if err != nil {
		return echo.NewHTTPError(spxStatus(err, http.StatusInternalServerError), fmt.Sprintf("Error fetching VM types: %v", err))
	}

	return c.JSON(http.StatusOK, vmTypes)
}
