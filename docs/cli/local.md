# Local TestWorkflow development

`testkube local` is a fast, developer-owned feedback loop for a sequential
TestWorkflow. It reads a TestWorkflow YAML file locally, creates the temporary
Kubernetes resources needed to execute it, follows the result in the terminal,
and normally removes only the resources owned by that run.

It does not require a Testkube installation or a TestWorkflow CRD in the
selected cluster.

It is intentionally separate from `testkube run testworkflow`: a local run
does not create a stored TestWorkflow execution, contact a Testkube API or
Control Plane, synchronize CLI settings, send telemetry, run an update check,
or produce Testkube history, analytics, webhooks, or resource metrics.

Use a normal committed Testkube execution when persisted history, policy,
services, parallelism, artifacts, or Control Plane integration matter.

## Prerequisites

- A developer-owned Kind or k3d Kubernetes cluster, plus working `kubectl`
  credentials for it.
- A `testkube` binary that contains the `local` command. From a source checkout,
  build one with `go build -o /tmp/testkube-local ./cmd/kubectl-testkube`.
- Container-registry access for every workflow image and TestWorkflow runtime
  image. A local run does not require a Testkube API, but Kubernetes still has
  to obtain container images and the processor may inspect image metadata.
- A local Kubernetes image-pull Secret in the selected namespace for private
  images. Local mode never copies credentials from a Testkube installation.
- An interactive terminal for a workflow that contains `paused: true`, unless
  `--auto-continue` is supplied.

The default namespace is `testkube-local`. The command creates it when absent,
but never treats it as permission to delete the entire namespace.

## Command reference

```text
testkube local run --file <testworkflow.yaml>
  [--source <directory>]
  [--source-mount <absolute-container-path>]
  [--source-include <pattern> ...]
  [--source-exclude <pattern> ...]
  [--max-source-bytes <bytes>]
  [--config <key=value> ...]
  [--variable <key=value> ...]
  [--namespace testkube-local]
  [--kubeconfig <path>]
  [--context <name>]
  [--allow-non-local-context]
  [--auto-continue]
  [--keep]
  [--dry-run]

testkube local pause <run-id> [Kubernetes selection flags]
testkube local resume <run-id> [Kubernetes selection flags]
testkube local shell <run-id> [Kubernetes selection flags]
testkube local clean <run-id> [Kubernetes selection flags]
```

`--dry-run` validates the local YAML, source options, and Kubernetes selection
without creating a namespace, inspecting images, uploading source, or creating
workloads. It is useful before the first real run on a cluster.

## Kubernetes context safety

Local commands accept contexts beginning with `kind-` or `k3d-` by default.
They never change kubeconfig's current context: `--context` is an in-memory
selection for that invocation only.

Any other context fails before a mutating Kubernetes request. Use
`--allow-non-local-context` only after independently confirming that the target
is a disposable developer cluster. It is deliberately not a convenience flag
for a shared, staging, or production cluster.

Pass `--context` explicitly in scripts and demonstrations. If you use a
separate kubeconfig, pass `--kubeconfig` too.

## Kind example: fake app and Playwright

The repository includes a self-contained browser fixture at
`test/local-runner/playwright-e2e`. Its fake application is an in-cluster
BusyBox HTTP server; the Playwright test exercises a title, heading, and real
button interaction through the Kubernetes Service DNS name.

From the `src/testkube` repository root, create a disposable Kind cluster and
keep every subsequent command explicit about its context:

```bash
kind create cluster --name testkube-local-e2e --wait 5m

kubectl --context kind-testkube-local-e2e \
  apply -f test/local-runner/playwright-e2e/fake-app.yaml
kubectl --context kind-testkube-local-e2e \
  -n testkube-local rollout status deployment/local-playwright-app --timeout=2m

/tmp/testkube-local local run \
  --context kind-testkube-local-e2e \
  --file test/local-runner/playwright-e2e/workflow.yaml \
  --source test/local-runner/playwright-e2e/source
```

The fixture uses `mcr.microsoft.com/playwright:v1.56.1` and a source project
locked to `@playwright/test` `1.56.1`. Keep those versions aligned. `npm ci`
therefore needs npm-registry access unless the test image/source has been
prepared for an offline environment.

The test workflow contains no TestWorkflow `services` or artifact block. The
fake app is deployed independently so the local workflow can validate a real
browser path without invoking unsupported TestWorkflow service semantics.

For a source checkout whose TestWorkflow init and toolkit images are not
published or reachable from Kind, be aware that the current processor also
looks up image metadata through a registry. Loading an image with
`kind load docker-image` can avoid a node-side pull, but it does **not** make a
private or host-local tag resolvable to that metadata lookup. Use a registry
name the developer machine can resolve (or a published test tag) with matching
`TESTKUBE_TW_INIT_IMAGE` and `TESTKUBE_TW_TOOLKIT_IMAGE` values, then load that
same tag into Kind if needed. Local mode is Control-Plane-independent; it is
not yet registry-air-gapped.

After recording the result, remove the disposable cluster by its exact name:

```bash
kind delete cluster --name testkube-local-e2e
```

## k3d example

The same workflow works on k3d. Do not rely on whichever context happens to be
current:

