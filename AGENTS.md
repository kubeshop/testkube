# Working with Testkube Core (testkube)

## Deprecated components — DO NOT reference

- **testkube-operator** (`k8s/helm/testkube-operator/`, `testkube-operator` Helm values): The Kubernetes operator is deprecated and disabled by default. Do not suggest enabling it, reference it in documentation, or add new code that depends on it. The Helm chart still carries it as a dependency for backwards compatibility only.

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
- `pkg/cliruntime/context.go` is a leaf package containing the CLI runtime-context helpers (`IsRunningInDocker`, `DockerContext`, `CliRunContext`). `pkg/telemetry` and `cmd/kubectl-testkube/commands/common` both depend on it; placing it in its own package avoids an import cycle between common and telemetry.

## CLI update check

- `cmd/kubectl-testkube/commands/common/update_check.go` implements `MaybeNotifyNewerRelease` (per-command post-run hint) and `CheckComponentsStatus` (richer per-component report rendered by `testkube version`). Both consult `pkg/cliruntime` to skip in CI/Docker/Kubernetes contexts and honor the `--output` flag and `TESTKUBE_DISABLE_UPDATE_CHECK` env opt-out.
- `cmd/kubectl-testkube/commands/common/install_source.go` classifies how the running CLI binary was installed (Homebrew, Chocolatey, APT, install.sh, Docker, `go install`, unknown) by inspecting the resolved `os.Executable` path and the Docker context. The classification drives the install-source-specific upgrade command surfaced in the hint.
- Adding a new install channel: extend `DetectInstallSource` and add a test case to `install_source_test.go` that exercises the new path under the relevant `goos`.
- Adding a new CI/runtime detection: extend `pkg/cliruntime/context.go` so both telemetry and the update-check feature stay in sync.

## Configuration references

- Agent behavior is driven by env vars defined in `internal/config/config.go` (scan for `envconfig:"..."` tags when researching a toggle).
- Helm chart values are the source of deployment defaults; `build/_local/values.dev.yaml` (shaped by the `values.dev.tpl.yaml` template) shows the local overrides used by `tk-dev` if you need a concrete reference.
- CLI update-check toggle: set `TESTKUBE_DISABLE_UPDATE_CHECK=1` to suppress both the per-command hint and the `testkube version` status block. The CLI persists `lastUpdateCheckAt` and `latestKnownVersion` in `~/.testkube/config.json` to throttle the per-command hint to once per day.

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

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `ci`, `chore`

## Tips

- Review the Makefile for additional helper targets when unfamiliar tasks come up.
