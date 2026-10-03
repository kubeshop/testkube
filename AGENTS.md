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
- `cmd/convert` is a one-shot tool that migrates control-plane data from MongoDB to PostgreSQL (see "Mongo to Postgres conversion" below).
- `cmd/debug-server` is a simple HTTP server that dumps incoming requests for debugging purposes.
- `cmd/proxy` proxies HTTP requests to the Testkube API server for local development and debugging.
- `cmd/choco-stub` displays a deprecation message for the old Chocolatey package location.
- `cmd/tools` contains internal tooling for release management and version bumping.

## MCP integration

- `pkg/mcp/` implements the Model Context Protocol server for AI assistant integration.
- Exposes tools across workflows, executions, artifacts, and metadata via `testkube mcp serve` (CLI), Docker image (`testkube/mcp-server`), or Control Plane's `/mcp` endpoint per environment.
- Uses interface-based tool design; new tools need registration in both `pkg/mcp/server.go` and control plane's `mcp_handler.go`.
- See `pkg/mcp/README.md` for architecture, tool patterns, and usage examples.
- Insights board tools (`pkg/mcp/tools/boards.go`) keep their rules in `pkg/mcp/boards/`: report param validation and defaults, the report-to-`/insights/*` query translation, and the layout. It is a port of the dashboard's TypeScript (`utils/insights.ts`, `DynamicFilters/types.ts`, `reports/*/type.ts` in `testkube-cloud-api/js/packages/web`), since the Control Plane stores report params opaquely. **The Control Plane's `HandlerClient` must use this package rather than reimplement it**, and a dashboard change to those files needs a matching change here; `testdata/translation_cases.json` pins the translation.
- Boards are organization-scoped and the Control Plane refuses API tokens on every board endpoint, so the board tools need a user session. `APIClient` refuses a `tkcapi_` token before sending anything and returns `tools.ErrBoardsRequireUser`. Every board write reads the board first and resends its description. Current Control Planes keep a description an update omits, but older ones clear it, so resending is what keeps it on those.
- **Board updates are optimistic-concurrency writes.** Resending a value read earlier (the description, a recomputed layout) would overwrite a concurrent edit, so every update - `update_board` and the three report tools - goes through `writeBoard` in `pkg/mcp/tools/boards.go`: it sends `expectedVersion` (the board `version` it read; every write to a board increments it), the Control Plane refuses a stale write with 409, both clients turn that into `tools.ErrBoardChanged`, and the write is rebuilt from a fresh read, up to `boardWriteAttempts` times. A write's builder must derive everything from the board it is handed, never from an earlier read. The token is a counter, not `updatedAt`: two writes can share a timestamp, and a reused token would let a stale write through. A Control Plane that predates versions returns none, and the write is then unconditional. `delete_board` is deliberately not conditional: it resends nothing it read (the read only resolves a slug to the ID it deletes by), and the Control Plane checks visibility and delete rights against the board as it is at delete time, so deleting removes the board whatever changed since the read, as deleting in the dashboard does.
- **Relative report ranges are anchored in a time zone.** The dashboard ends a `day`/`week`/`month`/`quarter` range at the viewer's local midnight, so `render_board` takes an IANA `timeZone` (default UTC) and passes it as `boards.QueryOptions.Location`; `boards` embeds `time/tzdata` because the MCP also runs from images without a zoneinfo database.

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
- Build the Mongo to Postgres convert tool with `make build-convert` (also part of `make build`). Its image, `kubeshop/testkube-convert`, is the `convert` target in `docker-bake.hcl`, built from `build/new/convert.Dockerfile` and published by `.github/workflows/new-build.yaml` with the other images.

## Mongo to Postgres conversion

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

## Execution lineage and reruns