```bash
k3d cluster create testkube-local-e2e --wait

kubectl --context k3d-testkube-local-e2e \
  apply -f test/local-runner/playwright-e2e/fake-app.yaml
kubectl --context k3d-testkube-local-e2e \
  -n testkube-local rollout status deployment/local-playwright-app --timeout=2m

/tmp/testkube-local local run \
  --context k3d-testkube-local-e2e \
  --file test/local-runner/playwright-e2e/workflow.yaml \
  --source test/local-runner/playwright-e2e/source

k3d cluster delete testkube-local-e2e
```

## Running uncommitted source

`--source` packages the selected working tree into a bounded, gzip-compressed
archive and supplies it to the workflow through a temporary run-owned relay.
The CLI rewrites only an in-memory copy of the TestWorkflow: the YAML file and
working tree are not edited. A top-level `spec.content.git` is replaced for the
local run, so an uncommitted edit is what the workflow sees rather than the
last pushed Git revision.

Without an existing Git mount, the source mount defaults to `/data/repo`; use
`--source-mount` to choose another absolute container path. Source overrides
apply only to top-level content. Nested step Git content is rejected because it
would otherwise continue to fetch committed remote state.

The default uncompressed source limit is 100 MiB. Use `--max-source-bytes` to
set a narrower or larger explicit bound when appropriate. The archive rejects
absolute or escaping entry names and unsafe symlink targets instead of
silently copying them into the cluster.

### `.testkubeignore` and source flags

Place `.testkubeignore` at the root supplied to `--source`. It uses a small,
deliberate Testkube pattern language, not the complete `.gitignore` grammar:

- Blank lines and `#` comments are ignored.
- `*`, `?`, character classes, and `**` are supported.
- A pattern without `/` matches a path component at any depth.
- A trailing `/` applies to a directory and its descendants.
- Prefix a pattern with `!` to re-include a later matching path.
- Rules are ordered: built-in exclusions, `.testkubeignore`,
  `--source-exclude`, then `--source-include`. Later rules win.

The built-in exclusions cover Git/Testkube metadata, dependency/output trees
such as `node_modules`, `vendor`, `dist`, `build`, and `coverage`, common IDE
directories, operating-system metadata, and common editor backup/swap files.
Use `--source-include` for an intentional exception, for example:

```bash
/tmp/testkube-local local run \
  --context kind-testkube-local-e2e \
  --file workflow.yaml \
  --source . \
  --source-include generated/fixture.json
```

## Supported subset

The local runner is deliberately a narrow Kubernetes execution path. It
supports one TestWorkflow document, sequential `setup`, `steps`, and `after`
trees, ordinary `run`, `shell`, and delay operations, top-level files/Git/
tarball content or `--source`, literal environment values, existing local
Secret and ConfigMap references, conditions, retries, timeouts, optional or
negative steps, non-sensitive configuration, variables, and ordinary
low-security Job/Pod settings.

It rejects or does not support the following features because they depend on
the Control Plane, additional worker behavior, storage, or unsafe Kubernetes
semantics:

- Top-level or step template references (`spec.use`, `use`, or `template`).
- TestWorkflow `services`, parallel steps, nested `execute`, and workflow PVCs.
- Artifact blocks, execution events/webhooks, storage/analytics features, and
  any operation requiring a Testkube API URL or execution token.
- A source override combined with conflicting nested Git content or tarball
  mounts.
- Command-line values for a configuration field marked sensitive; create a
  local Kubernetes Secret and reference it instead.
- Host paths, host networking/PID access, privileged containers, and other
  low-security fields rejected by the workflow processor.

Unsupported fields fail before the command creates the namespace, uploads a
source archive, or creates a Job. Keep the main workflow deliberately small;
run the normal Testkube path for the full feature set.

## Breakpoints and inspection

A step with `paused: true` is an interactive breakpoint. On an interactive
terminal, the prompt shows the step and run ID. Press Enter to continue, `s`
for a shell, `p` for the selected Pod/status/events, or `a` to abort. Shells
use the TestWorkflow-provided `/.tktw/bin/sh`, so the tested image does not
need to provide its own shell.

For a non-interactive invocation, a paused workflow fails before resource
creation unless `--auto-continue` is set. The `pause`, `resume`, and `shell`
commands can also operate on an exact retained run ID from another terminal.

## Cleanup and retained runs

Each invocation receives a unique run ID and labels its owned Job, Pod,
generated ConfigMaps/Secrets/PVCs, and source relay resources with
`testkube.io/local-run-id=<run-id>`. By default, terminal success, failure,
abort, and interrupt paths remove only those exact resources.

Use `--keep` to retain a run for debugging. The command prints the run ID and
copy-paste inspection commands; later remove only that run with:

```bash
/tmp/testkube-local local clean <run-id> \
  --context kind-testkube-local-e2e
```

`clean` is idempotent and has no broad `--all` form. It never removes
unrelated resources or existing user Secrets/ConfigMaps in `testkube-local`.
To inspect a run without changing it, use its exact label selector:

```bash
kubectl --context kind-testkube-local-e2e -n testkube-local \
  get job,pod,service,secret,configmap,pvc \
  -l testkube.io/local-run-id=<run-id>
```

If the CLI exits unexpectedly, use the same exact `local clean <run-id>`
command. Do not replace it with `kubectl delete namespace testkube-local`.

## Result boundaries

A successful local run confirms the generated Kubernetes execution and its
terminal TestWorkflow result. It is not an authoritative Testkube execution:
it does not appear in Testkube execution history, analytics, artifacts,
webhooks, metrics, or dashboards. Commit/push and use the normal Testkube
workflow when the run needs those durable product behaviors.
