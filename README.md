# SPX TF Proxy API

This project is a proxy API written in Go using the [Echo](https://echo.labstack.com/) framework. It serves as an intermediary layer to facilitate the generation of a Terraform provider for calling the SPX API.

> **Transparency Note**: This project was “vibecoded.” It certainly has room for improvement, and all contributions are welcome!

## Why a proxy?

The SPX API was built as the backend of the SPX web console, not as an infrastructure-as-code API. A console can cope with asynchronous operations (it refreshes the page), with raw Kubernetes objects (it picks the fields it displays) and with request and response bodies of different shapes. Terraform cannot: it compares what it sent with what it reads back, attribute by attribute, and any difference becomes a diff, a forced replacement or an "inconsistent result after apply" error. The constraints are listed in [What Terraform needs from an API](#what-terraform-needs-from-an-api).

The proxy bridges that gap. It exposes a small REST API, one flat resource per Terraform resource, with the semantics Terraform expects:

- A create or update call returns only once the resource is usable, and its response is the full resource.
- A read returns the same fields, in the same format, as the request that created the resource.
- A missing resource answers `404`.

This keeps the Terraform provider ([`terraform-provider-spx`](https://github.com/Agora-Calyce/terraform-provider-spx)) thin: it only maps Terraform attributes to flat JSON. All knowledge of the SPX API lives here, so it can be:

- **unit-tested** against mocked SPX responses (`tests/api/handlers`), without Terraform;
- **fixed when the SPX API changes** by redeploying the proxy, without releasing a new provider that every user must upgrade;
- **reused** by other clients that need a simpler, synchronous view of SPX.

The proxy is stateless. It does not store tokens or resources: each request carries the user's SPX token (`Authorization: api-token <token>`), which the proxy forwards to SPX.

## What Terraform needs from an API

For each requirement: why Terraform needs it, what the SPX API does, and how the proxy closes the gap. The details are in [Workarounds](#workarounds).

| # | Terraform requirement | Why | SPX API behaviour | Proxy |
|---|---|---|---|---|
| 1 | **Create returns the full resource**, including computed attributes (ID, IPs, fingerprint…) | Terraform must save every attribute in its state right after `apply`. An unknown value left after apply is a provider error. | Returns only `{"eid": "..."}` | Calls `GET` after create (or echoes the request when SPX computes nothing) and returns the mapped resource |
| 2 | **Create, update and delete finish before they return** | Terraform uses dependencies to order operations: an instance is created once its subnet exists, and a subnet is deleted once its instances are gone. | Operations are asynchronous: the call returns while Kubernetes is still reconciling. | Polls until the resource is ready or gone (see [Waiting for asynchronous operations](#waiting-for-asynchronous-operations)) |
| 3 | **Read returns what was written** (round-trip) | Any difference between config and read-back becomes a permanent diff. On an immutable attribute it forces a replacement on every `plan`. | Returns raw Kubernetes objects (VM, VMI, PVC, NetworkPolicy, kube-ovn subnet…) that look nothing like the create body | Maps each response back to the request format |
| 4 | **Request and response share one schema** | One Terraform schema describes both. | Nested create bodies (`general`, `spec`, `network`, `compute`…) and different response bodies; some fields use different types or casing in each direction | Exposes flat, symmetric JSON models (`internal/api/models`) |
| 5 | **Stable list order** | A list read back in a different order is a diff. | Lists built from Go/Kubernetes maps have no order | Sorts labels, selectors and match labels |
| 6 | **Only user-managed data in responses** | Values Terraform never set appear as unwanted changes. | Adds system labels, injected selectors and auto-generated routes | Filters them out |
| 7 | **Normalized values** (case, units, prefixes) | `"TCP"` vs `"tcp"`, `10` vs `"10Gi"` or `default` vs `spx-<az>-storage01.default` would all be diffs. | Mixed formats | Normalizes in both directions |
| 8 | **`404` when a resource no longer exists** | During refresh, Terraform drops deleted resources from state and plans to recreate them. Any other error stops the run. | Errors come back as a status code with a free-text body | Maps "not found" to `404` (see [Known limitations](#known-limitations)) |
| 9 | **Update applies the desired state, not a patch** | Terraform sends every attribute. Omitting an optional block must not wipe data that Terraform does not manage. | Some updates replace whole sub-collections, e.g. VPC static routes | Re-sends the current values when Terraform omits them |
| 10 | **A stable, unique ID** for `terraform import` and later calls | Terraform stores one ID per resource. | Resources have a local `id` and an effective `eid` (`spx-<uuid>`); every URL uses the `eid` | Uses the `eid` as the resource ID |
| 11 | **Immutable fields are known** | The provider must mark them `RequiresReplace`. | Some fields are silently ignored on update (disk storage class and source, subnet CIDR, SSH key) | Sends only the updatable fields. The provider marks the others `RequiresReplace`. |

## Features

- Proxy to an instance of the SPX API.
 - Built-in Swagger documentation.
- Logging support with Zerolog.
- `/health` health endpoint.

## Prerequisites

- [Go](https://golang.org/doc/install) (version 1.21+)
- [Make](https://www.gnu.org/software/make/)

## Sources

The proxy is based on the API implementation from https://github.com/super-phenix/superphenix:
- `internal/superphenix-api/`

Last synced against upstream commit `e95a634` (2026-10-06). Notable upstream changes since the previous sync:
- Firewalls were renamed to Security Groups (`/security-group`), exposed by the proxy as `/{orga_id}/{az}/{project_id}/security-groups`.
- Security groups can be scoped to subnets (`subnetIds`, max 10).
- Disks support custom labels; instances support container disks, advanced options (TPM/EFI), and per-NIC `enabled` / `mac_address`.
- VPCs support static routes (`static_routes`). The SPX API replaces every route on update, so the proxy re-sends the existing routes when `static_routes` is omitted.

## Installation

```bash
go mod download
```

## Usage

### Start the server

The target API URL is set with the `SPX_API_ENDPOINT` environment variable. The listening port is set with `PORT` (default `8080`).

```bash
SPX_API_ENDPOINT=https://console.mon-superphenix.net/api make run
```

The server will be accessible at `http://localhost:8080`.

### Using with Docker

A prebuilt image is published to GitHub Container Registry (`vX.Y.Z` for releases, `main-<sha>` for the main branch):

```bash
docker pull ghcr.io/agora-calyce/spx-tf-api:<tag>
```

You can also build the image yourself:

```bash
docker build -t spx-tf-api .
```

To start the container, you must set the `SPX_API_ENDPOINT` variable:

```bash
docker run -p 8080:8080 -e SPX_API_ENDPOINT=https://console.mon-superphenix.net/api spx-tf-api
```

### Deployment with Helm (Kubernetes)

A Helm Chart is available in [`chart/`](chart) and published as an OCI artifact:

```bash
helm install spx-tf-api oci://ghcr.io/agora-calyce/charts/spx-tf-api --version <X.Y.Z>
```

## Workarounds

This section lists every place where the proxy goes beyond translating calls one-to-one to work around SPX API behaviour. Check it whenever the SPX API changes: each item is an assumption about SPX that may stop being true.

### General

- **URL layout.** The proxy exposes `/{orga_id}/{az}/{project_id}/<resources>` with plural names and `PUT` for updates. SPX uses `/{orga_id}/api/spx-ctrl/{az}/{project_id}/<resource>` with singular names, and updates with `POST /<resource>/{eid}`. Exceptions:
  - The security group list endpoint has no AZ: `/{orga_id}/api/spx-ctrl/{project_id}/security-group`.
  - The proxy's security group update route is `POST`, not `PUT`.
- **Request bodies.** Flat proxy models are converted to SPX's nested bodies (`general.productName`, `spec`, `network`, `compute`…).
- **Create returns only the `eid`.** What the proxy returns depends on the resource:
  - **Subnet, EIP, instance:** waits until the resource is ready, then returns it as read with `GET`.
  - **SSH key, security group:** calls `GET` once and returns the result.
  - **VPC, disk, load balancer:** no `GET`. The proxy returns the request plus the `eid`, which is safe only because SPX computes nothing for these at creation.
  - **If the `GET` fails**, the response is still `201` with the request echoed back. For a subnet it contains only the ID, name and AZ, and for an instance only the ID and name.
- **Error status codes.** SPX `4xx` errors (`401`, `403`, `400`…) are returned with the same status code, so the provider shows the real cause. Other errors (SPX `5xx`, timeouts, network errors) use the handler's fallback status: `500`, or `404` on reads (see [Known limitations](#known-limitations)).
- **Read-after-write.** Every update handler returns the resource read back with `GET`, not the request. The exception is the LB update, which falls back to echoing the request when the `GET` fails.
- **Custom labels.** Kubernetes objects carry many system labels. Only keys starting with `user.superphenix.net/` are returned, formatted as `key:value` and sorted. This applies to instances and disks.

### Waiting for asynchronous operations

| Resource | Create | Update | Read | Delete |
|---|---|---|---|---|
| Subnet | Polls until `status.v4availableIPrange` or `v6availableIPrange` appears (5 s × 24 = 2 min) | Same | **Same polling on every `GET`** | Polls until `GET` fails (5 s × 60 = 5 min) |
| EIP | Polls until `eip.status.ready` (2 min) | Same, but **skips the first `ready`**, which still reflects the state before the update | Same polling | **Retries the delete once after 1 s** (the first call fails intermittently), then polls until `GET` fails (5 min) |
| Instance | Polls until `vm.status.ready` (5 min) | Same, **skipping the first `ready`** | No wait | **Retries once after 1 s**, then polls until `GET` returns `404` (5 min) |
| VPC, disk, LB, SSH key, security group | No wait | No wait | No wait | No wait |

The provider's HTTP client timeout is 10 minutes, so it outlasts every polling loop above. Each individual call from the proxy to SPX has a 30 s timeout.

### VPC

- **Static routes are replaced as a whole on update.** When the request has no `static_routes`, the proxy reads the VPC and re-sends its current routes, so a name change does not delete them.
- **Routes generated by NAT gateways** (`autoGenerated: true`) are hidden from responses and never sent back: Terraform does not manage them.
- A route without `policy` defaults to `policyDst`.

### Subnet

- **CIDRs.** SPX returns `cidrBlock` as a single comma-separated string (`"10.0.0.0/24,fd00::/64"`). The proxy splits it into `ipv4` and `ipv6`.
- **DNS servers.** SPX returns them as DHCP options (`dns_server=8.8.8.8`) in `dhcpV4Options` / `dhcpV6Options`, which may be either a string or an array of strings.
- **NAT gateway.** `nat_gateway` is inferred from the presence of `natGateway.spec.lanIp`, which is also returned as `nat_gateway_ip`.
- **Updatable fields.** An update sends only `name`, `nat_gateway`, the DNS servers and `private`. The VPC, protocol and CIDRs cannot change.

### Disk

- **Size.** SPX returns a Kubernetes quantity (`"10Gi"`), converted to an integer number of GiB.
- **Storage class.**
  - SPX returns the fully qualified name (`spx-<az>-storage01.default`), so the proxy keeps only the part after the last `.`.
  - Since October 2026, SPX no longer fills `disk.spec.storage.storageClassName`. The proxy then reads `pvc.spec.storageClassName`. Without this, the storage class read back empty and every `plan` forced a replacement of the disk.
- **Source.** The type (`blank`, `http`, `snapshot`, `clone`, `registry`) is deduced from whichever field is present under `disk.spec.source`.
- **Labels.** SPX updates labels on the PVC, while labels set at creation live on the DataVolume. The proxy reads the PVC first and falls back to the DataVolume.
- **Updatable fields.** An update sends only `name`, `size` and `labels`. The storage class and source are immutable (`RequiresReplace` in the provider).
- **Mounted disks cannot be deleted.** SPX refuses with `500 Cannot delete mounted disk`. The proxy passes the error through. See [Known limitations](#known-limitations).

### EIP

The EIP response is the least stable part of the SPX API, so the proxy decodes it as untyped JSON and tries several known formats:

- **Two product types.** `productTypeId: "eip"` objects carry their data in `eip.spec` / `eip.status` (`status.ip`, `status.nat`). Older objects carry it in `fip.spec` / `fip.status` (`status.externalIP`).
- **Subnet.** Read from `spec.natGwDp`, falling back to `spec.subnetEid`.
- **Field casing.** The internal IP may be `internalIP` or `internalIp`, in `spec` or `status`, and every variant is tried.
- **SNAT and DNAT.** Each is either a list of objects (`snat[].spec.internalCIDR`, `dnat[].spec`) or an older single object holding a list (`snat.spec.snat[]`, `dnat.spec.dnat[]`).
- **DNAT values.** Ports are kept as strings and the protocol is lower-cased in both directions.

### Load balancer

- **Selectors.** SPX stores them as `"key:value"` strings. The proxy exposes `{key, value}` objects, and the provider converts them back to `"key:value"` strings in Terraform.
- **Validation done before calling SPX:**
  - the VIP must be in `198.18.0.0/16`;
  - `selectors` and `endpoints` are mutually exclusive;
  - at least one port is required;
  - the protocol must be `TCP` or `UDP`.
- **Port names** are generated by SPX and returned as a computed attribute.

### SSH key

- **Public key.** SPX returns it base64-encoded in `ssh.data.spxSSHKey`. The proxy decodes it, and falls back to the raw value if decoding fails.
- **Fingerprint.** Read from `ssh.status.fingerprint`.
- **No update.** SPX has no update for SSH keys. The proxy exposes no `PUT`, and the provider marks `public_key` `RequiresReplace`.

### Instance

SPX returns the KubeVirt `VirtualMachine` (`vm`) and `VirtualMachineInstance` (`vmi`) objects. The proxy rebuilds the creation request from them:

| Attribute | Read from |
|---|---|
| `cpu` | `vm.spec.template.spec.domain.cpu.cores` |
| `memory` | `domain.memory.guest` (`"4Gi"`), falling back to `vm.spec.compute.memory` |
| `vm_type` | `vm.spec.preference.name` |
| `ssh_keys` | `template.spec.accessCredentials[].sshPublicKey.source.secret.secretName` (the secret is named after the key's `eid`), falling back to `vm.spec.sshKeys` |
| `network[].subnet_id` | `template.spec.networks[].multus.networkName` (`<namespace>/<subnet eid>`) |
| `network[].model`, `mac_address`, `enabled` | `domain.devices.interfaces[i]`, matched by position; `enabled` is `state != "down"` |
| `disks` | `vm.spec.disks`. Since October 2026 that field is absent, so the disks are rebuilt from `domain.devices.disks`: the device name is the disk `eid`, the order is the device order, and a `cdrom` device sets `cdrom = true`. The `cloud-init` device is always skipped. |
| `ip_address` | `vmi.status.interfaces[0].ipAddress`, falling back to `ipAddresses[0]` |
| `advanced` | TPM and firmware overrides in `domain`. `null` when there are none, meaning everything is inherited from the `vm_type`. |
| `container_disks` | top-level `containerDisks`, with `null` returned as `[]` |
| `cloud_init` | `vm.spec.cloudInit`, which SPX no longer returns. The provider keeps the values from its state instead. |

Other details:

- **Update body.** An update sends the same body as a create, and SPX reconciles the VM.
- **Not-found detection.** `404` is detected by searching for `"404"` in the SPX error message, because the SPX client does not expose the status code.

### Security group

SPX stores security groups as Kubernetes `NetworkPolicy` objects:

- **Target.** `podSelector.matchLabels` is a map, so it is converted to sorted `key:value` strings.
- **Injected selectors.** SPX adds `superphenix.net/projectID` to the selector and a `superphenix.net/ignoreNetworkPolicies` match expression. Both are filtered out.
- **`deny_all` / `allow_all`.**
  - A direction listed in `policyTypes` with no rules means "deny all" and is returned as `deny_all = true`.
  - An empty rule (`{}`) means "allow all" and is returned as `allow_all = true`.
- **Ports.**
  - `port` is a Kubernetes `IntOrString`.
  - A missing `endPort` is returned as equal to `port`.
  - In requests, the proxy accepts ports as numbers or numeric strings (`PortNumber`).
- **Subnet scope.** `subnet_ids` (max 10) is sent as `subnetEIds`. `null` is returned as `[]`.

### Workarounds in the provider

A few fixes cannot live in the proxy, because they depend on Terraform's state or plan:

- **`cloud_init` read-back.** The `cloud_init` block is read back only when it is set in the configuration. `bus` and `config` keep their state value when SPX returns them empty.
- **No `(known after apply)` noise.** Computed attributes that SPX fills only after reconciliation, such as the NIC model, disk bus and MAC address, use `UseStateForUnknown`.
- **Sorted storage classes.** The `spx_storage_classes` data source sorts the names, so `names[0]` is stable.

## Known limitations

- **Read errors become `404`.** For VPC, subnet, disk, EIP, LB, SSH key and security group, any SPX `5xx`, timeout or network error on `GET` is returned as `404`. During a refresh, Terraform then drops the resource from state and plans to recreate it. Only instances return `500` for these errors.
- **Create fallbacks hide errors.** When the `GET` after a create fails, or times out while waiting for the resource to be ready, the handler still answers `201` with the request echoed back. Terraform records a resource that may not be ready, and the real state is only fetched on the next refresh. A failure after an update, by contrast, returns `500`, except for load balancers, which echo the request.
- **Subnet reads can block for up to 2 minutes** while a subnet is not ready, because the readiness polling also runs on `GET`.
- **Mounted disks cannot be replaced in place.** When a disk attached to an instance must be replaced (e.g. a `storage_class` change), Terraform deletes the old disk before updating the instance, and SPX refuses because the disk is still mounted. To work around it, detach the disk first, or replace the instance as well (`terraform apply -replace=...`).
- **Leftover state.** `VPCHandler` still keeps an in-memory map of created VPCs. Nothing reads it; it is a remnant of an earlier version.