- `TestWorkflowExecutionLineage` (`baseId`, `rootId`, `attempt`) records what an execution is a rerun of. It is written for **every** execution, not only reruns: an original run is its own root at attempt 1, so "every execution of chain R" is one predicate and includes the original - in SQL `COALESCE(lineage_root_id, id) = R`, matching the chain index, because a legacy row carries NULL there and is its own root. `TestWorkflowExecution.EffectiveLineage()` synthesizes that default for rows written before the columns existed, which is why nothing has to be backfilled - derive it there and nowhere else, or the two repositories will disagree.
- It travels `ScheduleRequest.base_execution_id` -> `Enqueuer.deriveLineage` -> the execution record -> `ExecutionStart.lineage` -> `ExecutionConfig.Lineage` -> `RerunExecutionId()` in the pod, which is what makes the reserved `execution("rerun")` reference resolve. **Every path that builds an `ExecutionStart` must set it from the execution record**, in this repo and in `testkube-cloud-api`: a writer that forgets sends the pod no lineage, `execution("rerun")` silently stops resolving, and nothing reports it. That has already happened once - `go build` cannot catch it, because an unused helper function is not a compile error; the `unused` linter is the guard.
- Only the base id travels on the wire. The Control Plane derives the root and attempt by loading the base through the environment-scoped results repository, which is also where it confirms the base belongs to the caller's environment - `proto/service.proto` requires that, and the load is what enforces it. A caller able to assert a root or an attempt could forge a chain.
- **An original run has an empty `baseId` and must resolve to no rerun.** Returning `rootId` there would make every execution a rerun of itself.
- **Reserved references win over the registry.** `execution("parent")` and `execution("rerun")` are resolved before the executions the workflow scheduled, so a child aliased - or a workflow named - `parent`/`rerun` cannot shadow them. A collision is refused rather than resolved either way, because preferring the reserved meaning would instead make that child unreachable by name. `IsReservedRef` is the list.

## Transient-failure retries

