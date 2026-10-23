# Working with Testkube Core (testkube)

Testkube Core: the agent services that run in a cluster, the `kubectl-testkube` CLI, and the shared Go packages and API contract the Control Plane builds on.

Owner: `@kubeshop/testkube-backend`, per `.github/CODEOWNERS`. `test/` is shared with `@kubeshop/testkube-qa`; CI, release and lint configuration (`.github/`, `goreleaser_files/`, `.goreleaser*.yml`, `.golangci.yml`, `.builds-*`) belong to `@kubeshop/testkube-devops`.

## 1. Verifying a change

Verification has two loops. Run these local commands before every commit, and the pull request workflows in section 6 before asking for review.

```bash
make lint          # golangci-lint (or `make lint-fix` to auto-fix); about 2m in CI
make compile       # go build ./..., compilation only
make unit-tests    # what oss-unit-tests runs, about 2m in CI; add GOFLAGS=-count=1 to skip cached results
```

Use these make targets rather than calling `go build` or `go test` directly: `oss-unit-tests` runs `make unit-tests`, and the test targets carry the pinned `gotestsum` and the environment the integration tests expect (`INTEGRATION`, `TESTKUBE_PROJECT_ROOT`, the MinIO credentials). `make unit-tests` and `make integration-tests` take `PKG=` (packages) and `RUN=` (a `go test -run` pattern) to narrow a run.

Run the narrower checks when the change touches what they cover:

- `make verify-protobuf-generated` after editing `proto/`, and `make verify-crds-generated` after editing `api/` types (see "Regenerating artifacts").
- `make integration-tests` runs only tests ending in `_Integration`. It expects MongoDB on `:27017`, NATS on `:4222`, MinIO on `:9000` (`minio99`/`minio123`) and Postgres on `:5432` (`API_POSTGRES_URL=postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable`); `make run-mongo run-nats` starts only the first two, as Docker containers, so it is for a developer host only.
- `make integration-tests` also runs the toolkit tests (`test/integration/components/toolkit`), which `oss-integration-tests` (section 6) skips, so no pull request check covers them. They need a disposable cluster in the current kube context, such as the Tilt `k3d-testkube-dev` ("Running locally", section 7), never a shared one: each test creates and deletes a `testkube-test-<n>` namespace with `busybox`, `python` and `golang` pods. Run only them with `make integration-tests PKG=./test/integration/components/toolkit/ GOFLAGS=-count=1`.

`make lint` and `oss-lint-go` both run golangci-lint `v2.15.0`: `GOLANGCI_LINT_VERSION` in the `Makefile`, and the image in `test/testkube/oss-tests/gh-quality-loop.yaml`. Bump both together, along with the version in that workflow's cache key, or the two can disagree; the workflow is the one that gates the pull request.

