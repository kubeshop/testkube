# Working with Testkube Core (testkube)

Testkube Core: the agent services that run in a cluster, the `kubectl-testkube` CLI, and the shared Go packages and API contract the Control Plane builds on.

Owner: `@kubeshop/testkube-backend`, per `.github/CODEOWNERS`. `test/` is shared with `@kubeshop/testkube-qa`; CI, release and lint configuration (`.github/`, `goreleaser_files/`, `.goreleaser*.yml`, `.golangci.yml`, `.builds-*`) belong to `@kubeshop/testkube-devops`.

## 1. Verifying a change

Verification has two loops. Run these local commands before every commit, and the pull request workflows in section 6 before asking for review.

```bash
make lint          # golangci-lint (or `make lint-fix` to auto-fix)
make compile       # go build ./..., compilation only
make unit-tests    # what oss-unit-tests runs; add GOFLAGS=-count=1 to skip cached results
```

Use these make targets rather than calling `go build` or `go test` directly: `oss-unit-tests` runs `make unit-tests`, and the test targets carry the pinned `gotestsum` and the environment the integration tests expect (`INTEGRATION`, `TESTKUBE_PROJECT_ROOT`, the MinIO credentials). `make unit-tests` and `make integration-tests` take `PKG=` (packages) and `RUN=` (a `go test -run` pattern) to narrow a run.

Run the narrower checks when the change touches what they cover:

- `make verify-protobuf-generated` after editing `proto/`, and `make verify-crds-generated` after editing `api/` types (see "Regenerating artifacts").
- `make integration-tests` runs only tests ending in `_Integration`. It expects MongoDB on `:27017`, NATS on `:4222`, MinIO on `:9000` (`minio99`/`minio123`) and Postgres on `:5432` (`API_POSTGRES_URL=postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable`); `make run-mongo run-nats` starts only the first two, as Docker containers, so it is for a developer host only.
- `make integration-tests` also runs the toolkit tests (`test/integration/components/toolkit`), which `oss-integration-tests` (section 6) skips, so no pull request check covers them. They need a disposable cluster in the current kube context, such as the Tilt `k3d-testkube-dev` ("Running locally", section 7), never a shared one: each test creates and deletes a `testkube-test-<n>` namespace with `busybox`, `python` and `golang` pods. Run only them with `make integration-tests PKG=./test/integration/components/toolkit/ GOFLAGS=-count=1`.

`make lint` and `oss-lint-go` both run golangci-lint `v2.15.0`: `GOLANGCI_LINT_VERSION` in the `Makefile`, and the image in `test/testkube/oss-tests/gh-quality-loop.yaml`. Bump both together, along with the version in that workflow's cache key, or the two can disagree; the workflow is the one that gates the pull request.

## 2. Reproducing a bug

Write the failing test first, at the lowest level that shows the bug, then fix it.

