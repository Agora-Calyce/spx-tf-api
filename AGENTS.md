# AGENTS

## Project

SPX TF Proxy API: a stateless Go/Echo service that sits between the Terraform provider ([`terraform-provider-spx`](https://github.com/Agora-Calyce/terraform-provider-spx)) and the Superphenix (SPX) console API. It turns the SPX API (asynchronous, raw Kubernetes objects, nested request bodies) into flat, synchronous, round-trippable REST resources that Terraform can manage.

`README.md` is the reference for the reasons behind the proxy and for **every SPX workaround** (polling, field mapping, normalization, known limitations). Read the matching section before changing a handler, and keep it up to date.

Upstream SPX API source: https://github.com/super-phenix/superphenix (`internal/superphenix-api/`). The README records the last upstream commit the proxy was synced against.

## Commands

```bash
make build                                         # builds ./spx-tf-api
SPX_API_ENDPOINT=https://console.mon-superphenix.net/api make run
make test                                          # go test -v ./...
go test ./tests/api/handlers -run TestSSHKey       # single test
make swagger                                       # regenerates docs/ (swag annotations)
```

Server listens on `PORT` (default `8080`). Swagger UI: `/swagger/index.html`, health: `/health`.

## Layout

- `cmd/spx-tf-api/main.go`: Echo setup and **all route registrations**.
- `internal/api/handlers/`: one handler per resource. Each handler converts proxy models to SPX bodies, calls the SPX client, waits/polls when needed, and maps SPX responses back to proxy models (`mapXxxFromSPX` functions).
- `internal/api/models/`: flat proxy JSON models (snake_case). These are the contract with the provider.
- `internal/api/store/`: holds the SPX endpoint set at startup.
- `pkg/client/spx/`: thin HTTP client for the SPX API (30 s timeout per call); `models/models.go` holds the SPX-side types.
- `tests/api/handlers/`, `tests/api/models/`: unit tests with `httptest` servers mocking SPX responses (package `handlers_test` / `models_test`).
- `docs/`: generated Swagger files. Do not edit by hand, run `make swagger`.
- `chart/`, `Dockerfile`, `.github/workflows/`: Helm chart and image, built and pushed to GHCR (`ghcr.io/agora-calyce/spx-tf-api`, `oci://ghcr.io/agora-calyce/charts`) by GitHub Actions on `main` and `v*` tags.

## Conventions

- Routes: `/{orga_id}/{az}/{project_id}/<plural-resource>[/{id}]`, `PUT` for updates (exception: security group update is `POST`). The ID is always the SPX `eid`.
- Auth: the provider sends `Authorization: api-token <token>`; handlers forward it to SPX. The proxy stores no token and no resource state.
- Terraform semantics the handlers must preserve (see README "What Terraform needs from an API"):
  - create/update return the full resource, read back with `GET` after the operation;
  - a read returns the same fields and formats as the create request (sorted lists, normalized values, system labels/selectors filtered out);
  - a missing resource returns `404`;
  - an update must not wipe data Terraform omitted (e.g. VPC static routes).
- Keep Swagger annotations (`// @Summary`, `// @Router`…) on handlers in sync with routes.
- Code comments, log and error messages are in English; README is in English. Follow the existing style of the file you edit.

## Changing or adding a field / resource

A change usually spans both projects:

1. Proxy: `internal/api/models` → handler mapping (request to SPX and `mapXxxFromSPX`) → route in `main.go` if new → test in `tests/api/handlers` with a realistic mocked SPX payload → `make swagger` → README "Workarounds" if SPX behaviour had to be worked around.
2. Provider (`terraform-provider-spx`): client struct, resource schema/model, tests, docs and examples (see its `CLAUDE.md`).

When syncing with a new upstream SPX version, re-check every item in the README "Workarounds" section: each one is an assumption about SPX that may no longer hold.

## Gotchas

- Most `GET` handlers return `404` on any SPX error (only instances check for a real 404), and create handlers return `201` with the request echoed back when the follow-up `GET` fails. Documented in README "Known limitations".
- Subnet `GET` polls for readiness (up to 2 min).
- The `spx-tf-api` binary at the repo root is a gitignored build artifact.