- Review the Makefile for additional helper targets when unfamiliar tasks come up.

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
- `cmd/convert` is a one-shot tool that migrates control-plane data from MongoDB to PostgreSQL (see "Mongo to Postgres conversion" below).
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
- Helm chart values are the source of deployment defaults. For concrete local overrides, see `build/_local/values.dev.tpl.yaml`: `tk-dev` (in testkube-cloud-api, `js/scripts/tk-dev`) deploys the agent with a copy of it at `build/_local/values.dev.yaml`, which is gitignored.
- The convert tool does not use `internal/config`. Each flag in `cmd/convert/main.go` falls back to an env var: it reuses the API server's `API_MONGO_*`, `API_POSTGRES_DSN`, `SKIP_DB_CREATION` and `DISABLE_POSTGRES_MIGRATIONS`, and adds its own `CONVERT_BATCH_SIZE`, `CONVERT_READ_BATCH_SIZE`, `CONVERT_DRY_RUN`, `CONVERT_RESET`, `CONVERT_RESET_CONFIRMED`, `CONVERT_SKIP_ERRORS`, `CONVERT_SKIP` and `CONVERT_VERIFY`. MongoDB TLS material is passed as file paths, because the tool has no cluster client to read a Secret with.
- GitOps sync of Kubernetes resources into the Control Plane is gated by `GITOPS_KUBERNETES_TO_CLOUD_ENABLED` (default `false`), and additionally requires the Control Plane to report cloud storage support.
- TestTriggers accept `spec.event` or a `spec.events` list (mutually exclusive, validated service-side); always consume them via `EffectiveEvents()` so both forms are honored — classification gates that read the single `event` field directly will silently skip list-form triggers.
- Git trigger informer behavior is tuned via `TEST_TRIGGER_GIT_INFORMER_RECONCILE_INTERVAL`, `TEST_TRIGGER_GIT_INFORMER_REPO_DEPTH`, `TEST_TRIGGER_GIT_INFORMER_LIST_TIMEOUT`, `TEST_TRIGGER_GIT_INFORMER_MAX_COMMITS_SCAN`, `TEST_TRIGGER_GIT_INFORMER_PULL_RETRIES`, and `TEST_TRIGGER_GIT_INFORMER_PULL_RETRY_DELAY`.
- Git trigger informer execution is leader-gated in `cmd/api-server/main.go` through the shared `leader` coordinator tasks, so only the active leader performs periodic git pulls/reconciliation.
- Standalone execution dispatch is bounded by `EXECUTION_DISPATCH_BATCH_SIZE` (default `25`), the number of executions one `GetExecutionUpdates` poll hands to the runner. A dispatched execution is re-offered after `EXECUTION_REDISPATCH_AFTER` (default `60s`) if the runner never acknowledged it, and is failed explicitly after `EXECUTION_START_TIMEOUT` (default `10m`) by a reaper that runs every `EXECUTION_REAPER_INTERVAL` (default `1m`). `EXECUTION_START_TIMEOUT` must stay above `EXECUTION_REDISPATCH_AFTER` or the reaper fails executions before they are retried; the server clamps it if it is not. Raising the batch size trades a bounded poll for a larger one — it is what keeps a burst of workflows sharing a cron minute from exceeding the runner's call deadline.
- Runner execution polling is tuned via `RUNNER_POLL_INTERVAL` (default `1s`, a latency floor rather than a throughput cap — one poll returns a whole batch), `RUNNER_CALL_TIMEOUT` (default `30s`), `RUNNER_POLL_MAX_BACKOFF` (default `15s`, the ceiling on the retry delay after a failed poll), `RUNNER_START_CONCURRENCY` (default `4`, how many executions from one batch start at once), and `RUNNER_POLL_STALE_AFTER` (default `2m`, how long without a successful poll before `/ready` reports the runner unready). Keep `RUNNER_POLL_STALE_AFTER` above `RUNNER_CALL_TIMEOUT + RUNNER_POLL_MAX_BACKOFF` so one slow poll cannot flap readiness.
- Health vs readiness: `/health` answers unconditionally and is what the startup and liveness probes use; `/ready` evaluates registered checks and returns 503 with the failing check names. The api and runner charts point their readiness probes at `/ready`. Liveness deliberately stays on `/health` — a stalled poll loop usually means the control plane is unreachable, and restarting every runner over that is worse than the stall.
- Helm chart values are the source of deployment defaults; `tilt-values.yaml` (repo root) shows the local overrides the Tilt setup uses if you need a concrete reference. `build/_local/values.dev.tpl.yaml` is the template for the Skaffold setup, which copies it to the gitignored `values.dev.yaml`.
- `testkube-api` chart values `jobTolerations`/`jobAffinity`/`jobNodeSelector` (`k8s/helm/testkube-api/values.yaml`) set the `tolerations`/`affinity`/`nodeSelector` applied to the ephemeral pods spawned per test execution (`_job-template.yaml.tpl` for legacy prebuilt/container executors, `_slave-pod-template.yaml.tpl` for test-workflow slave pods). Unset (empty) by default and deliberately does not fall back to `global.tolerations`/`global.affinity`/`global.nodeSelector`, since those already carry a non-empty default (an arm64 toleration) that would otherwise silently change job/slave pod scheduling for every chart consumer.
- Telemetry opt-out: `DO_NOT_TRACK=1` (any value except empty, `0` or `false`) skips every telemetry send in the CLI and the agent (`pkg/telemetry`).
- CLI update-check toggle: set `TESTKUBE_DISABLE_UPDATE_CHECK=1` to suppress both the per-command hint and the `testkube version` status block. The CLI persists `lastUpdateCheckAt` and `latestKnownVersion` in `~/.testkube/config.json` to throttle the per-command hint to once per day.
- Object retention is driven by `STORAGE_EXPIRATION` (whole bucket, in days) and `STORAGE_CACHE_EXPIRATION` (step dependency caches under the `.tkcache/v1` prefix, in days). **`STORAGE_CACHE_EXPIRATION` defaults to 1 day; `STORAGE_EXPIRATION` stays opt-in.** The difference is the filter, not taste: the cache rule is confined to the cache prefix and can only delete caches, where the bucket-wide rule is unfiltered and governs artifacts and logs too, so a default there would delete a deployment's results on an upgrade. `TestExpirationDefaults` pins both. `SetExpirationPolicies` in `pkg/storage/minio/minio.go` applies them with `SetBucketLifecycle`, which replaces the bucket lifecycle wholesale - so it reads the existing configuration first and carries through every rule Testkube does not own, matched by ID (`mergeLifecycleRules`). That merge is what makes a default safe at all: without it, defaulting either setting would drop the rules of installations whose bucket lifecycle is managed elsewhere, purely by upgrading. It fails closed - if the existing lifecycle cannot be read, nothing is written. That read is a permission earlier versions did not need (`s3:GetLifecycleConfiguration` on S3 and MinIO, `storage.buckets.get` on GCS), and because `STORAGE_CACHE_EXPIRATION` now defaults to 1 the call happens on every installation rather than only those configuring an expiration - so an upgrade can need permissions the deployment never granted. The error names them. Note also that the bucket-wide rule is unfiltered and so covers cache objects too, and the earlier expiration wins — a cache TTL can only bring eviction forward, never postpone it, and `MustGetMinioClient` warns when it is set longer than the bucket-wide one.