- `pkg/runner/runner.go` runs `worker.Destroy` (cleanup of the execution's Secrets/Pods after the workflow ends) through the shared `retry()` helper via `destroyResources`. Bounded by `CleanupResourcesRetryCount` and `CleanupResourcesRetryDelay`; a brief `kube-apiserver` blip during teardown should not leave orphan resources in the customer namespace.
- `pkg/event/kind/webhook/listener.go` retries the outbound `HttpClient.Do` for `sendRetryCount` attempts with a linear `sendRetryBaseDelay`. Retryable outcomes: network errors, `5xx`, and `429`. Other `4xx` short-circuit so a bad URL / auth failure is not spammed at the subscriber. Delivery is intentionally at-least-once (subscribers own dedupe, matching Stripe/GitHub/Slack convention).

## Step dependency cache

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
- `pkg/executioncache/volume/` is the optional shared-volume backend, described below.

### Shared volume for cache entries

With `TESTKUBE_STEP_CACHE_VOLUME_CLAIM` set, a cache object holds a few hundred bytes of
**pointer** naming a directory on a ReadWriteMany volume instead of the archive itself.
An entry is a directory mirroring the filesystem from `/` (under `<entry>/root`), so a
restore is a copy rather than an unpack and nothing is gzipped.

- **The object store is still the index.** Object name, `resolveCacheScope`, the
  namespaces and the `If-None-Match` conditional write are all unchanged, so a pull
  request still cannot publish under a key a trusted run restores - it still cannot write
  that object - and the race on one key is still resolved where it always was. **Nothing
  about authorization moves onto the volume**, and a change that moves it there would
  undo the whole point.
- **The format is its own fallback, but only for entries this installation can still
  address.** A body that does not begin with `volume.Magic` is the archive, so an object
  holding one restores through the path it always did - which is what lets the
  object-store save branch, and a cluster without a volume, keep working unchanged.
  It does **not** reach entries written before keys were partitioned by volume: those
  live under the unscoped key and a volume-enabled runner now asks only for
  `<volume id>/<key>`, so they are unreachable and go cold until they expire. That is a
  one-time cost on upgrade, bounded by `STORAGE_CACHE_EXPIRATION` (default one day).
  A migration lookup for the unscoped key was considered and not taken: it would add a
  second round trip to every miss, permanently, and would have to refuse an unscoped
  *pointer* - which may name an entry on a volume this runner cannot read - to avoid
  reintroducing the collision partitioning exists to prevent.
- **The mounts are the confinement, and they are kubelet's.** Both cache stages are
  `SetPure(true)`, so `action.Group` merges them into the step's own container and
  `CreateContainer` unions the volume mounts - the step's own command therefore holds
  whatever the cache stages mount. The read side is the whole volume **read-only**; the
  write side is a **subPath** resolving to `inbox/<root execution id>` - the execution,
  not the pod, so that a parallel or service worker writes into the inbox the agent
  already made for its parent (nothing inside a pod can reach the volume root to make
  one of its own). Dropping either would
  let any step rewrite any entry, so `processor_cache_test.go` pins both.
- **Everything on the volume is readable by every execution in the cluster.** That is the
  accepted cost of sharing one, stated in the Helm value's own documentation.
- **And the step's own command holds the writable inbox mount, which is a deliberate
  decision rather than an oversight.** Both cache stages are pure, so `action.Group`
  merges them into the step's container: the subPath confines *where* a pod may write,
  to its own execution's inbox, but not *what* it writes there. A workflow can therefore
  write past `CopyLimits` - which bound what the cache copies, not what the step does -
  and can leave directories only its own user may enter, which the agent can never
  reclaim, being neither root nor the owner with `fsGroup` inoperative on NFS. The
  second is permanent and outlives the execution.
  **Do not treat the limits or `SharedDirMode` as a security boundary**; they keep the
  cache's own writes well-behaved and nothing more. The boundary is that the volume is
  only enabled where every workflow in the environment is trusted with it, which both
  charts' values say at length. Making `ProcessCacheSave` non-pure would move the mount
  into a toolkit container of its own and close this, at one extra init container per
  cached step - that is the change to make if the trust assumption ever stops holding.
- **The sweep is the agent's, not the control plane's.** The control plane is not
  necessarily in the cluster (`cmd/api-server/services/executionworker.go` points pods at
  `TestkubeProURL`), so it cannot reach the volume. `volume.Sweeper` runs as a
  leader-gated task in `cmd/api-server/main.go` and expires whole inboxes by mtime.
  Retention shorter than a pointer's lifetime is **raised to it** rather than warned
  about and used: an entry outliving its pointer only wastes space, where a pointer
  outliving its entry turns every hit on that key into a miss nothing can explain, and
  the key is immutable, so no later run can replace it until the object expires. That
  lifetime is `volume.PointerLifetime`, and it is **the earlier of both lifecycle
  rules, not just `STORAGE_CACHE_EXPIRATION`** - the bucket-wide `STORAGE_EXPIRATION`
  rule is deliberately unfiltered and so expires cache objects too, so with the cache
  rule disabled a pointer still lives the bucket-wide span. With neither rule set,
  nothing expires a pointer, no retention can outlast one, and the agent says so at
  startup instead of raising retention without bound. **Both charts mount the claim on
  their own Deployment**, because a
  runner-only installation has no api Deployment and the sweep would otherwise read its
  missing inbox directory as an empty volume and let the claim fill.
- **The sweep rests on an assumption it cannot verify, and this is the known residual
  risk of the whole arrangement.** Deleting an entry is safe only once the object naming
  it has stopped being served, and the agent has no way to know that: a day-based
  lifecycle rule is not a deadline, the store rounds it up to a UTC midnight and then
  removes asynchronously with **no bound promised on the lag**.
  `stepCacheLifecycleMargin` (48h, on top of the pointer's configured lifetime and the
  publication grace) is sized for the rounding and an ordinary delay, so it narrows the
  window - but a store slow enough still leaves a pointer answering with a hit to an
  entry that is gone, under a key no run can replace until the object finally expires.
  **Do not read the margin as a guarantee**; widening it buys probability, not
  certainty. Closing it needs a positive signal, and the agent cannot obtain one -
  attached to a Control Plane it has no access to the bucket its pointers go to, which
  is the same wall the expiration confirmation runs into above. The fix belongs to the
  Control Plane: deleting or replacing a pointer when a restore reports its entry
  missing, or reporting removals the agent can act on. Both are proto and
  `testkube-cloud-api` changes, so this branch documents the risk rather than carrying
  a half of them.
- **The volume is enabled, and swept, in every mode.** `commons.StepCacheVolumeEnabled`
  answers on the claim alone, and both the mounts and the sweep ask it. The disk is the
  agent's own and is written to in every mode, so it has to be bounded in every mode; a
  shared volume filling without limit takes the cluster's storage with it.
- **Only one half of the arrangement can be confirmed, and it is not the disk.** An
  entry is reachable only through the object naming it, so a sweep is safe exactly when
  that object also goes away. This process installs that rule only where it owns the
  bucket - standalone mode, through `SetExpirationPolicies`, whose failure is logged
  while startup continues because reading the existing lifecycle needs a permission an
  upgrade may not have granted. **An agent or runner attached to a Control Plane owns
  no bucket**: it never calls `SetExpirationPolicies`, cannot read the lifecycle, and
  must be told how long a cache object lives there - `stepCacheVolume.objectExpirationDays`
  on the runner chart, `storage.cacheExpiration` on the api chart, both reaching
  `STORAGE_CACHE_EXPIRATION`. Retention is still raised to that lifetime
  (`volume.PointerLifetime`). Where nothing is known to expire a pointer, the sweep runs
  anyway on the configured retention and the risk is stated once at startup: an entry
  swept before its pointer expires leaves a key that restores nothing until the object
  goes, and the key is immutable, so nothing repairs it meanwhile.
- **Cache keys are partitioned by the volume they are stored on.** A key is shared by
  every runner in an environment - `cacheScope` carries the environment, workflow,
  scope and namespace, and no runner or volume - while an entry on a volume is
  reachable only from that volume, and `spec.target` decides which runner runs an
  execution. Unpartitioned, whichever runner saved first owned the key for its whole
  lifetime: every runner on another volume got an exact hit it could not follow and
  could not replace, because the object is immutable. `volume.EnsureID` writes a stable
  id to `<volume>/.volume-id`, the agent carries it to the pod in
  `StepCacheVolumeConfig.ID` and `TK_CACHE_VOLUME_ID` (a pod cannot read the volume
  root on the save side - its only writable mount is a subPath), and
  `volume.ScopedKey` prefixes the key **and every restore key** with it. A prefix, not
  a suffix, so prefix matching cannot stray onto another volume's entry. The prefix is spent out of
  `MaxKeyBytes`, so the key is validated **after** scoping (`validateScopedCacheKey`) -
  checking the author's key and then adding bytes to it let a key just inside the limit
  pass in the pod and be refused by the API on every restore and save. The id
  belongs to the volume rather than the runner, so runners that do share a volume go on
  sharing its entries. `O_EXCL` publishes the name before the contents, so an agent
  killed in between leaves an empty `.volume-id` that would otherwise turn the cache off
  for the whole installation for good - every later startup waiting out the attempts and
  refusing the volume. One old enough not to be anybody's open write is **superseded,
  never repaired and never unlinked**: the agent moves to the next generation of the name
  (`.volume-id.2`, then `.volume-id.3`, bounded by `idGenerations`), and a generation that
  exists always wins over the one before it, so an abandoned name written long afterwards
  cannot take the volume back off the agents using its successor. **The agent that was
  declared abandoned must not keep its own identity either**, and that is closed on both
  sides, because one side cannot do it: after writing, a creator looks for a successor and
  adopts it (`errIDSuperseded`); and a new generation decides what it holds only once its
  `O_EXCL` create is won, inheriting whatever the generation it supersedes has gained by
  then (`inheritedID`). A successor made before that look is found by it, one made after
  it inherits the identity just written, and there is no ordering in which the two
  disagree. That look covers **every** path out of `ensureID`, not just the one that
  writes: an agent waiting out an empty generation holds no opinion for a second at a
  time, long enough for another to supersede it and the stalled creator to fill the old
  name, so a waiter can be handed an identity that was superseded while it waited. Note this needs no five-minute stall to reach: an mtime set by a skewed
  creating node can make an identity look abandoned the moment it is made, so
  `idStaleAfter` is a margin and not a guarantee. **Nothing in this file
  unlinks anything**, which is the whole point: every name is settled by an atomic
  `O_EXCL` create and no name is ever reused, so there is no instant at which one agent
  can remove what another has just created. Repairing in place instead needs exclusion,
  and a lock on a shared filesystem cannot be reclaimed safely - reclaiming it means
  unlinking a path on the strength of a stat taken earlier, which is the very race the
  lock exists to prevent, so a crash mid-repair would either wedge the volume for good or
  let two agents write different identities into it. The mode is set
  **before** the contents for the same family of reasons - a death in between would
  otherwise leave a non-empty identity at the creator's umask that no other agent can
  read and that the repair deliberately will not touch, a file with contents being
  somebody's identity rather than a half-made one.
- **A running execution holds its inbox through a lease.** The inbox is made before the
  pod starts and its mtime only moves when something is staged inside it, so an
  execution that has cached nothing yet, or runs one long step, looks expired to the
  sweep however alive it is - and unlinking it leaves the pod writing through its
  subPath to an inode nothing can reach, then publishing a pointer naming a path that
  is gone. `volume.TouchLease` refreshes `<inbox>/.lease`, `Sweeper.LeaseTTL` skips an
  inbox whose lease is fresh - but only when its age is **non-negative**, within a
  five minute skew window, because the step's own command holds the inbox mount and
  could otherwise date `.lease` a century ahead and pin that directory for good; the
  inbox's own mtime is bounded the same way, and `refreshStepCacheLeases`
  (`cmd/api-server/stepcachelease.go`) renews them every 5 minutes against a 30 minute
  TTL, inside the sweeper task so the first renewal precedes the first sweep. The live
  set is **read back from the cluster** (`worker.List` with `Finished: false`) rather
  than tracked in memory: a registration that is never cleared would pin an inbox for
  good, where a listing simply stops returning what has stopped running. Time-based for
  the same reason - a lease nobody refreshes goes stale and the sweep carries on.
  **Nothing is swept until that first listing succeeds** (`awaitStepCacheLeases`): a
  leader that has just taken over holds no leases, so a failed first pass would make
  every live inbox look abandoned. A later pass may fail safely, falling back on the
  leases the previous one wrote, which is why the interval is well under the TTL.
  **A lease is how one agent tells *another*** - an api and a runner sharing a volume
  hold separate leader elections and know only their own executions - and within one
  agent it is an indirection that can fail, because the step holds its inbox's mount and
  can create `.lease` itself, as its own user and umask. A lease the agent cannot write
  is **replaced** rather than given up on (unlinking needs the write bit on the 0777
  inbox, not ownership of the file), and `Sweeper.Protected` is the backstop: the live
  set from the last successful listing is held in memory (`liveInboxes`) and consulted
  **before** the lease and the mtime, so what this agent knows directly about its own
  executions never depends on reading it back off the volume. Without it a renewal that
  keeps failing leaves a running execution's inbox looking abandoned, and the empty-inbox
  branch takes it 30 minutes later while its pod is still writing through the subPath.
- **The agent makes each execution's inbox before its pod starts**
  (`prepareStepCacheInbox` in `kubernetesworker/worker.go`, called at both deploy
  sites), keyed on the execution's **root** id so that one inbox serves an execution
  and everything it spawns - a parallel or service worker builds its pods from inside a
  pod, where nothing can reach the volume root, so it cannot make an inbox of its own.
  For the same reason it is made for **every** execution the volume is enabled for, and
  deliberately **not** gated on this bundle mounting the claim: a workflow may cache
  only inside a nested worker, whose spec is bundled later, so the root bundle proves
  nothing. The empty directories that costs are the sweep's to reclaim - an inbox
  holding no entries goes as soon as its lease is stale, well before the retention a
  real entry gets, since retention exists to outlive a pointer and an empty inbox has
  none.
  kubelet would create the `subPath` directory itself - it creates a missing
  one and its parents - but owned by **root**, with the mode of the volume root and no
  regard for `runAsUser`; `fsGroup` is not applied to a multi-writer volume, since the
  in-tree NFS plugin never reads it. A step running as another user would then be
  unable to write to its own inbox and would fall back to the object store on every
  save, so the cache would look configured and never once be used. kubelet takes an
  existing directory as it is, so the agent makes it first, mode `0777` because it
  cannot know which user the step runs as. Doing so also keeps the shared `inbox`
  parent out of kubelet's hands: creating a `subPath` is not tolerant of a concurrent
  creation before Kubernetes v1.37, so two executions starting together on one node
  while that parent was still missing could fail to start. It is **best effort** - a
  failure logs and continues, because kubelet still creates the directory and the
  pod-side probe still falls back. Note this needs the agent's claim and the execution
  namespace's claim to share backing storage, which the chart documents.
- **Every directory written onto the volume is chmod'ed to `SharedDirMode` (0777).** A
  step container creates them under its own umask, as whatever user the workflow chose,
  while the agent sweeps them as another; a directory's write bit is what permits
  unlinking what is inside it, so one left at 0755 is a subtree the agent can never
  reclaim. Mkdir's mode argument is not enough - the usual 022 turns 0777 into 0755
  before it reaches the filesystem - so `mkdirShared` sets it explicitly, and
  `mkdirAllShared` covers the prefix above a declared path's root, which the walk never
  visits. The sweep also **reports a removal it could not make and continues**, rather
  than returning at the first: one stuck inbox must not stop it reclaiming everything
  after it.
- **The entry limit is the volume's own, and a refusal on it falls back.** The volume
  counts every directory where the archive's walker emits only files and links, so a
  directory holding two files is three entries here and two there. The archive path
  enforces its own count before asking for a grant, which is what keeps an entry no
  restore would accept out of the store - so falling back cannot publish one, and
  settling here would drop a cache the archive had room for. The size limit falls back
  for the same reason, measuring the tree where the archive measures the gzip of it.
- **Permissions travel beside the tree, not on it.** A file on the volume is read by an
  execution that may run as another user, so what is stored is widened to be readable
  by anyone - which loses the source's own mode. `modes.v1`, written beside
  `<entry>/root`, records every path whose permissions differ from `DefaultFileMode`
  (0644) or `DefaultDirMode` (0755); the restore **sets** those defaults on everything
  it writes - including the declared directory itself, which `openDeclaredRoot` makes
  rather than the walk - and then applies the recorded ones, **deepest first**, so
  narrowing a directory cannot shut the restore out of what is still inside it. Set,
  not passed to `MkdirAll` and `OpenFile` and left there: a mode given at creation is
  filtered by the restoring process's umask, so under 0077 a tree cached at an ordinary
  0755/0644 would come back private to whoever restored it, and the manifest records
  only what differs from the defaults so the defaults have to be applied rather than
  assumed. Without this a key saved 0600 came back
  0666 and ssh refused it, where the archive backend carries each mode in its tar
  header - the same workflow behaving differently by backend. The file is versioned in
  its name so an older agent meeting a newer entry simply ignores it and restores at
  the defaults, and `Store.OpenEntry` therefore roots at the **entry**, with the tree
  inside it. It is **streamed and bounded by the entry limit** (`DecodeModesFrom`, with
  one record capped at `maxModeRecordBytes`, records counted as **read** rather than as
  kept so that a repeated or malformed one still spends the budget, and the paths
  retained capped at `maxModeTotalBytes` because a count alone leaves their length
  free): it is read before any copy limit applies,
  and the inbox an entry is built in is mounted into the step's own container - so a
  workflow can commit a manifest of any size, which every later execution restoring
  that key would otherwise read into memory whole. **The save records under the same
  cap** (`Recorder`) and streams the manifest out rather than encoding it into a buffer
  first: the walk feeding it is bounded only by the entry limit, so 500,000 paths of up
  to `PATH_MAX` is gigabytes of strings held in the step's own container, where being
  killed for it fails the step. Matching the decode's number is what makes dropping the
  excess free - a record past it is one every restore would discard anyway.
- **Symlinks are carried, and never followed on the way out.** A restore writes through
  an `os.Root` opened on each declared path, so a symlink already sitting there cannot
  redirect a write outside it - the entry may have been written by another workflow. The
  links inside an entry are recreated as links: `node_modules/.bin` is entirely symlinks,
  and dropping them would restore an incomplete tree that still answers as an exact hit,
  so the step would never repair it.
- **A volume that will not take an entry falls back to the object store.** The open-time
  probe cannot predict a volume filling mid-copy, so `saveToVolume` reports whether it
  settled the save. What falls through is decided by whether the other backend would
  answer differently - and **both** limits do: the size, because the volume weighs the
  tree where the archive weighs the gzip of it, and the count, because the volume counts
  every directory where the archive's walker emits only files and links. A failed copy
  or commit falls back too. What keeps a fallback from publishing an entry no restore
  would accept is that the archive path enforces `cacheMaxEntries` on its own count
  before asking for a grant: an archive over it would store happily and then be refused
  by every restore of it, under a key that is immutable, so the step would reinstall
  every execution until it expired.
- **`SaveRequest.Size` is the object's size, so on this path it is the pointer's.** The
  contract is what the bucket is about to receive, which a quota can refuse before the
  transfer; the tree is on a volume the Control Plane neither provisions nor can
  measure. Sending the tree's size would let a quota refuse a save that stores a few
  hundred bytes. The tree is bounded instead by the agent's own size limit and
  reclaimed by the sweep. This is also why `Commit` runs **before** `Save` here, unlike
  the archive path: the pointer does not exist until the entry has a published name,
  and a committed entry that no pointer names is discarded on every path that does not
  publish.
- A key never becomes a path segment, which is why `MaxEncodedKeyBytes`' budget against
  the object store's 1024-byte limit does not have to answer to POSIX's 255-byte
  `NAME_MAX`.

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

## CLI prompts and non-interactive runs

- `pkg/ui/interactive.go` holds the terminal check every prompt goes through. `ui.Select`, `ui.Confirm` and `ui.TextInput` call `requireInteractive` first and exit with an actionable message when stdin is not a terminal, so no call site has to guard itself. `ui.StdinIsInteractive()` exposes the same check for callers that want to take a different path instead of failing.
- The guard exists because a prompt with nobody to answer it used to spin: `atomicgo.dev/keyboard` cannot open a non-terminal stdin, reports every failed read as an empty keypress with a nil error, and pterm has no case for an empty keypress, so its listener loops forever and burns more than a core.
- `GetClient` (`cmd/kubectl-testkube/commands/common/client.go`) uses `ui.StdinIsInteractive()` to refuse starting a login when a token refresh fails with no terminal present, the same way the email-link branch beside it already does.
- Adding a new prompt: use the `ui` helpers rather than pterm directly, and it inherits the guard. Adding a command that must not prompt at all: branch on `ui.StdinIsInteractive()`.
- Tests flip the unexported `stdinIsInteractive` seam rather than allocating a pseudo terminal.

## On-prem demo install

- `testkube init demo` (`cmd/kubectl-testkube/commands/init.go`) installs the On-Prem demo on the new architecture: the Control Plane (enterprise chart + `values.demo.v2.yaml`) plus a **separate** listener-enabled runner (`kubeshop/testkube-runner`). The bundled agent is gone.
- The CLI generates one agent secret key per install (`common.GenerateDemoAgentSecretKey`) and passes the *same* key to both sides — injected into the Control Plane's `bootstrapConfig` runner so the CP provisions it, and into the runner install (`common.HelmUpgradeOrInstallTestkubeOnPremDemoRunner` → `demoRunnerHelmOptions`). No secret is baked into the binary or chart.
- The runner identity (`demoRunnerID`/`OrgID`/`EnvID`) must stay in sync with the runner declared under `bootstrapConfig` in `values.demo.v2.yaml` (in `testkube-cloud-charts`).
- The legacy `values.demo.yaml` profile (bundled agent, MongoDB) is deprecated but kept for older CLIs.
- Install lifecycle telemetry: `init demo` reports `cli_install_started` (before Helm install) and `cli_install_finished` (after success) to the license service at `POST https://license.testkube.io/events`. The client lives in `pkg/diagnostics/validators/license/client.go` (`Client.ReportEvent`, `EventRequest`, the `LicenseEventsURL` and `EventCLIInstall*` constants); the `reportLicenseEvent`/`waitLicenseEvents` helpers in `init.go` make delivery telemetry-gated, non-blocking (background goroutine), and flushed before exit with a bounded wait. The license key in the body is the credential (validated worker-side before recording), so no shared secret ships in the CLI. Adding a new lifecycle event: add an `EventCLIInstall*` constant and a `reportLicenseEvent` call, and allowlist it in the license worker's `/events` handler (`testkube-infrastructure`). Keep `ARCHITECTURE.md` in sync.

## Configuration references

- Agent behavior is driven by env vars defined in `internal/config/config.go` (scan for `envconfig:"..."` tags when researching a toggle).
- The convert tool does not use `internal/config`. Each flag in `cmd/convert/main.go` falls back to an env var: it reuses the API server's `API_MONGO_*`, `API_POSTGRES_DSN`, `SKIP_DB_CREATION` and `DISABLE_POSTGRES_MIGRATIONS`, and adds its own `CONVERT_BATCH_SIZE`, `CONVERT_READ_BATCH_SIZE`, `CONVERT_DRY_RUN`, `CONVERT_RESET`, `CONVERT_RESET_CONFIRMED`, `CONVERT_SKIP_ERRORS`, `CONVERT_SKIP` and `CONVERT_VERIFY`. MongoDB TLS material is passed as file paths, because the tool has no cluster client to read a Secret with.
- GitOps sync of Kubernetes resources into the Control Plane is gated by `GITOPS_KUBERNETES_TO_CLOUD_ENABLED` (default `false`), and additionally requires the Control Plane to report cloud storage support.
- TestTriggers accept `spec.event` or a `spec.events` list (mutually exclusive, validated service-side); always consume them via `EffectiveEvents()` so both forms are honored — classification gates that read the single `event` field directly will silently skip list-form triggers.
- Git trigger informer behavior is tuned via `TEST_TRIGGER_GIT_INFORMER_RECONCILE_INTERVAL`, `TEST_TRIGGER_GIT_INFORMER_REPO_DEPTH`, `TEST_TRIGGER_GIT_INFORMER_LIST_TIMEOUT`, `TEST_TRIGGER_GIT_INFORMER_MAX_COMMITS_SCAN`, `TEST_TRIGGER_GIT_INFORMER_PULL_RETRIES`, and `TEST_TRIGGER_GIT_INFORMER_PULL_RETRY_DELAY`.
- Git trigger informer execution is leader-gated in `cmd/api-server/main.go` through the shared `leader` coordinator tasks, so only the active leader performs periodic git pulls/reconciliation.
- Helm chart values are the source of deployment defaults; `build/_local/values.dev.yaml` (shaped by the `values.dev.tpl.yaml` template) shows the local overrides used by `tk-dev` if you need a concrete reference.
- `testkube-api` chart values `jobTolerations`/`jobAffinity`/`jobNodeSelector` (`k8s/helm/testkube-api/values.yaml`) set the `tolerations`/`affinity`/`nodeSelector` applied to the ephemeral pods spawned per test execution (`_job-template.yaml.tpl` for legacy prebuilt/container executors, `_slave-pod-template.yaml.tpl` for test-workflow slave pods). Unset (empty) by default and deliberately does not fall back to `global.tolerations`/`global.affinity`/`global.nodeSelector`, since those already carry a non-empty default (an arm64 toleration) that would otherwise silently change job/slave pod scheduling for every chart consumer.
- CLI update-check toggle: set `TESTKUBE_DISABLE_UPDATE_CHECK=1` to suppress both the per-command hint and the `testkube version` status block. The CLI persists `lastUpdateCheckAt` and `latestKnownVersion` in `~/.testkube/config.json` to throttle the per-command hint to once per day.
- Step dependency caches can be kept on a shared ReadWriteMany volume instead of whole in the object store, via `TESTKUBE_STEP_CACHE_VOLUME_CLAIM` (an existing RWX PVC, which must exist in every execution namespace; empty disables it), `TESTKUBE_STEP_CACHE_VOLUME_MOUNT_PATH` (where the agent mounts it for the expiry sweep), `TESTKUBE_STEP_CACHE_VOLUME_RETENTION_DAYS` (default `7`) and `TESTKUBE_STEP_CACHE_VOLUME_SWEEP_INTERVAL` (default `1h`). Helm exposes all four as `stepCacheVolume.*` on **both** the `testkube-api` and `testkube-runner` charts, because both build execution pods, and both mount the claim on their own Deployment so the leader-gated sweep can reach it in a runner-only installation too. The volume and its sweep run in **every** mode; the claim alone enables them. Retention shorter than the earlier of `STORAGE_CACHE_EXPIRATION` and `STORAGE_EXPIRATION` is raised to it at startup, with a warning, rather than used as given. A runner owns no bucket and cannot read its lifecycle, so it is told one through `stepCacheVolume.objectExpirationDays` (default `1`), which reaches `STORAGE_CACHE_EXPIRATION`; the api chart uses `storage.cacheExpiration` as before. With nothing expiring a pointer the sweep still runs on the configured retention, and the agent warns once that a swept entry may leave a pointer behind it. See "Shared volume for cache entries" above for what the arrangement does and does not confine.
- Object retention is driven by `STORAGE_EXPIRATION` (whole bucket, in days) and `STORAGE_CACHE_EXPIRATION` (step dependency caches under the `.tkcache/v1` prefix, in days). **`STORAGE_CACHE_EXPIRATION` defaults to 1 day; `STORAGE_EXPIRATION` stays opt-in.** The difference is the filter, not taste: the cache rule is confined to the cache prefix and can only delete caches, where the bucket-wide rule is unfiltered and governs artifacts and logs too, so a default there would delete a deployment's results on an upgrade. `TestExpirationDefaults` pins both. `SetExpirationPolicies` in `pkg/storage/minio/minio.go` applies them with `SetBucketLifecycle`, which replaces the bucket lifecycle wholesale - so it reads the existing configuration first and carries through every rule Testkube does not own, matched by ID (`mergeLifecycleRules`). That merge is what makes a default safe at all: without it, defaulting either setting would drop the rules of installations whose bucket lifecycle is managed elsewhere, purely by upgrading. It fails closed - if the existing lifecycle cannot be read, nothing is written. That read is a permission earlier versions did not need (`s3:GetLifecycleConfiguration` on S3 and MinIO, `storage.buckets.get` on GCS), and because `STORAGE_CACHE_EXPIRATION` now defaults to 1 the call happens on every installation rather than only those configuring an expiration - so an upgrade can need permissions the deployment never granted. The error names them. Note also that the bucket-wide rule is unfiltered and so covers cache objects too, and the earlier expiration wins — a cache TTL can only bring eviction forward, never postpone it, and `MustGetMinioClient` warns when it is set longer than the bucket-wide one.

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
