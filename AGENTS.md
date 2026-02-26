# Working with Testkube Core (testkube)

## Purpose

- Implements the Testkube agent services that run inside clusters.
- Provides the Testkube CLI (`kubectl-testkube`) for interacting with Testkube.
- Exposes shared primitives and client structs for downstream tooling.
- Defines the agent OpenAPI contract in `api/v1/testkube.yaml`.

## Entry points

- `cmd/api-server` is the main agent API server; agent personas (superagent, runner, listener, GitOps, etc.) are enabled through Helm values and env configuration.
- `cmd/kubectl-testkube` is the Testkube CLI for managing tests, workflows, and interacting with Testkube installations.
- `cmd/testworkflow-init` initializes TestWorkflow execution containers and orchestrates workflow step groups.
- `cmd/testworkflow-toolkit` provides runtime utilities and commands for TestWorkflow containers (artifacts, services, parallel execution, etc.).
- `cmd/tcl/devbox-mutating-webhook` is a Kubernetes mutating webhook for injecting devbox containers into pods.
- `cmd/tcl/devbox-binary-storage` serves as a binary storage server for devbox dependencies and cached files.
- `cmd/debug-server` is a simple HTTP server that dumps incoming requests for debugging purposes.
- `cmd/proxy` proxies HTTP requests to the Testkube API server for local development and debugging.
- `cmd/choco-stub` displays a deprecation message for the old Chocolatey package location.
- `cmd/tools` contains internal tooling for release management and version bumping.

## MCP integration

- `pkg/mcp/` implements the Model Context Protocol server for AI assistant integration.
- Exposes tools across workflows, executions, artifacts, and metadata via `testkube mcp serve` (CLI), Docker image (`testkube/mcp-server`), or Control Plane's `/mcp` endpoint per environment.
- Uses interface-based tool design; new tools need registration in both `pkg/mcp/server.go` and control plane's `mcp_handler.go`.
- See `pkg/mcp/README.md` for architecture, tool patterns, and usage examples.

## Regenerating artifacts

- Update the agent OpenAPI files with `make generate-openapi` after schema edits.
- Regenerate Kubernetes CRDs after editing type definitions in `api/` via `make generate-crds`.
- Regenerate SQL code when query files change via `make generate-sqlc`.
- Refresh mocks for new or updated interfaces using `make generate-mocks`.

## Telemetry and cluster detection

- `pkg/telemetry/` contains all telemetry event construction, sending, and cluster identification logic.
- `pkg/telemetry/cluster_type.go` implements Kubernetes cluster type detection using a layered approach (node providerID → node labels → server version → kube-system pod names). The result is cached with `sync.Once`.
- When adding support for a new cluster type, add detection entries to the appropriate layer(s) in `cluster_type.go` and add corresponding test cases in `cluster_type_test.go`.
- `cmd/api-server/services/telemetry.go` drives the heartbeat loop that sends `testkube_api_heartbeat` events hourly, including the detected cluster type and agent capabilities.
- `cmd/api-server/services/capabilities.go` extracts agent capability tags (persona, mode, feature flags) from the runtime config for inclusion in telemetry events. When adding new agent features/toggles that should be tracked, add them here and in `capabilities_test.go`.

## Configuration references

- Agent behavior is driven by env vars defined in `internal/config/config.go` (scan for `envconfig:"..."` tags when researching a toggle).
- Helm chart values are the source of deployment defaults; `build/_local/values.dev.yaml` (shaped by the `values.dev.tpl.yaml` template) shows the local overrides used by `tk-dev` if you need a concrete reference.

## Architecture reference

- See [`ARCHITECTURE.md`](ARCHITECTURE.md) for a detailed description of the agent's components, storage layer, event system, CRDs, CLI, and Kubernetes deployment.
- When making changes that affect the architecture (new entry points, storage backends, event listeners, CRDs, API routes, etc.), update `ARCHITECTURE.md` to keep it in sync.

## Keeping documentation in sync

After completing any code change, check whether `AGENTS.md` or `ARCHITECTURE.md` need updates. Apply changes when any of the following are true:

- **New or removed entry points** (`cmd/` binaries, API routes, controllers) → update both files.
- **New or changed packages / key files** (e.g. adding a file like `pkg/telemetry/cluster_type.go`) → add or update the relevant section in `AGENTS.md` so future agents know where to look, and in `ARCHITECTURE.md` so the system description stays accurate.
- **Changed detection / identification logic** (cluster type, CLI run context, Docker context, etc.) → update the corresponding section in `ARCHITECTURE.md` and any guidance in `AGENTS.md`.
- **New storage backends, event listeners, CRDs, or external integrations** → update `ARCHITECTURE.md`.
- **New configuration knobs or environment variables** → mention them in `AGENTS.md` under "Configuration references" if they affect agent behavior.
- **New code-generation or build steps** → add them under "Regenerating artifacts" in `AGENTS.md`.

When in doubt, err on the side of updating — stale documentation is worse than a small extra commit.

## Pre-commit checks

Before committing, always verify your changes pass linting and build:

```bash
make lint          # Run golangci-lint (or `make lint-fix` to auto-fix)
go build ./...     # Verify compilation
```

If your changes include tests, also run `make unit-tests` before pushing.

## PR title format

PR titles **must** follow [Conventional Commits](https://www.conventionalcommits.org/) format with a type prefix. CI will reject PRs without one. Examples:

- `feat: Add soft-delete for workflow executions`
- `fix: Retry log stream on 502 errors`
- `chore: Add contextcheck linter`

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

## Tips

- Review the Makefile for additional helper targets when unfamiliar tasks come up.