### Architecture reference

- See [`ARCHITECTURE.md`](ARCHITECTURE.md) for a detailed description of the agent's components, storage layer, event system, CRDs, CLI, and Kubernetes deployment.
- When making changes that affect the architecture (new entry points, storage backends, event listeners, CRDs, API routes, etc.), update `ARCHITECTURE.md` to keep it in sync.

### MCP integration

- `pkg/mcp/` implements the Model Context Protocol server for AI assistant integration.
- Exposes tools across workflows, executions, artifacts, and metadata via `testkube mcp serve` (CLI), Docker image (`testkube/mcp-server`), or Control Plane's `/mcp` endpoint per environment.
- Uses interface-based tool design; new tools need registration in both `pkg/mcp/server.go` and control plane's `mcp_handler.go`.
- See `pkg/mcp/README.md` for architecture, tool patterns, and usage examples. Section 6 shows how to run the pull request workflows on a branch through these tools.
- Insights board tools (`pkg/mcp/tools/boards.go`) keep their rules in `pkg/mcp/boards/`: report param validation and defaults, the report-to-`/insights/*` query translation, and the layout. It is a port of the dashboard's TypeScript (`utils/insights.ts`, `DynamicFilters/types.ts`, `reports/*/type.ts` in `testkube-cloud-api/js/packages/web`), since the Control Plane stores report params opaquely. **The Control Plane's `HandlerClient` must use this package rather than reimplement it**, and a dashboard change to those files needs a matching change here; `testdata/translation_cases.json` pins the translation.
- Boards are organization-scoped and the Control Plane refuses API tokens on every board endpoint, so the board tools need a user session. `APIClient` refuses a `tkcapi_` token before sending anything and returns `tools.ErrBoardsRequireUser`. Every board write reads the board first and resends its description. Current Control Planes keep a description an update omits, but older ones clear it, so resending is what keeps it on those.
- **Board updates are optimistic-concurrency writes.** Resending a value read earlier (the description, a recomputed layout) would overwrite a concurrent edit, so every update - `update_board` and the three report tools - goes through `writeBoard` in `pkg/mcp/tools/boards.go`: it sends `expectedVersion` (the board `version` it read; every write to a board increments it), the Control Plane refuses a stale write with 409, both clients turn that into `tools.ErrBoardChanged`, and the write is rebuilt from a fresh read, up to `boardWriteAttempts` times. A write's builder must derive everything from the board it is handed, never from an earlier read. The token is a counter, not `updatedAt`: two writes can share a timestamp, and a reused token would let a stale write through. A Control Plane that predates versions returns none, and the write is then unconditional. `delete_board` is deliberately not conditional: it resends nothing it read (the read only resolves a slug to the ID it deletes by), and the Control Plane checks visibility and delete rights against the board as it is at delete time, so deleting removes the board whatever changed since the read, as deleting in the dashboard does.
- **Relative report ranges are anchored in a time zone.** The dashboard ends a `day`/`week`/`month`/`quarter` range at the viewer's local midnight, so `render_board` takes an IANA `timeZone` (default UTC) and passes it as `boards.QueryOptions.Location`; `boards` embeds `time/tzdata` because the MCP also runs from images without a zoneinfo database.

### GitOps resource sync

- `internal/sync/` implements the agent side of the GitOps (Kubernetes → Control Plane) sync capability. Reconcilers live in `internal/sync/controller/`, one per syncable kind, and the Control Plane client lives in `internal/sync/grpc/`.
- The syncable kinds are `TestWorkflow`, `TestWorkflowTemplate`, `TestTrigger`, `WorkflowTrigger`, `Webhook`, and `WebhookTemplate`. Adding a kind means touching `internal/sync/controller/`, `internal/sync/grpc/`, the `SyncService` proto, and the Control Plane together.
- The sync controllers are registered in `cmd/api-server/main.go` behind `proContext.CloudStorageSupportedInControlPlane` and `GITOPS_KUBERNETES_TO_CLOUD_ENABLED`, so they only run for a GitOps-persona agent connected to a Control Plane.

#### Resource ownership contract

A synced resource in the Control Plane is exclusively owned by one GitOps agent, so that agents syncing from different namespaces cannot silently overwrite each other's resources. The contract spans four layers, and a change to one usually needs a matching change in the others:

1. **Schema** — `Syncable.gitOpsOwner` in `api/v1/testkube.yaml`, a `GitOpsOwner` holding the authoritative `agentId` plus a display-only `agentName` that may be stale after a rename. Run `make generate-openapi` after editing.
2. **Wire** — the Control Plane rejects a sync from a non-owning agent with the gRPC status `codes.FailedPrecondition`. That code is reserved for ownership conflicts on the sync API; do not reuse it there for other validation failures, or agents will misreport the cause.
3. **Agent translation** — `translateError` in `internal/sync/grpc/errors.go` maps `FailedPrecondition` onto the `ErrOwnershipConflict` sentinel in `internal/sync/errors.go`, keeping the original status in the error chain because its message names the current owner. Callers match on the sentinel so that no layer above the client depends on gRPC.
4. **Reconciliation** — `terminalOnRejection` in `internal/sync/controller/errors.go` wraps a rejection (an ownership conflict or an invalid resource, see `syncagent.IsRejection`) in `reconcile.TerminalError` so controller-runtime stops requeueing something no retry can fix, while every other error stays retryable with backoff. Rejections are deliberately not logged there: controller-runtime already logs whatever a reconciler returns, along with the kind and name of the resource.