- **Unit tests** sit next to the code (`foo_test.go`) and use `testify` (`require` for preconditions, `assert` for checks), usually as a table of cases. Fakes go in through interfaces; reviewers ask for that instead of package-level function variables that tests swap. Generated `gomock` mocks (`mock_*.go`) exist for the larger interfaces. Run one test with `make unit-tests PKG=./pkg/<pkg>/ RUN='^TestName$' GOFLAGS=-count=1`.
- **Fixtures** live in a `testdata/` folder beside the test (for example `pkg/mcp/boards/testdata/`, `cmd/testworkflow-init/orchestration/testdata/`). Test Workflow specs for smoke tests are under `test/*/crd-workflow/`.
- **Tests against a real service** end in `_Integration` and start with `test.IntegrationTest(t)` (`pkg/utils/test/integration.go`), which skips unless `INTEGRATION` is set. For Postgres, `testpostgres.PreparePostgresTestDatabase(t, "<name>")` (`pkg/test/postgres/helpers.go`) creates a database per test from `API_POSTGRES_URL` and drops it on cleanup. The Mongo tests connect to `mongodb://localhost:27017`. Start the services as "Verifying a change" describes, then run `make integration-tests PKG=./pkg/<pkg>/ RUN='^TestName_Integration$' GOFLAGS=-count=1`.
- **`testworkflow-init` and `testworkflow-toolkit` behaviour** is tested in `test/integration/components/`; `make integration-tests PKG=./test/integration/components/...` builds both binaries before running them.
- **A bug that needs a cluster** (pod scheduling, a workflow that runs wrong): start the Tilt agent and point its CLI at it, as [`DEVELOPMENT.md`](DEVELOPMENT.md#testkube-cli) describes (`run-cli-command` → Configure). Starting Tilt does not switch the CLI's saved context, so skip that step and the run goes to whatever Control Plane the CLI was last on. With the kube context on `k3d-testkube-dev`, run `kubectl apply -n testkube-dev -f <workflow>.yaml` and `./build/_local/kubectl-testkube run testworkflow <name> --watch`; `testkube pro login` switches back. Turn the result into a unit test on the processor (`pkg/testworkflows/testworkflowprocessor`) or the runner where you can, so the fix stays covered without a cluster.
- **A bug that needs the Control Plane** reproduces only on the full stack in testkube-workspace.

## 3. Repository map

### Purpose

- Implements the Testkube agent services that run inside clusters.
- Provides the Testkube CLI (`kubectl-testkube`) for interacting with Testkube.
- Exposes shared primitives and client structs for downstream tooling.
- Defines the agent OpenAPI contract in `api/v1/testkube.yaml`.

### Entry points

- `cmd/api-server` is the main agent API server; agent personas (superagent, runner, listener, GitOps, etc.) are enabled through Helm values and env configuration.
- `cmd/kubectl-testkube` is the Testkube CLI for managing tests, workflows, and interacting with Testkube installations.
- `cmd/testworkflow-init` initializes TestWorkflow execution containers and orchestrates workflow step groups.
- `cmd/kubectl-testkube/commands/completion.go` implements custom completion generation that ensures zsh completion works with the actual binary name (`kubectl-testkube`)
- `cmd/testworkflow-toolkit` provides runtime utilities and commands for TestWorkflow containers (artifacts, services, parallel execution, etc.).
- `cmd/tcl/devbox-mutating-webhook` is a Kubernetes mutating webhook for injecting devbox containers into pods.
- `cmd/tcl/devbox-binary-storage` serves as a binary storage server for devbox dependencies and cached files.
- `cmd/convert` is a one-shot tool that migrates control-plane data from MongoDB to PostgreSQL (see `pkg/convert/doc.go`).
- `cmd/debug-server` is a simple HTTP server that dumps incoming requests for debugging purposes.
- `cmd/proxy` proxies HTTP requests to the Testkube API server for local development and debugging.
- `cmd/choco-stub` displays a deprecation message for the old Chocolatey package location.
- `cmd/tools` contains internal tooling for release management and version bumping.

### Regenerating artifacts

- Update the agent OpenAPI files with `make generate-openapi` after schema edits.
- Regenerate Kubernetes CRDs after editing type definitions in `api/` via `make generate-crds`.
- Regenerate SQL code when query files change via `make generate-sqlc`.
- Refresh mocks for new or updated interfaces using `make generate-mocks`.
- Build the Mongo to Postgres convert tool with `make build-convert` (also part of `make build`). Its image, `kubeshop/testkube-convert`, is the `convert` target in `docker-bake.hcl`, built from `build/new/convert.Dockerfile` and published by `.github/workflows/new-build.yaml` with the other images.

### Configuration references

- Agent behavior is driven by env vars defined in `internal/config/config.go` (scan for `envconfig:"..."` tags when researching a toggle).
- Helm chart values are the source of deployment defaults. For concrete local overrides, see `tilt-values.yaml` (repo root), which the Tilt setup uses; personal overrides go in the gitignored `tilt-values.local.yaml`. `build/_local/values.dev.tpl.yaml` is the template that Skaffold and `tk-dev` (in testkube-cloud-api, `js/scripts/tk-dev`) copy to the gitignored `build/_local/values.dev.yaml`.
- The convert tool reads its own env vars, not `internal/config`; `pkg/convert/doc.go` lists them.
- GitOps sync of Kubernetes resources into the Control Plane is gated by `GITOPS_KUBERNETES_TO_CLOUD_ENABLED` (default `false`), and additionally requires the Control Plane to report cloud storage support.
- TestTriggers accept `spec.event` or a `spec.events` list (mutually exclusive, validated service-side); always consume them via `EffectiveEvents()` so both forms are honored — classification gates that read the single `event` field directly will silently skip list-form triggers.
- Git trigger informer behavior is tuned via `TEST_TRIGGER_GIT_INFORMER_RECONCILE_INTERVAL`, `TEST_TRIGGER_GIT_INFORMER_REPO_DEPTH`, `TEST_TRIGGER_GIT_INFORMER_LIST_TIMEOUT`, `TEST_TRIGGER_GIT_INFORMER_MAX_COMMITS_SCAN`, `TEST_TRIGGER_GIT_INFORMER_PULL_RETRIES`, and `TEST_TRIGGER_GIT_INFORMER_PULL_RETRY_DELAY`.
- Git trigger informer execution is leader-gated in `cmd/api-server/main.go` through the shared `leader` coordinator tasks, so only the active leader performs periodic git pulls/reconciliation.
- Standalone execution dispatch is bounded by `EXECUTION_DISPATCH_BATCH_SIZE` (default `25`), the number of executions one `GetExecutionUpdates` poll hands to the runner. A dispatched execution is re-offered after `EXECUTION_REDISPATCH_AFTER` (default `60s`) if the runner never acknowledged it, and is failed explicitly after `EXECUTION_START_TIMEOUT` (default `10m`) by a reaper that runs every `EXECUTION_REAPER_INTERVAL` (default `1m`). `EXECUTION_START_TIMEOUT` must stay above `EXECUTION_REDISPATCH_AFTER` or the reaper fails executions before they are retried; the server clamps it if it is not. Raising the batch size trades a bounded poll for a larger one — it is what keeps a burst of workflows sharing a cron minute from exceeding the runner's call deadline.
- Runner execution polling is tuned via `RUNNER_POLL_INTERVAL` (default `1s`, a latency floor rather than a throughput cap — one poll returns a whole batch), `RUNNER_CALL_TIMEOUT` (default `30s`), `RUNNER_POLL_MAX_BACKOFF` (default `15s`, the ceiling on the retry delay after a failed poll), `RUNNER_START_CONCURRENCY` (default `4`, how many executions from one batch start at once), and `RUNNER_POLL_STALE_AFTER` (default `2m`, how long without a successful poll before `/ready` reports the runner unready). Keep `RUNNER_POLL_STALE_AFTER` above `RUNNER_CALL_TIMEOUT + RUNNER_POLL_MAX_BACKOFF` so one slow poll cannot flap readiness.
- Health vs readiness: `/health` answers unconditionally and is what the startup and liveness probes use; `/ready` evaluates registered checks and returns 503 with the failing check names. The api and runner charts point their readiness probes at `/ready`. Liveness deliberately stays on `/health` — a stalled poll loop usually means the control plane is unreachable, and restarting every runner over that is worse than the stall.
- `testkube-api` chart values `jobTolerations`/`jobAffinity`/`jobNodeSelector` (`k8s/helm/testkube-api/values.yaml`) set the `tolerations`/`affinity`/`nodeSelector` applied to the ephemeral pods spawned per test execution (`_job-template.yaml.tpl` for legacy prebuilt/container executors, `_slave-pod-template.yaml.tpl` for test-workflow slave pods). Unset (empty) by default and deliberately does not fall back to `global.tolerations`/`global.affinity`/`global.nodeSelector`, since those already carry a non-empty default (an arm64 toleration) that would otherwise silently change job/slave pod scheduling for every chart consumer.
- Telemetry opt-out: `DO_NOT_TRACK=1` (any value except empty, `0` or `false`) skips every telemetry send in the CLI and the agent (`pkg/telemetry`).
- CLI update-check toggle: set `TESTKUBE_DISABLE_UPDATE_CHECK=1` to suppress both the per-command hint and the `testkube version` status block. The CLI persists `lastUpdateCheckAt` and `latestKnownVersion` in `~/.testkube/config.json` to throttle the per-command hint to once per day.
- Object retention: `STORAGE_EXPIRATION` (whole bucket, opt-in) and `STORAGE_CACHE_EXPIRATION` (step dependency caches, default 1 day); `pkg/storage/minio/doc.go` explains the defaults and the permission the lifecycle read needs.

### Architecture reference

- See [`ARCHITECTURE.md`](ARCHITECTURE.md) for a detailed description of the agent's components, storage layer, event system, CRDs, CLI, and Kubernetes deployment.
- When making changes that affect the architecture (new entry points, storage backends, event listeners, CRDs, API routes, etc.), update `ARCHITECTURE.md` to keep it in sync.

### Package documentation

Each area below documents itself in the `doc.go` of its package; read that before changing it. A line that names another repository is a contract both ends change together.

- **MCP integration:** `pkg/mcp/doc.go`, with the Insights board rules in `pkg/mcp/boards/doc.go` and how the board tools write in `pkg/mcp/tools/doc.go`. A new tool is also registered in the Control Plane's `mcp_handler.go`, the Control Plane's `HandlerClient` uses `pkg/mcp/boards` rather than reimplementing it, and a dashboard change to the ported report files needs a matching change in `pkg/mcp/boards`.
- **GitOps resource sync** and its resource ownership contract: `internal/sync/doc.go`. The Control Plane enforces ownership in testkube-cloud-api's `internal/sync/ownership.go`.
- **Mongo to Postgres conversion** (`cmd/convert`): `pkg/convert/doc.go`. A schema migration touching `test_workflow_executions` or its child tables is a change there too.
- **Execution lineage and reruns:** `pkg/api/v1/testkube/doc.go`. Every path that builds an `ExecutionStart`, here and in testkube-cloud-api, sets the lineage from the execution record.
- **Step dependency cache:** `pkg/executioncache/doc.go`. The Control Plane imports `pkg/executioncache` rather than reimplementing it.
- **Object retention** (`STORAGE_EXPIRATION`, `STORAGE_CACHE_EXPIRATION`): `pkg/storage/minio/doc.go`.
- **Transient-failure retries:** `pkg/runner/doc.go` (resource cleanup), `pkg/event/kind/webhook/doc.go` (webhook delivery) and `cmd/tcl/testworkflow-toolkit/commands/doc.go` (services cleanup in the kill step).
- **Telemetry and cluster detection:** `pkg/telemetry/doc.go`, with the CLI runtime context and AI-tool detection in `pkg/cliruntime/doc.go`. `hostedRunnerNamePrefix` stays in sync with `naming.HostedRunnerAgentName` in testkube-cloud-api.
- **CLI update check, file input and `upgrade --reuse-values`:** `cmd/kubectl-testkube/commands/common/doc.go`.
- **CLI prompts and non-interactive runs:** `pkg/ui/doc.go`.
- **CLI diagnostics:** `pkg/diagnostics/doc.go`.
- **On-prem demo install** (`testkube init demo`) and the install lifecycle events: `cmd/kubectl-testkube/commands/doc.go`. The demo runner's identity matches the runner under `bootstrapConfig` in `values.demo.v2.yaml` (testkube-cloud-charts), and a new lifecycle event is allowlisted in the license worker's `/events` handler (testkube-infrastructure).

## 4. Conventions reviewers enforce

- Package behaviour goes in the package's `doc.go` (see "Keeping documentation in sync").
- Dependencies are injected through small interfaces, with the real implementation in the code and a fake in the test, not through package-level variables.
- Parse structured output (`--format json`) rather than the lines a tool happens to print; the line format changes between versions.
- Name constants for durations and limits (`const shutdownTimeout = 10 * time.Second`) rather than inlining them.
- A change in behaviour that a user or operator sees says in the pull request whether it is breaking, what to migrate, and whether the Helm chart needs a matching change (a new Secret, value or permission).
- `make lint` enforces the rest: `govet`, `revive`, `staticcheck`, `unused`, `ineffassign`, `noctx` (an HTTP request carries a context) and `goimports` formatting. `unused` is what catches a helper that was written but never called.

### Keeping documentation in sync

A package documents its own behaviour in its `doc.go`. Reviewers ask for that instead of a new `AGENTS.md` or `ARCHITECTURE.md` section, so write or update the `doc.go` first. After a code change, update these two files only when one of the following is true:

- **New or removed entry points** (`cmd/` binaries, API routes, controllers) → update both files.
- **Changed detection or identification logic** (cluster type, CLI run context, Docker context, AI tool) → update the package's `doc.go` (`pkg/telemetry`, `pkg/cliruntime`, `cmd/kubectl-testkube/commands/common`) and anything `ARCHITECTURE.md` says about it.
- **A contract with another repository or component changes** (a proto, the OpenAPI schema, something the Control Plane imports) → update both packages' `doc.go` and the line that names both ends under "Package documentation".
- **New storage backends, event listeners, CRDs, or external integrations** → update `ARCHITECTURE.md`.
- **New configuration knobs or environment variables** → mention them in `AGENTS.md` under "Configuration references" if they affect agent behavior.
- **New code-generation or build steps** → add them under "Regenerating artifacts" in `AGENTS.md`.
- **A new package an agent would not find by its name** → one line in `AGENTS.md` pointing to its `doc.go`, as the `pkg/localinstall` line under "Running locally" does.

### PR title format

PR titles **must** follow [Conventional Commits](https://www.conventionalcommits.org/) format with a type prefix. The `oss-lint-pr` workflow rejects PRs without one. Examples:

- `feat: Add soft-delete for workflow executions`
- `fix: Retry log stream on 502 errors`
- `chore: Add contextcheck linter`

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`. A scope is allowed (`fix(cli): ...`), and `!` marks a breaking change.

## 5. Never do these

- **Do not hand-edit generated code.** Change the source and run the generator (see "Regenerating artifacts"); the `oss-verify-*` workflows fail on any difference.
  - `pkg/api/v1/testkube/model_*.go`, from `api/v1/testkube.yaml` (`make generate-openapi`). The `model_*_extended.go` and `model_*_aliases.go` files, and the files without the `model_` prefix, are hand-written.
  - `pkg/proto/`, from `proto/` (`make generate-protobuf`).
  - `k8s/crd/` and `zz_generated.deepcopy.go` under `api/`, from the `api/` types (`make generate-crds`).
  - The `sqlc/` packages (`pkg/database/postgres/sqlc/`, `pkg/controlplane/scheduling/sqlc/`), from the queries (`make generate-sqlc`).
  - `mock_*.go`, from the `//go:generate mockgen` lines (`make generate-mocks`).
- **Do not edit `.golangci.yml` to silence a finding.** Fix the code; the file belongs to devops.
- **Do not run** `make dev`/`make run-api` expecting a working agent (see "Running locally"), `testkube install local` to test this checkout (it installs published images), or anything that publishes a release (`goreleaser`, the version and release tooling in `cmd/tools`).
- **Do not run the `oss-*` workflows against a revision that is not pushed to `kubeshop/testkube`.** They check the revision out from GitHub, so a local-only commit fails to resolve.

### Deprecated components — DO NOT reference

- **testkube-operator** (`k8s/helm/testkube-operator/`, `testkube-operator` Helm values): The Kubernetes operator is deprecated and disabled by default. Do not suggest enabling it, reference it in documentation, or add new code that depends on it. The Helm chart still carries it as a dependency for backwards compatibility only.

## 6. Pull request checks and the MCP tools

Pull request checks are not GitHub Actions. They are Testkube TestWorkflows in `test/testkube/oss-tests/gh-quality-loop.yaml`, started by the Testkube GitHub integration on every pull request. Each takes a `revision` config input (default `main`) and checks out `https://github.com/kubeshop/testkube.git` at that revision, so the branch must be pushed to this repository first.

| Workflow | What it runs |
| --- | --- |
| `oss-lint-go` | `golangci-lint run ./...` |
| `oss-lint-proto` | `buf lint`, `buf format --diff`, and `buf breaking` against `main` |
| `oss-unit-tests` | `make unit-tests` |
| `oss-integration-tests` | the `_Integration` tests with MongoDB, NATS, MinIO and Postgres |
| `oss-verify-crds` | `make verify-crds-generated` |
| `oss-verify-protobuf` | `make verify-protobuf-generated` |
| `oss-lint-pr` | the pull request title (see "PR title format"); also needs `PR_NUMBER`, so it only runs once the pull request exists |
| `security-pr-scan` | Semgrep, Gitleaks and Trivy on the changed files; `make security-scan` runs the same locally (needs Docker). Its definition is not in this repository, and it reports as the `testkube-dev` GitHub App rather than `testkubebot` |

The same file also holds `docs-prettier`, which checks testkube-docs, not this repo.

To run them yourself, use the Testkube MCP server (see "Package documentation") connected to the environment that runs these checks: the "GitHub Advanced Integration" environment of the Testkube OSS organization on Prod US (`app.testkube.io`). Either run `testkube mcp serve` from a CLI logged in to that environment, or use its remote endpoint, `https://api.testkube.io/organizations/<organization ID>/environments/<environment ID>/mcp`, taking the IDs from the environment's dashboard URL (`/organization/<organization ID>/environment/<environment ID>/`). Both need a member of that organization. Argo CD syncs `test/testkube/oss-tests/gh-quality-loop.yaml` there from `main` (testkube-deployment `envs/prd-agents/apps/testkube-runner-github-advanced-integration.yaml`), so a merged change to it applies to every pull request at once; the checks report as `testkubebot`. The dev Control Plane's agent syncs a copy of the same file, and `security-pr-scan` runs from there, reporting as `testkube-dev`.

Then:

1. `run_workflow` with `workflowName` and `config: {"revision": "<branch or commit SHA>"}`. It returns the execution ID.
2. `wait_for_executions` with `executionIds` (comma-separated, to wait on several runs at once).
3. On failure, `fetch_execution_logs` with `executionId` (narrow with `step`, `grep` or `tail`), and `list_artifacts` and `read_artifact` (`fileName`) for the JUnit reports such as `unit-tests.xml`.

Without MCP, `testkube run testworkflow <name> --config revision=<branch> --watch` does the same from a CLI logged in to that environment.

## 7. Skills and starting a component

### Running locally

Pick the smallest setup that exercises the change. Everything past building needs Docker and a local Kubernetes cluster; the Makefile accepts only Linux and macOS, so on Windows use WSL.

In a `tk-sbx` sandbox, Testkube is already running and there is no Docker: build there (`make build`), then deploy with the `testkube-hot-reload` skill.

- **CLI:** `make build-kubectl-testkube-cli` builds `bin/app/kubectl-testkube`. `make build` builds every binary into `bin/app/`.
- **Agent alone (OSS, no Control Plane):** the Tilt setup in [`DEVELOPMENT.md`](DEVELOPMENT.md). `./scripts/tilt-cluster.sh` creates the k3d cluster `testkube-dev` with a registry on `localhost:5001`, then `tilt up` builds the API server, `testworkflow-init` and `testworkflow-toolkit` from this checkout and installs `k8s/helm/testkube` into namespace `testkube-dev` with Postgres, MinIO and NATS. The API is on `localhost:8088`. `tilt ci` runs the same setup headless, runs the smoke test and exits non-zero on failure, so it is the one to use without a browser. `tilt up -- --db=mongo` switches the database.
- **`testworkflow-init` and `testworkflow-toolkit`** only run inside a TestWorkflow pod. Exercise them through the Tilt setup by running a workflow; its `install-sample-workflows` resource applies the smoke workflows from `test/*/crd-workflow/smoke.yaml`.
- **Skills:** the stack skills are in testkube-workspace (below). For writing or running the TestWorkflows under `test/`, the public skills in [testkube-skills](https://github.com/kubeshop/testkube-skills) (`testworkflow-author`, `testworkflow-runner`) know the schema and the CLI.
- **Agent with a Control Plane (full stack):** this repo cannot start a Control Plane. Use [testkube-workspace](https://github.com/kubeshop/testkube-workspace), which builds this checkout and a `testkube-cloud-api` checkout side by side: `tilt up oss` (agent alone), `tilt up cp` (Control Plane, dashboard and a connected agent) or `tilt up agent` (local agent, remote Control Plane). Its `tilt-local-dev` and `local-e2e-verify` skills cover the bring-up and how to verify a change against the running stack. It replaces `tk-dev`.
- **Agent against a sandbox Control Plane:** to check an agent change end to end without a local Control Plane, follow the `start-component` skill. It takes the sandbox's agent endpoint, organization, environment and tokens as inputs, starts the agent from this checkout with `tilt up agent`, checks it connected, runs one workflow on it and tears it down.
- **MCP server:** `kubectl-testkube mcp serve` needs a CLI context logged in to a Control Plane (organization, environment and a token), or `TK_MCP_ENV_MODE=true` with `TK_ACCESS_TOKEN`, `TK_ORG_ID`, `TK_ENV_ID`, `TK_CONTROL_PLANE_URL` and `TK_DASHBOARD_URL`. It cannot drive the OSS Tilt agent, whose context uses `--client direct`.
- **Not a dev loop:**
  - `make dev` and `make run-api` start the API server outside a cluster with no database DSN and no MinIO, so it does not come up usable.
  - `testkube install local` (`pkg/localinstall`, see its `doc.go`) installs pinned, published charts and images for trial users, so it never runs the code in this checkout. `testkube status local` and `testkube uninstall local` sit beside it, and pointing the CLI at that install (after install, `testkube set context local`, and on uninstall) lives in `cmd/kubectl-testkube/commands/common/localcontext.go`.
