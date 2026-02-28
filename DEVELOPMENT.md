# Local Development Guide

This guide explains how to set up and use the local development environment for Testkube using [Tilt](https://tilt.dev).

The Tilt-driven development environment builds and deploys:

- **testkube-api-server** — The main API server (with optional live reload on code changes)
- **testworkflow-init** — Init container for Test Workflow execution
- **testworkflow-toolkit** — Runtime utilities for Test Workflow containers

> **Note**: This guide applies to developing the **standalone/open-source Testkube agent**. It does not cover development with the agent connected to the Testkube Control Plane.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker** with BuildX support
- **Kubernetes cluster** — one of:
  - [kind](https://kind.sigs.k8s.io/) (recommended)
  - [k3d](https://k3d.io/)
  - [minikube](https://minikube.sigs.k8s.io/)
  - [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled
  - [Rancher Desktop](https://rancherdesktop.io/)
- **[Tilt](https://docs.tilt.dev/install.html)** v0.30.0 or later
- **[Helm](https://helm.sh/docs/intro/install/)** v3.x
- **Go** 1.25+ (enables live reload — optional but recommended)

## Quick Start

1. **Create a local Kubernetes cluster** (if you don't have one):

   ```bash
   # Using the provided script (creates a kind cluster)
   ./scripts/tilt-cluster.sh

   # Or manually with kind
   kind create cluster --name testkube-dev

   # Or with k3d
   ./scripts/tilt-cluster.sh --k3d
   ```

2. **Start the development environment**:

   ```bash
   tilt up
   ```

   This will:
   - Detect your Go toolchain and enable live reload automatically
   - Build 3 images: `testkube-api-server`, `testworkflow-init`, `testworkflow-toolkit`
   - Create the `testkube-dev` namespace
   - Deploy the Testkube Helm chart with all dependencies (PostgreSQL, MinIO, NATS)
   - Create MinIO buckets for artifacts and logs
   - Set up port forwards for local access

3. **Open the Tilt UI** at http://localhost:10350 to monitor the deployment.

4. **Verify the setup**: If you have Go installed, Tilt automatically compiles the Testkube CLI and configures it to talk to the local API server. Use the **Run CLI Command** button on the `configure-cli` resource in the Tilt UI, or run commands directly:

   ```bash
   ./build/_local/kubectl-testkube get testworkflows
   ```

## Options

The Tiltfile supports several command-line options:

```bash
# Default: auto-detects Go for live reload, uses PostgreSQL
tilt up

# Enable Delve debugger (attach on :56268)
tilt up -- --debug

# Use MongoDB instead of PostgreSQL
tilt up -- --db=mongo

# Use both MongoDB and PostgreSQL
tilt up -- --db=both

# Disable live reload (force full Docker rebuilds)
tilt up -- --no-live-reload

# CI mode: auto-runs smoke tests, exits on success/failure
tilt ci
```

| Option | Default | Description |
|--------|---------|-------------|
| `--live-reload` / `--no-live-reload` | Auto-detect Go | Live reload compiles Go locally and syncs the binary into the container (~2s vs ~30s full Docker rebuild) |
| `--debug` | Off | Builds with Delve debugger, disables Go optimizations, exposes debug port :56268 |
| `--db=<backend>` | `postgres` | Database backend: `mongo`, `postgres`, or `both` |

## Architecture

### Components

```
┌────────────────────────────────────────────────────────────────────────┐
│                        testkube-dev namespace                           │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌─────────────────────┐    ┌─────────────────────┐                   │
│  │  testkube-api-server │◄──►│  PostgreSQL / Mongo │                   │
│  │    (your code)       │    │   :5432 / :27017    │                   │
│  │    :8088 (HTTP)      │    └─────────────────────┘                   │
│  │    :8089 (gRPC)      │                                              │
│  └──────────┬───────────┘    ┌─────────────────────┐                   │
│             │                │       MinIO         │                   │
│             │                │   :9000 (API)       │                   │
│             │                │   :9001 (Console)   │                   │
│             │                └─────────────────────┘                   │
│             │                                                          │
│             │                ┌─────────────────────┐                   │
│             └───────────────►│        NATS         │                   │
│                              │   :4222             │                   │
│                              └─────────────────────┘                   │
│                                                                        │
│  Test Workflow Execution (spawned dynamically by API server):          │
│  ┌─────────────────────┐    ┌─────────────────────┐                   │
│  │  testworkflow-init  │───►│ testworkflow-toolkit │                   │
│  │  (init container)   │    │  (runtime utilities) │                   │
│  └─────────────────────┘    └─────────────────────┘                   │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Images Built

All images use Dockerfiles in `build/_local/` and use production image names so Tilt can auto-match them against the Helm-rendered manifests:

| Image | Dockerfile | Build Mode | Description |
|-------|------------|------------|-------------|
| `kubeshop/testkube-api-server` | `agent-server.Dockerfile` | Live reload (default) or Docker build | Main API server — rebuilds on code changes |
| `kubeshop/testkube-tw-init` | `testworkflow-init.Dockerfile` | Docker build | Init container for TW execution |
| `kubeshop/testkube-tw-toolkit` | `testworkflow-toolkit.Dockerfile` | Docker build | Runtime utilities for TW containers |

### Build Modes

**Live reload** (default when Go is installed): The Go binary is compiled locally using your host toolchain (fast incremental builds) and synced into the running container. Only the binary is transferred — the container does not restart from scratch.

**Docker build** (fallback or `--no-live-reload`): A full Docker build is triggered using the `build/_local/` Dockerfiles. Slower but does not require a local Go installation.

**Debug build** (`--debug`): Adds Delve debugger to images and disables Go compiler optimizations so breakpoints work correctly. In live reload mode, gcflags `all=-N -l` are passed during compilation.

### Dockerfile Targets

Each Dockerfile provides multiple build targets:

| Target | Used When | Description |
|--------|-----------|-------------|
| `dist` | Default (no --debug) | Distroless/minimal image, no debugger |
| `live` | Live reload (no --debug) | BusyBox-based image with shell (required for binary sync) |
| `debug` | `--debug` flag | Includes Delve debugger, Go runtime |

## Port Forwards

| Service | Port | Description | Condition |
|---------|------|-------------|-----------|
| testkube-api-server | 8088 | HTTP REST API | Always |
| testkube-api-server | 8089 | gRPC API | Always |
| testkube-api-server | 56268 | Delve debugger | `--debug` only |
| PostgreSQL | 5432 | Database | `--db=postgres` or `both` |
| MongoDB | 27017 | Database | `--db=mongo` or `both` |
| MinIO | 9000 | S3-compatible artifact storage | Always |
| MinIO | 9001 | MinIO web console | Always |
| NATS | 4222 | Message queue | Always |

## Configuration

### Custom Helm Values

Create a `tilt-values.yaml` file in the repo root to override any Helm values (this file is not committed):

```bash
cp tilt-values.yaml.example tilt-values.yaml
```

> **Warning**: `tilt-values.yaml.example` ships with hardcoded default passwords for MinIO (`minio` / `minio123`) and PostgreSQL (`testkube` / `postgres5432`). These are fine for a throwaway local cluster but should be changed in `tilt-values.yaml` if your cluster is accessible from a network.

You can also start from scratch and only override what you need:

```yaml
# tilt-values.yaml
testkube-api:
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi
```

### Tiltfile Constants

You can modify the constants at the top of the `Tiltfile`:

```python
NAMESPACE = "testkube-dev"       # Kubernetes namespace
HELM_RELEASE_NAME = "testkube"   # Helm release name
HELM_CHART_PATH = "./k8s/helm/testkube"  # Path to Helm chart
```

## Development Workflow

### Making Code Changes

**API Server (live reload enabled):**

1. Edit Go files in `cmd/api-server/`, `pkg/`, or `internal/`
2. Tilt detects the change and triggers a local Go compile (~2s)
3. The compiled binary is synced into the running container
4. The process restarts with your changes — no full Docker rebuild needed

**API Server (live reload disabled):**

1. Edit Go files
2. Tilt triggers a full Docker build using `build/_local/agent-server.Dockerfile`
3. The deployment is updated with the new image

**Test Workflow images:**

1. Edit files in `cmd/testworkflow-init/` or `cmd/testworkflow-toolkit/`
2. Tilt detects the change and triggers a Docker rebuild
3. New Test Workflow executions will use the updated images

### Verification

The Tilt UI includes verification resources under the **verify** label:

- **health-check** — Manually trigger a health check against the API (`curl http://localhost:8088/health`)
- **smoke-test** — Manually trigger a smoke test that verifies the API and workflows endpoint
- **Health Check button** — Click the heart icon on the `testkube-api-server` resource in the Tilt UI

In CI mode (`tilt ci`), the smoke test runs automatically and Tilt exits on success or failure.

### Testkube CLI

When Go is installed, Tilt automatically compiles the Testkube CLI (`kubectl-testkube`) for your host OS and configures it to connect directly to the local API server. The following resources appear under the **cli** label in the Tilt UI:

- **compile:cli** — Compiles `cmd/kubectl-testkube` to `build/_local/kubectl-testkube`. Rebuilds automatically when CLI or shared code changes.
- **configure-cli** — Runs `set context --kubeconfig` to point the CLI at the local API server (`http://localhost:8088`), the `testkube-dev` namespace, and direct client mode. Runs automatically after compilation.
- **run-cli-command** — A dedicated resource for running arbitrary CLI commands. Click the **Run** button, enter any testkube subcommand (e.g. `run testworkflow curl-workflow-smoke`), and the output appears in the resource logs.

You can also use the compiled CLI directly from your terminal:

```bash
./build/_local/kubectl-testkube get testworkflows
./build/_local/kubectl-testkube run testworkflow curl-workflow-smoke
./build/_local/kubectl-testkube get testworkflowexecutions
```

> **Note**: The CLI context is stored in `~/.testkube/config.json`. If you also have a system-installed `testkube` CLI, the `configure-cli` step will update the shared config to point at your local dev environment. Run `testkube set context` again to reconfigure it when you're done with local development.

### Sample Test Workflows

The Tilt UI includes an **install-sample-workflows** resource (under the **setup** label) that installs a set of sample workflows you can use to validate your local environment. Trigger it manually from the Tilt UI to install:

| Workflow | Tool | Description |
|----------|------|-------------|
| `curl-workflow-smoke` | curl | Simple HTTP request to a public URL |
| `postman-workflow-smoke` | Postman/Newman | Runs a Postman collection from the repo |
| `junit5-workflow-smoke` | JUnit 5 / Maven | Compiles and runs JUnit 5 tests from the repo |
| `k6-workflow-smoke` | k6 | Runs a k6 load test script from the repo |

After installing, run a workflow using the **run-cli-command** resource or from the terminal:

```bash
# Via the Run button on run-cli-command:
#   run testworkflow curl-workflow-smoke

# Or from the terminal:
./build/_local/kubectl-testkube run testworkflow curl-workflow-smoke
```

### Running Tests and Linting

From the Tilt UI, trigger these manual resources:

- **make test** — Runs the full test suite
- **make lint** — Runs golangci-lint

Or run directly:

```bash
make test
make lint

# Run specific package tests
go test ./cmd/api-server/... -v
go test ./pkg/... -race
```

### Code Generation

The Tilt UI exposes all code generation targets under the **generate** label:

- **make generate** — Run all generators
- **make generate-protobuf** — Regenerate protobuf code
- **make generate-openapi** — Regenerate OpenAPI models
- **make generate-mocks** — Regenerate mock files
- **make generate-sqlc** — Regenerate SQL client code
- **make generate-crds** — Regenerate Kubernetes CRDs

### Viewing Logs

**Via Tilt UI**: Click on any resource to see live logs.

**Via kubectl**:

```bash
kubectl logs -f -n testkube-dev deployment/testkube-api-server
```

## Debugging

Debugging is opt-in via the `--debug` flag:

```bash
tilt up -- --debug
```

This builds all images with the `debug` Dockerfile target (which includes Delve) and exposes the debugger port.

**Debug Ports:**

| Image | Delve Port | Notes |
|-------|------------|-------|
| testkube-api-server | 56268 | Port-forwarded automatically |
| testworkflow-init | 56268 | Spawned dynamically during test execution |
| testworkflow-toolkit | 56300 | Spawned dynamically during test execution |

### Connecting Your IDE

**VSCode** — Add to `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Attach to API Server",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/app",
            "port": 56268,
            "host": "127.0.0.1"
        }
    ]
}
```

**GoLand/IntelliJ:**

1. Run → Edit Configurations → Add New → Go Remote
2. Host: `localhost`, Port: `56268`
3. Click Debug

### Accessing PostgreSQL

```bash
psql -h localhost -p 5432 -U testkube -d backend
# Password: postgres5432

# Connection string:
# postgresql://testkube:postgres5432@localhost:5432/backend
```

### Accessing MinIO

- **Web Console**: http://localhost:9001
- **Credentials**: `minio` / `minio123`
- **API Endpoint**: http://localhost:9000

The Tiltfile automatically creates the required buckets (`testkube-artifacts`, `testkube-logs`) via a Kubernetes Job after MinIO starts.

## Troubleshooting

### Helm Dependency Update Fails

```bash
# Manually update dependencies
helm dependency update ./k8s/helm/testkube

# Or rebuild the lock file
rm ./k8s/helm/testkube/Chart.lock
helm dependency build ./k8s/helm/testkube
```

### Image Pull Errors

Tilt handles image loading into local clusters automatically (kind, k3d, minikube). If you encounter pull errors:

- Ensure your cluster is one of the allowed contexts (see `allow_k8s_contexts` in the Tiltfile)
- For minikube, you may need to configure Tilt to use minikube's Docker daemon:

  ```bash
  eval $(minikube docker-env)
  ```

### Port Already in Use

```bash
# Find the process using the port
lsof -i :8088

# Kill it or change the port forward in tilt-values.yaml
```

### Database Connection Issues

```bash
# Check database pods are running
kubectl get pods -n testkube-dev | grep -E 'postgresql|mongodb'

# Check database logs
kubectl logs -n testkube-dev statefulset/testkube-postgresql
```

### Live Reload Not Working

If live reload is not activating:

1. Check that Go is installed and in your PATH: `which go`
2. Check the Tilt startup output for "Live reload: enabled"
3. Force it on explicitly: `tilt up -- --live-reload`
4. Ensure your Go version matches what the project requires (1.25+)

### Cleaning Up

```bash
# Stop Tilt and remove deployed resources
tilt down

# Delete the namespace
kubectl delete namespace testkube-dev

# Delete the cluster
./scripts/tilt-cluster.sh --delete

# Or manually
kind delete cluster --name testkube-dev
```

## Advanced Topics

### Allowed Kubernetes Contexts

The Tiltfile permits these contexts by default:

- `docker-desktop` / `docker-for-desktop`
- `minikube`
- `kind-kind` / `kind-testkube-dev`
- `k3d-testkube-dev`
- `rancher-desktop`

To allow additional contexts, modify the `allow_k8s_contexts()` call in the Tiltfile.

### Building Images Manually

You can build images outside of Tilt using the `build/_local/` Dockerfiles:

```bash
# Build the API server (production target)
docker build -t testkube-api-server:dev --target dist \
  -f build/_local/agent-server.Dockerfile .

# Build with Delve debugger
docker build -t testkube-api-server:dev --target debug \
  -f build/_local/agent-server.Dockerfile .

# Build Test Workflow images
docker build -t testworkflow-init:dev --target dist \
  -f build/_local/testworkflow-init.Dockerfile .

docker build -t testworkflow-toolkit:dev --target dist \
  -f build/_local/testworkflow-toolkit.Dockerfile .
```

### CI Usage

Use `tilt ci` to run the environment in CI mode. This auto-triggers the smoke test and exits with a non-zero code on failure:

```bash
tilt ci
```

## Related Documentation

- [Testkube Documentation](https://docs.testkube.io)
- [Tilt Documentation](https://docs.tilt.dev)
- [Helm Chart README](./k8s/helm/testkube/README.md)
- [Contributing Guide](./CONTRIBUTING.md)