`skipRejectedResource` in `cmd/api-server/superagentmigration.go` applies the same rule outside the reconcilers, during SuperAgent migration. That migration retries sync failures forever by design, so a rejection has to break the loop or a single unowned or invalid resource wedges the migration indefinitely.

The Control Plane side lives in `testkube-cloud-api`: it persists the owner on the six syncable tables and enforces it in `internal/sync/ownership.go`, on every Upsert and Delete path. Enforcement is opt-in per organization (the `gitOpsOwnership` setting), so with it off an agent never sees a conflict. Ownership is transferred from Git by setting the `testkube.io/gitops-owner` annotation on the Kubernetes resource to the new owner's agent ID. The agent sends the annotation with the resource, and the Control Plane reads it, so the agent needs no code for it. The agent name is display-only and never authorizes anything.

### Mongo to Postgres conversion

`cmd/convert` (the CLI and flags) and `pkg/convert` (the migrator) move an OSS installation's
data from MongoDB to PostgreSQL, so that switching the API server from `API_MONGO_DSN` to
`API_POSTGRES_DSN` keeps execution history and numbering instead of starting empty and
restarting execution numbers at 1, which would collide with the names of old executions.

- **Two tasks, and nothing else.** `executions` copies `testworkflowresults` into the seven
  `test_workflow_*` tables (`executions.go`, `executions_row.go`). `sequences` copies Test
  Workflow counters into `execution_sequences` (`sequences.go`), and never moves a counter
  backwards. Logs, outputs and artifacts are in object storage, definitions and triggers are
  CRDs, and the `triggers` collection holds only a short-lived leader lease, so none of them
  are migrated.
- **Rows are written by hand as COPY text**, not through the repository. That makes every
  schema migration touching `test_workflow_executions` or its child tables a change here
  too: add the column to the matching `*Columns` list and `write*Row` serializer in
  `executions_row.go`, projected the way `pkg/repository/testworkflow/postgres` writes it. A
  column the converter leaves out is silently NULL for every migrated row.
  `TestConvertExecutions_Integration` reads each execution back through both repositories and
  compares them, which catches the omission **only if the fixture in `buildExecution` sets
  the field**. Both repositories synthesize some fields on read (lineage through
  `EffectiveLineage()`), so a fixture that leaves such a field unset round-trips even when
  the column was dropped.
- **Resumable and exactly-once.** Each batch commits in the same transaction as its row in
  `convert_checkpoints` (migration `20260826120000_convert_checkpoints.sql`, written only by
  this tool), so an interrupted run resumes after the last committed batch. `--reset --yes`
  truncates the target and clears the checkpoints. `--dry-run` serializes everything and
  writes nothing, but still applies schema migrations, because the checkpoint lookup and the
  verification read those tables.
- **The tool migrates the schema itself**, with goose's `WithAllowOutofOrder(true)` like
  the API server, and an out-of-date schema is fatal here rather than a warning: the COPY
  statements name columns that may not exist yet.
- **Exit status.** After the summary, the tool exits non-zero if any document failed or the
  run raised a warning, such as a verification count mismatch (`ErrIncomplete`), so a Job is marked failed.
- **Deployment.** The `testkube` chart runs it as the `convert` Job
  (`templates/convert-job.yaml`, `convert.*` values), disabled by default. It is deliberately
  not a Helm hook: the operator triggers the cutover. The Job reads both DSNs from
  `testkube-api.mongodb` and `testkube-api.postgresql`, and a retry resumes from the
  checkpoint.

### Execution lineage and reruns

- `TestWorkflowExecutionLineage` (`baseId`, `rootId`, `attempt`) records what an execution is a rerun of. It is written for **every** execution, not only reruns: an original run is its own root at attempt 1, so "every execution of chain R" is one predicate and includes the original - in SQL `COALESCE(lineage_root_id, id) = R`, matching the chain index, because a legacy row carries NULL there and is its own root. `TestWorkflowExecution.EffectiveLineage()` synthesizes that default for rows written before the columns existed, which is why nothing has to be backfilled - derive it there and nowhere else, or the two repositories will disagree.
- It travels `ScheduleRequest.base_execution_id` -> `Enqueuer.deriveLineage` -> the execution record -> `ExecutionStart.lineage` -> `ExecutionConfig.Lineage` -> `RerunExecutionId()` in the pod, which is what makes the reserved `execution("rerun")` reference resolve. **Every path that builds an `ExecutionStart` must set it from the execution record**, in this repo and in `testkube-cloud-api`: a writer that forgets sends the pod no lineage, `execution("rerun")` silently stops resolving, and nothing reports it. That has already happened once - `go build` cannot catch it, because an unused helper function is not a compile error; the `unused` linter is the guard.
- Only the base id travels on the wire. The Control Plane derives the root and attempt by loading the base through the environment-scoped results repository, which is also where it confirms the base belongs to the caller's environment - `proto/service.proto` requires that, and the load is what enforces it. A caller able to assert a root or an attempt could forge a chain.
- **An original run has an empty `baseId` and must resolve to no rerun.** Returning `rootId` there would make every execution a rerun of itself.
- **Reserved references win over the registry.** `execution("parent")` and `execution("rerun")` are resolved before the executions the workflow scheduled, so a child aliased - or a workflow named - `parent`/`rerun` cannot shadow them. A collision is refused rather than resolved either way, because preferring the reserved meaning would instead make that child unreachable by name. `IsReservedRef` is the list.

