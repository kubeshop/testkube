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
- `cmd/kubectl-testkube/commands/completion.go` implements custom completion generation that ensures zsh completion works with the actual binary name (`kubectl-testkube`)
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

## GitOps resource sync

- `internal/sync/` implements the agent side of the GitOps (Kubernetes → Control Plane) sync capability. Reconcilers live in `internal/sync/controller/`, one per syncable kind, and the Control Plane client lives in `internal/sync/grpc/`.
- The syncable kinds are `TestWorkflow`, `TestWorkflowTemplate`, `TestTrigger`, `WorkflowTrigger`, `Webhook`, and `WebhookTemplate`. Adding a kind means touching `internal/sync/controller/`, `internal/sync/grpc/`, the `SyncService` proto, and the Control Plane together.
- The sync controllers are registered in `cmd/api-server/main.go` behind `proContext.CloudStorageSupportedInControlPlane` and `GITOPS_KUBERNETES_TO_CLOUD_ENABLED`, so they only run for a GitOps-persona agent connected to a Control Plane.

### Resource ownership contract

A synced resource in the Control Plane is exclusively owned by one GitOps agent, so that agents syncing from different namespaces cannot silently overwrite each other's resources. The contract spans four layers, and a change to one usually needs a matching change in the others:

1. **Schema** — `Syncable.gitOpsOwner` in `api/v1/testkube.yaml`, a `GitOpsOwner` holding the authoritative `agentId` plus a display-only `agentName` that may be stale after a rename. Run `make generate-openapi` after editing.
2. **Wire** — the Control Plane rejects a sync from a non-owning agent with the gRPC status `codes.FailedPrecondition`. That code is reserved for ownership conflicts on the sync API; do not reuse it there for other validation failures, or agents will misreport the cause.
3. **Agent translation** — `translateError` in `internal/sync/grpc/errors.go` maps `FailedPrecondition` onto the `ErrOwnershipConflict` sentinel in `internal/sync/errors.go`, keeping the original status in the error chain because its message names the current owner. Callers match on the sentinel so that no layer above the client depends on gRPC.
4. **Reconciliation** — `terminalOnOwnershipConflict` in `internal/sync/controller/errors.go` wraps a conflict in `reconcile.TerminalError` so controller-runtime stops requeueing something no retry can fix, while every other error stays retryable with backoff. Conflicts are deliberately not logged there: controller-runtime already logs whatever a reconciler returns, along with the kind and name of the resource.

`skipUnownedResource` in `cmd/api-server/superagentmigration.go` applies the same rule outside the reconcilers, during SuperAgent migration. That migration retries sync failures forever by design, so a conflict has to break the loop or a single unowned resource wedges the migration indefinitely.

Still to come: Control Plane persistence and enforcement of the owner, and the `testkube.io/gitops-owner` annotation for declaring or transferring ownership from Git. Until those land the agent handles a rejection that nothing yet sends.

## Regenerating artifacts

- Update the agent OpenAPI files with `make generate-openapi` after schema edits.
- Regenerate Kubernetes CRDs after editing type definitions in `api/` via `make generate-crds`.
- Regenerate SQL code when query files change via `make generate-sqlc`.
- Refresh mocks for new or updated interfaces using `make generate-mocks`.

## Transient-failure retries