### Transient-failure retries

- `pkg/runner/runner.go` runs `worker.Destroy` (cleanup of the execution's Secrets/Pods after the workflow ends) through the shared `retry()` helper via `destroyResources`. Bounded by `CleanupResourcesRetryCount` and `CleanupResourcesRetryDelay`; a brief `kube-apiserver` blip during teardown should not leave orphan resources in the customer namespace.
- `pkg/event/kind/webhook/listener.go` retries the outbound `HttpClient.Do` for `sendRetryCount` attempts with a linear `sendRetryBaseDelay`. Retryable outcomes: network errors, `5xx`, and `429`. Other `4xx` short-circuit so a bad URL / auth failure is not spammed at the subscriber. Delivery is intentionally at-least-once (subscribers own dedupe, matching Stripe/GitHub/Slack convention).
- `cmd/tcl/testworkflow-toolkit/commands/kill.go` runs the services cleanup of the `kill` step: `worker.List`, each `getServiceHealth` pod read, and `worker.DestroyGroup`, through the `retryTransient` helper and the classifiers in `retry.go`, bounded by `cleanupRetryCount` and `cleanupRetryDelay`. Only a failure whose every branch is transient is downgraded to a warning; the test steps already have their result, so a teardown blip is not the test's verdict. Anything else still fails the step. `retry.go` documents which error classes count as transient and why.

### Step dependency cache

A step's `cache` block (`api/testworkflows/v1/step_types.go`, `StepCache`) restores
directories from object storage before the step runs and saves them back once it passes,
so dependency installs survive between executions.

- `pkg/executioncache/` is the transport-free core: the object-key derivation
  (`objectkey.go`), the restore-key match policy (`match.go`), the payload and handshake
  shapes both sides share (`args.go`), and the repository interface plus its
  degrade-to-miss classification (`repository.go`). **The Control Plane must import this
  package rather than reimplement it** — a disagreement produces entries the other side
  can never find, so every run silently misses its own cache.
- `ProcessCacheRestore` / `ProcessCacheSave` in
  `pkg/testworkflows/testworkflowprocessor/operations_cache.go`, registered in **both**
  presets. Restore sits after the content operations so the repository is checked out
  when the key is resolved; save sits after the step's work and before artifacts.
- `cmd/testworkflow-toolkit/commands/cache.go` is the pod-side half. It never exits
  non-zero: a cache is an optimization, so a miss, an unreachable Control Plane, a
  missing capability, a corrupt archive or a refused upload all leave the step to install
  from the network.

Three constraints are easy to break and worth knowing before editing any of it:

- **The specification travels base64-encoded in one argument.** `testworkflow-init`
  resolves every container argument with `expressions.FinalizerFail` and exits the step
  on failure, so a key holding `hash_files()` over an absent lockfile would kill the step
  rather than miss the cache. `TestProcessCache_KeyTemplateStaysOpaque` guards this.
- **The two stages hand the resolved key over through `TK_CACHE_STATE`** on the shared
  `/testkube` volume instead of each computing it. They are separate containers, and an
  install may rewrite the very lockfile the key hashes (`npm ci` does), so a recomputed
  key could store the entry where nothing later searches for it.
- **Cached paths are mounted automatically.** Each stage is its own container, and
  containers share volumes but not their root filesystems, so a path outside every volume
  is restored where the container running the install cannot see it. `mount: false` on an
  uncovered path is refused at bundle time rather than silently doing nothing.

The save stage sets no condition, inheriting `passed` — deliberately unlike the artifacts
stage, which is `always`: publishing a failed install under a content-hash key would
poison every later run with no way for a user to invalidate it.

`hash()` (`pkg/expressions/stdlib.go`) and `hash_files()`
(`pkg/expressions/libs/fs.go`) exist for building keys. Prefer `hash_files`:
`hash(glob(...))` digests the matched **paths**, so it does not change when a file's
contents do.

### Telemetry and cluster detection

- `pkg/telemetry/` contains all telemetry event construction, sending, and cluster identification logic.
- `pkg/telemetry/cluster_type.go` implements Kubernetes cluster type detection using a layered approach (node providerID → node labels → server version → kube-system pod names). The result is cached with `sync.Once`.
- When adding support for a new cluster type, add detection entries to the appropriate layer(s) in `cluster_type.go` and add corresponding test cases in `cluster_type_test.go`.
- Every `Sender` returns the body it transmitted (Segment: the queued `Track` events as JSON), and `sendData` returns them one sender per line, so users can inspect telemetry before consenting: the CLI prints it as "telemetry send event response" under `--verbose`. A new sender must return what it sends, not a response status alone.
- `cmd/api-server/services/telemetry.go` drives the heartbeat loop that sends `testkube_api_heartbeat` events hourly, including the detected cluster type and agent capabilities.
- `cmd/api-server/services/capabilities.go` extracts agent capability tags (persona, mode, feature flags) from the runtime config for inclusion in telemetry events. When adding new agent features/toggles that should be tracked, add them here and in `capabilities_test.go`.
- The `hosted-runner` tag marks runners that Testkube provisions for trial users, detected from the `tkcagent_hr_` prefix the control plane assigns to `RUNNER_NAME`. Keep `hostedRunnerNamePrefix` in sync with `naming.HostedRunnerAgentName` in `testkube-cloud-api`.
- `pkg/cliruntime/context.go` is a leaf package containing the CLI runtime-context helpers (`IsRunningInDocker`, `DockerContext`, `CliRunContext`, `DetectAITool`). `pkg/telemetry` and `cmd/kubectl-testkube/commands/common` both depend on it; placing it in its own package avoids an import cycle between common and telemetry.
- `DetectAITool` reports the AI coding agent that invoked the CLI (`claude-code`, `codex`, `cursor`, `gemini-cli`, or `""`) purely from env vars the agents set in their subprocesses. Telemetry surfaces it as the `ai_tool` param (Segment property `aiTool`) across all CLI-origin payloads (`NewCLIPayload`, `NewCLIWithLicensePayload`, the inline error/preview payloads in `telemetry.go`, and the MCP tool payload in `mcp.go`).

### CLI update check

- `cmd/kubectl-testkube/commands/common/update_check.go` implements `MaybeNotifyNewerRelease` (per-command post-run hint) and `CheckComponentsStatus` (richer per-component report rendered by `testkube version`). Both consult `pkg/cliruntime` to skip in CI/Docker/Kubernetes contexts and honor the `--output` flag and `TESTKUBE_DISABLE_UPDATE_CHECK` env opt-out.
- `cmd/kubectl-testkube/commands/common/install_source.go` classifies how the running CLI binary was installed (Homebrew, Chocolatey, APT, install.sh, Docker, `go install`, unknown) by inspecting the resolved `os.Executable` path and the Docker context. The classification drives the install-source-specific upgrade command surfaced in the hint.
- Adding a new install channel: extend `DetectInstallSource` and add a test case to `install_source_test.go` that exercises the new path under the relevant `goos`.
- Adding a new CI/runtime detection: extend `pkg/cliruntime/context.go` so both telemetry and the update-check feature stay in sync.
- Adding a new AI-tool detection: extend `DetectAITool` in `pkg/cliruntime/context.go` (add the env-var check and a `TestDetectAITool` case in `context_test.go`); no telemetry wiring changes are needed since payloads already read the `AITool` field.

### CLI prompts and non-interactive runs

- `pkg/ui/interactive.go` holds the terminal check every prompt goes through. `ui.Select`, `ui.Confirm` and `ui.TextInput` call `requireInteractive` first and exit with an actionable message when stdin is not a terminal, so no call site has to guard itself. `ui.StdinIsInteractive()` exposes the same check for callers that want to take a different path instead of failing.
- The guard exists because a prompt with nobody to answer it used to spin: `atomicgo.dev/keyboard` cannot open a non-terminal stdin, reports every failed read as an empty keypress with a nil error, and pterm has no case for an empty keypress, so its listener loops forever and burns more than a core.
- `GetClient` (`cmd/kubectl-testkube/commands/common/client.go`) uses `ui.StdinIsInteractive()` to refuse starting a login when a token refresh fails with no terminal present, the same way the email-link branch beside it already does.
- Adding a new prompt: use the `ui` helpers rather than pterm directly, and it inherits the guard. Adding a command that must not prompt at all: branch on `ui.StdinIsInteractive()`.
- Tests flip the unexported `stdinIsInteractive` seam rather than allocating a pseudo terminal.

### CLI file input

- A flag that names a file to read takes `-` for the command's standard input (`common.ReadFileOrStdin` in `cmd/kubectl-testkube/commands/common/input.go`), so content can be piped: `cat payload.json | testkube create webhook --payload-template -`. Read new file flags through it, and use `cmd.InOrStdin()` rather than `os.Stdin` so tests can supply the input.

### CLI upgrade

- `testkube upgrade --reuse-values` passes Helm's `--reuse-values`. Helm applies `--set` over reused values, so in that mode `prepareCommonHelmArgs` sends only the settings of flags the user passed (`HelmOptions.ExplicitFlags`, `sendsFlag`/`sendsAnyFlag` in `common/helper.go`): a flag default such as `--no-postgres=true` would otherwise silently turn off what the release had enabled. A new flag-derived Helm setting must go through `sendsFlag`. Never gate on a value being non-empty: `ProcessMasterFlags` composes a runner's control plane addresses from default prefixes and root domain, so they always are, and sending them would point a runner of a custom control plane at SaaS. The flag lists (`agentURIFlags`, `uiURIFlags`) must name every flag `ProcessMasterFlags` reads for an address, the `--*-uri-override` flags included, or that flag is silently ignored. A runner's disabled MinIO/MongoDB/PostgreSQL is always sent.

### CLI diagnostics

- `testkube diagnostics` runs validator groups from `pkg/diagnostics`; the commands live in `cmd/kubectl-testkube/commands/diagnostics/`. Groups: `install` (`validators/deps`), `agent` (`validators/agent`, the agent pods are ready: `api-server`, `testkube-api` or `testkube-runner`) and `license`.
- Adding a check: implement `validators.Validator` in a package under `pkg/diagnostics/validators/`, register it in the group's `Register*Validators`, and build commands through `diagnostics.New(cmd)` in that package so `--output json` applies.
- A validator that needs the cluster takes a connect function rather than a client, so an unreachable cluster is reported as a result (`WithBreak()`) instead of aborting the run. Do not add checks for the deprecated operator.

### On-prem demo install

- `testkube init demo` (`cmd/kubectl-testkube/commands/init.go`) installs the On-Prem demo on the new architecture: the Control Plane (enterprise chart + `values.demo.v2.yaml`) plus a **separate** listener-enabled runner (`kubeshop/testkube-runner`). The bundled agent is gone.
- The CLI generates one agent secret key per install (`common.GenerateDemoAgentSecretKey`) and passes the *same* key to both sides — injected into the Control Plane's `bootstrapConfig` runner so the CP provisions it, and into the runner install (`common.HelmUpgradeOrInstallTestkubeOnPremDemoRunner` → `demoRunnerHelmOptions`). No secret is baked into the binary or chart.
- The runner identity (`demoRunnerID`/`OrgID`/`EnvID`) must stay in sync with the runner declared under `bootstrapConfig` in `values.demo.v2.yaml` (in `testkube-cloud-charts`).
- The legacy `values.demo.yaml` profile (bundled agent, MongoDB) is deprecated but kept for older CLIs.
- Install lifecycle telemetry: `init demo` and `install local` report `cli_install_started` (before Helm install; after the license step for `install local`) and `cli_install_finished` (after success) to the license service at `POST https://license.testkube.io/events`. The client lives in `pkg/diagnostics/validators/license/client.go` (`Client.ReportEvent`, `EventRequest`, the `LicenseEventsURL` and `EventCLIInstall*` constants); the `reportLicenseEvent`/`waitLicenseEvents` helpers in `init.go` make delivery telemetry-gated (`install local` also skips them when its install tracker is off, e.g. `DO_NOT_TRACK`), non-blocking (background goroutine), and flushed before exit with a bounded wait. The license key in the body is the credential (validated worker-side before recording), so no shared secret ships in the CLI. Adding a new lifecycle event: add an `EventCLIInstall*` constant and a `reportLicenseEvent` call, and allowlist it in the license worker's `/events` handler (`testkube-infrastructure`). `init demo` also sends `init_demo_started`/`init_demo_finished` to the PostHog install tracker, joined to `install local` runs by machine id. Keep `ARCHITECTURE.md` in sync.

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
- **Changed detection or identification logic** (cluster type, CLI run context, Docker context, AI tool) → update its section here ("Telemetry and cluster detection", "CLI update check") and anything `ARCHITECTURE.md` says about it.
- **A contract with another repository or component changes** (a proto, the OpenAPI schema, something the Control Plane imports) → update the section that names both ends here.
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

| Workflow | What it runs | Time |
| --- | --- | --- |
| `oss-lint-go` | `golangci-lint run ./...` | 1m45s |
| `oss-lint-proto` | `buf lint`, `buf format --diff`, and `buf breaking` against `main` | 40s |
| `oss-unit-tests` | `make unit-tests` | 2m15s |
| `oss-integration-tests` | the `_Integration` tests with MongoDB, NATS, MinIO and Postgres | 2m55s |
| `oss-verify-crds` | `make verify-crds-generated` | 1m05s |
| `oss-verify-protobuf` | `make verify-protobuf-generated` | 35s |
| `oss-lint-pr` | the pull request title (see "PR title format"); also needs `PR_NUMBER`, so it only runs once the pull request exists | 45s |
| `security-pr-scan` | Semgrep, Gitleaks and Trivy on the changed files; `make security-scan` runs the same locally (needs Docker). Its definition is not in this repository, and it reports as the `testkube-dev` GitHub App rather than `testkubebot` | 40s |

Times are the median wall-clock time of each check on the last 12 merged pull requests (October 2026), including pod start-up. They run in parallel, so a full pass takes about as long as `oss-integration-tests`.

The same file also holds `docs-prettier`, which checks testkube-docs, not this repo.

To run them yourself, use the Testkube MCP server (see "MCP integration") connected to the environment that runs these checks: the "GitHub Advanced Integration" environment of the Testkube OSS organization on Prod US (`app.testkube.io`), organization `tkcorg_2f14949f54f2e2d8`, environment `tkcenv_98c3a7b4da95e4e8`, whose MCP endpoint is `https://api.testkube.io/organizations/tkcorg_2f14949f54f2e2d8/environments/tkcenv_98c3a7b4da95e4e8/mcp`. The IDs are not secret, as every pull request's check links show them; the endpoint needs a token from a member of that organization. Argo CD syncs `test/testkube/oss-tests/gh-quality-loop.yaml` there from `main` (testkube-deployment `envs/prd-agents/apps/testkube-runner-github-advanced-integration.yaml`), so a merged change to it applies to every pull request at once; the checks report as `testkubebot`. The dev Control Plane's agent syncs a copy of the same file, and `security-pr-scan` runs from there, reporting as `testkube-dev`.

Then:

1. `run_workflow` with `workflowName` and `config: {"revision": "<branch or commit SHA>"}`. It returns the execution ID.
2. `wait_for_executions` with `executionIds` (comma-separated, to wait on several runs at once).
3. On failure, `fetch_execution_logs` with `executionId` (narrow with `step`, `grep` or `tail`), and `list_artifacts` and `read_artifact` (`fileName`) for the JUnit reports such as `unit-tests.xml`.

Without MCP, `testkube run testworkflow <name> --config revision=<branch> --watch` does the same from a CLI logged in to that environment.

## 7. Skills and starting a component

### Running locally

Pick the smallest setup that exercises the change. Everything past building needs Docker and a local Kubernetes cluster; the Makefile accepts only Linux and macOS, so on Windows use WSL.

- **CLI:** `make build-kubectl-testkube-cli` builds `bin/app/kubectl-testkube`. `make build` builds every binary into `bin/app/`.
- **Agent alone (OSS, no Control Plane):** the Tilt setup in [`DEVELOPMENT.md`](DEVELOPMENT.md). `./scripts/tilt-cluster.sh` creates the k3d cluster `testkube-dev` with a registry on `localhost:5001`, then `tilt up` builds the API server, `testworkflow-init` and `testworkflow-toolkit` from this checkout and installs `k8s/helm/testkube` into namespace `testkube-dev` with Postgres, MinIO and NATS. The API is on `localhost:8088`. `tilt ci` runs the same setup headless, runs the smoke test and exits non-zero on failure, so it is the one to use without a browser. `tilt up -- --db=mongo` switches the database.
- **`testworkflow-init` and `testworkflow-toolkit`** only run inside a TestWorkflow pod. Exercise them through the Tilt setup by running a workflow; its `install-sample-workflows` resource applies the smoke workflows from `test/*/crd-workflow/smoke.yaml`.
- **Skills:** the stack skills are in testkube-workspace (below). For writing or running the TestWorkflows under `test/`, the public skills in [testkube-skills](https://github.com/kubeshop/testkube-skills) (`testworkflow-author`, `testworkflow-runner`) know the schema and the CLI.
- **Agent with a Control Plane (full stack):** this repo cannot start a Control Plane. Use [testkube-workspace](https://github.com/kubeshop/testkube-workspace), which builds this checkout and a `testkube-cloud-api` checkout side by side: `tilt up oss` (agent alone), `tilt up cp` (Control Plane, dashboard and a connected agent) or `tilt up agent` (local agent, remote Control Plane). Its `tilt-local-dev` and `local-e2e-verify` skills cover the bring-up and how to verify a change against the running stack. It replaces `tk-dev`.
- **Agent against a sandbox Control Plane:** to check an agent change end to end without a local Control Plane, follow the [`start-component`](https://github.com/kubeshop/testkube-workspace/blob/main/.agents/skills/start-component/SKILL.md) skill in testkube-workspace. It takes the sandbox's agent endpoint, organization, environment and tokens as inputs, starts the agent from this checkout with `tilt up agent`, checks it connected, runs one workflow on it and tears it down.
- **MCP server:** `kubectl-testkube mcp serve` needs a CLI context logged in to a Control Plane (organization, environment and a token), or `TK_MCP_ENV_MODE=true` with `TK_ACCESS_TOKEN`, `TK_ORG_ID`, `TK_ENV_ID`, `TK_CONTROL_PLANE_URL` and `TK_DASHBOARD_URL`. It cannot drive the OSS Tilt agent, whose context uses `--client direct`.
- **Not a dev loop:**
  - `make dev` and `make run-api` start the API server outside a cluster with no database DSN and no MinIO, so it does not come up usable.
  - `testkube install local` (`pkg/localinstall`, see its `doc.go`) installs pinned, published charts and images for trial users, so it never runs the code in this checkout. `testkube status local` and `testkube uninstall local` sit beside it, and pointing the CLI at that install (after install, `testkube set context local`, and on uninstall) lives in `cmd/kubectl-testkube/commands/common/localcontext.go`.