- `pkg/runner/runner.go` runs `worker.Destroy` (cleanup of the execution's Secrets/Pods after the workflow ends) through the shared `retry()` helper via `destroyResources`. Bounded by `CleanupResourcesRetryCount` and `CleanupResourcesRetryDelay`; a brief `kube-apiserver` blip during teardown should not leave orphan resources in the customer namespace.
- `pkg/event/kind/webhook/listener.go` retries the outbound `HttpClient.Do` for `sendRetryCount` attempts with a linear `sendRetryBaseDelay`. Retryable outcomes: network errors, `5xx`, and `429`. Other `4xx` short-circuit so a bad URL / auth failure is not spammed at the subscriber. Delivery is intentionally at-least-once (subscribers own dedupe, matching Stripe/GitHub/Slack convention).

## Telemetry and cluster detection

- `pkg/telemetry/` contains all telemetry event construction, sending, and cluster identification logic.
- `pkg/telemetry/cluster_type.go` implements Kubernetes cluster type detection using a layered approach (node providerID → node labels → server version → kube-system pod names). The result is cached with `sync.Once`.
- When adding support for a new cluster type, add detection entries to the appropriate layer(s) in `cluster_type.go` and add corresponding test cases in `cluster_type_test.go`.
- `cmd/api-server/services/telemetry.go` drives the heartbeat loop that sends `testkube_api_heartbeat` events hourly, including the detected cluster type and agent capabilities.
- `cmd/api-server/services/capabilities.go` extracts agent capability tags (persona, mode, feature flags) from the runtime config for inclusion in telemetry events. When adding new agent features/toggles that should be tracked, add them here and in `capabilities_test.go`.
- The `hosted-runner` tag marks runners that Testkube provisions for trial users, detected from the `tkcagent_hr_` prefix the control plane assigns to `RUNNER_NAME`. Keep `hostedRunnerNamePrefix` in sync with `naming.HostedRunnerAgentName` in `testkube-cloud-api`.
- `pkg/cliruntime/context.go` is a leaf package containing the CLI runtime-context helpers (`IsRunningInDocker`, `DockerContext`, `CliRunContext`, `DetectAITool`). `pkg/telemetry` and `cmd/kubectl-testkube/commands/common` both depend on it; placing it in its own package avoids an import cycle between common and telemetry.
- `DetectAITool` reports the AI coding agent that invoked the CLI (`claude-code`, `codex`, `cursor`, `gemini-cli`, or `""`) purely from env vars the agents set in their subprocesses. Telemetry surfaces it as the `ai_tool` param (Segment property `aiTool`) across all CLI-origin payloads (`NewCLIPayload`, `NewCLIWithLicensePayload`, the inline error/preview payloads in `telemetry.go`, and the MCP tool payload in `mcp.go`).

## CLI update check

- `cmd/kubectl-testkube/commands/common/update_check.go` implements `MaybeNotifyNewerRelease` (per-command post-run hint) and `CheckComponentsStatus` (richer per-component report rendered by `testkube version`). Both consult `pkg/cliruntime` to skip in CI/Docker/Kubernetes contexts and honor the `--output` flag and `TESTKUBE_DISABLE_UPDATE_CHECK` env opt-out.
- `cmd/kubectl-testkube/commands/common/install_source.go` classifies how the running CLI binary was installed (Homebrew, Chocolatey, APT, install.sh, Docker, `go install`, unknown) by inspecting the resolved `os.Executable` path and the Docker context. The classification drives the install-source-specific upgrade command surfaced in the hint.
- Adding a new install channel: extend `DetectInstallSource` and add a test case to `install_source_test.go` that exercises the new path under the relevant `goos`.
- Adding a new CI/runtime detection: extend `pkg/cliruntime/context.go` so both telemetry and the update-check feature stay in sync.
- Adding a new AI-tool detection: extend `DetectAITool` in `pkg/cliruntime/context.go` (add the env-var check and a `TestDetectAITool` case in `context_test.go`); no telemetry wiring changes are needed since payloads already read the `AITool` field.

## On-prem demo install

- `testkube init demo` (`cmd/kubectl-testkube/commands/init.go`) installs the On-Prem demo on the new architecture: the Control Plane (enterprise chart + `values.demo.v2.yaml`) plus a **separate** listener-enabled runner (`kubeshop/testkube-runner`). The bundled agent is gone.
- The CLI generates one agent secret key per install (`common.GenerateDemoAgentSecretKey`) and passes the *same* key to both sides — injected into the Control Plane's `bootstrapConfig` runner so the CP provisions it, and into the runner install (`common.HelmUpgradeOrInstallTestkubeOnPremDemoRunner` → `demoRunnerHelmOptions`). No secret is baked into the binary or chart.
- The runner identity (`demoRunnerID`/`OrgID`/`EnvID`) must stay in sync with the runner declared under `bootstrapConfig` in `values.demo.v2.yaml` (in `testkube-cloud-charts`).
- The legacy `values.demo.yaml` profile (bundled agent, MongoDB) is deprecated but kept for older CLIs.

## Configuration references

- Agent behavior is driven by env vars defined in `internal/config/config.go` (scan for `envconfig:"..."` tags when researching a toggle).
- GitOps sync of Kubernetes resources into the Control Plane is gated by `GITOPS_KUBERNETES_TO_CLOUD_ENABLED` (default `false`), and additionally requires the Control Plane to report cloud storage support.
- TestTriggers accept `spec.event` or a `spec.events` list (mutually exclusive, validated service-side); always consume them via `EffectiveEvents()` so both forms are honored — classification gates that read the single `event` field directly will silently skip list-form triggers.
- Git trigger informer behavior is tuned via `TEST_TRIGGER_GIT_INFORMER_RECONCILE_INTERVAL`, `TEST_TRIGGER_GIT_INFORMER_REPO_DEPTH`, `TEST_TRIGGER_GIT_INFORMER_LIST_TIMEOUT`, `TEST_TRIGGER_GIT_INFORMER_MAX_COMMITS_SCAN`, `TEST_TRIGGER_GIT_INFORMER_PULL_RETRIES`, and `TEST_TRIGGER_GIT_INFORMER_PULL_RETRY_DELAY`.
- Git trigger informer execution is leader-gated in `cmd/api-server/main.go` through the shared `leader` coordinator tasks, so only the active leader performs periodic git pulls/reconciliation.
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
