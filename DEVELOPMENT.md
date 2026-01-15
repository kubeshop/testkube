# Local Development Guide

This guide explains how to set up and use the local development environment for Testkube using [Tilt](https://tilt.dev).

This Tilt-driven development environment builds and deploys:
- **testkube-api-server** - The main API server (live reload on code changes)
- **testworkflow-init** - Init container for Test Workflow execution (built as local resource)
- **testworkflow-toolkit** - Runtime utilities for Test Workflow containers (built as local resource)

> **Note**: This guide applies specifically to developing the **standalone/open-source Testkube agent**, and not to development of the agent when it is connected to the Testkube Control Plane (in which case storage/etc is managed there instead).

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker** with BuildX support
- **Kubernetes cluster** - one of:
  - [kind](https://kind.sigs.k8s.io/) (recommended - images are automatically loaded)
  - [minikube](https://minikube.sigs.k8s.io/)
  - [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes enabled
  - [Rancher Desktop](https://rancherdesktop.io/)
- **[Tilt](https://docs.tilt.dev/install.html)** v0.30.0 or later
- **[Helm](https://helm.sh/docs/intro/install/)** v3.x
- **Go** 1.25+ (for running tests locally)

## Quick Start

1. **Start your Kubernetes cluster** (if not already running):

   ```bash
   # For kind
   kind create cluster --name testkube-dev

   # For minikube
   minikube start
   ```

2. **Run Tilt**:

   ```bash
   cd /path/to/testkube
   tilt up
   ```

   This will:
   - Create the `testkube-dev` namespace
   - Update Helm dependencies automatically
   - Build 3 images: `testkube-api-server-dev`, `testworkflow-init-dev`, `testworkflow-toolkit-dev`
   - Deploy the Testkube helm chart with all dependencies (PostgreSQL, MinIO, NATS)
   - Create MinIO buckets for artifacts and logs
   - Set up port forwards for easy local access
   - For kind clusters: automatically load images into the cluster

3. **Access the Tilt UI**:

   Open http://localhost:10350 in your browser to see the Tilt dashboard.

4. **Configure the Testkube CLI** to use your local API:

   ```bash
   testkube config api-server-uri http://localhost:8088
   ```

5. **Verify the setup**:

   ```bash
   testkube get testworkflows
   ```

## Architecture

### Enabled Features

The local development setup enables the following features:

- **K8s Controllers** (`ENABLE_K8S_CONTROLLERS=true`) - Watches for `TestWorkflowExecution` CRDs to trigger workflow runs
- **Debug Mode** - Enables verbose logging and debugging endpoints
- **Delve Debugging** - All images built with debug target for remote debugging

### Components

The local development setup deploys the following components:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        testkube-dev namespace                           │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌─────────────────────┐    ┌─────────────────────┐                   │
│  │  testkube-api-server │◄──►│     PostgreSQL      │                   │
│  │    (your code)       │    │   :5432             │                   │
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
│  Test Workflow Execution (spawned by API server):                      │
│  ┌─────────────────────┐    ┌─────────────────────┐                   │
│  │  testworkflow-init  │───►│ testworkflow-toolkit │                   │
│  │  (init container)   │    │  (runtime utilities) │                   │
│  └─────────────────────┘    └─────────────────────┘                   │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Images Built

All images are built with local-only names (no registry prefix) and use Dockerfiles in `build/_local/`:

| Image | Dockerfile | Build Type | Description |
|-------|------------|------------|-------------|
| `testkube-api-server-dev` | `build/_local/agent-server.Dockerfile` | `docker_build` (Tilt-tracked) | Main API server - live reloads on code changes |
| `testworkflow-init-dev` | `build/_local/testworkflow-init.Dockerfile` | `local_resource` | Init container for TW execution - rebuild manually via Tilt UI |
| `testworkflow-toolkit-dev` | `build/_local/testworkflow-toolkit.Dockerfile` | `local_resource` | Runtime utilities - rebuild manually via Tilt UI |

**Note**: The Test Workflow images (`testworkflow-init-dev` and `testworkflow-toolkit-dev`) are built as Tilt local resources, not tracked docker builds. This means they won't automatically rebuild when code changes - you need to trigger a rebuild from the Tilt UI when needed. For kind clusters, images are automatically loaded using `kind load docker-image`.

## Port Forwards

When Tilt is running, the following ports are forwarded to your localhost:

| Service | Port | Description |
|---------|------|-------------|
| testkube-api-server | 8088 | HTTP REST API |
| testkube-api-server | 8089 | gRPC API |
| testkube-api-server | 56268 | Delve debugger |
| PostgreSQL | 5432 | Database |
| MinIO | 9000 | S3-compatible artifact storage |
| MinIO | 9001 | MinIO web console |
| NATS | 4222 | Message queue |

## Configuration

### Custom Helm Values

To customize the deployment, create a `tilt-values.yaml` file:

```bash
cp tilt-values.yaml.example tilt-values.yaml
```

Then edit `tilt-values.yaml` to override any helm values. Common customizations:

```yaml
# tilt-values.yaml

testkube-api:
  # Increase resource limits for heavy workloads
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi

  # Enable liveness/readiness probes (disabled by default for faster restarts)
  livenessProbe:
    enabled: true
  readinessProbe:
    enabled: true

# Enable the Testkube Operator if you need to test CRD functionality
testkube-operator:
  enabled: true
```

### Tiltfile Configuration

You can modify variables at the top of the `Tiltfile`:

```python
# Change the namespace
NAMESPACE = "testkube-dev"

# Change the helm release name
HELM_RELEASE_NAME = "testkube"

# Change the Helm chart path
HELM_CHART_PATH = "./k8s/helm/testkube"

# Change the image names (local-only names without registry prefix)
API_SERVER_IMAGE = "testkube-api-server-dev"
TW_INIT_IMAGE = "testworkflow-init-dev:latest"
TW_TOOLKIT_IMAGE = "testworkflow-toolkit-dev:latest"
```

The Tiltfile automatically detects kind clusters and loads images appropriately.

**Watch Settings**: The Tiltfile ignores Helm chart dependency files (`k8s/helm/testkube/charts/*.tgz`, `Chart.lock`) to prevent reload loops when Helm updates dependencies.

## Development Workflow

### Making Code Changes

**For API Server changes:**
1. Edit any Go files in `cmd/api-server/`, `pkg/`, `internal/`, or `api/`
2. Tilt automatically detects changes and triggers a rebuild
3. The new image is built using Docker with the `build/_local/agent-server.Dockerfile`
4. The deployment is updated with the new image
5. The API server restarts with your changes

**For Test Workflow image changes:**
1. Edit files in `cmd/testworkflow-init/` or `cmd/testworkflow-toolkit/`
2. In the Tilt UI, click on `build-tw-init` or `build-tw-toolkit` to trigger a manual rebuild
3. The images are rebuilt and (for kind) loaded into the cluster
4. New Test Workflow executions will use the updated images

### Running Tests

From the Tilt UI, you can trigger manual resources:

- **go-test**: Runs `go test` on the API server packages
- **go-vet**: Runs `go vet` for static analysis

Or run tests directly:

```bash
# Run all tests
make test

# Run specific package tests
go test ./cmd/api-server/... -v

# Run with race detection
go test ./pkg/... -race
```

### Viewing Logs

**Via Tilt UI**: Click on the `testkube-api-server` resource to see live logs.

**Via kubectl**:

```bash
kubectl logs -f -n testkube-dev deployment/testkube-api-server
```

### Debugging

**Delve debugging is enabled by default.** All images are built with the `debug` target which includes the Delve debugger.

The API server also runs with `enableDebugMode: true` for verbose logging.

**Debug Ports:**
| Image | Delve Port | Notes |
|-------|------------|-------|
| testkube-api-server | 56268 | Port-forwarded automatically |
| testworkflow-init | 56268 | Spawned dynamically during test execution |
| testworkflow-toolkit | 56300 | Spawned dynamically during test execution |

#### Connecting Your IDE

**VSCode** - Add to `.vscode/launch.json`:

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

#### Disabling Debug Mode

To use production images without Delve (faster startup), change `target="debug"` to `target="dist"` in the Tiltfile's `docker_build` call:

```python
docker_build(
    API_SERVER_IMAGE,
    context=".",
    dockerfile="build/_local/agent-server.Dockerfile",
    target="dist",  # Production build without Delve
    ...
)
```

For Test Workflow images, update the `docker build` commands in the `local_resource` definitions:

```python
local_resource(
    "build-tw-init",
    cmd="docker build -t testworkflow-init-dev:latest --target dist ...",
    ...
)
```

### Accessing PostgreSQL

```bash
# Connect with psql
psql -h localhost -p 5432 -U testkube -d backend
# Password: postgres5432

# Or use a GUI like pgAdmin, DBeaver, or TablePlus
# Connection string: postgresql://testkube:postgres5432@localhost:5432/backend
```

### Accessing MinIO

- **Web Console**: http://localhost:9001
- **Credentials**: `minio` / `minio123`
- **API Endpoint**: http://localhost:9000

**Note**: The Tiltfile automatically creates the required buckets (`testkube-artifacts`, `testkube-logs`) via a job after MinIO starts. This compensates for a race condition where the API server may start before MinIO is ready.

## Troubleshooting

### Build Fails with Architecture Mismatch

The Tiltfile automatically detects your machine's architecture. If you encounter issues:

```bash
# Check your architecture
uname -m

# The Tiltfile should detect:
# - arm64/aarch64 → linux/arm64
# - x86_64 → linux/amd64
```

### Helm Dependency Update Fails

If helm dependencies fail to update:

```bash
# Manually update dependencies
helm dependency update ./k8s/helm/testkube

# Or rebuild the lock file
rm ./k8s/helm/testkube/Chart.lock
helm dependency build ./k8s/helm/testkube
```

### Image Pull Errors

Since images are loaded locally (not pushed to a registry), ensure:

1. `imagePullPolicy: Never` is set (default in Tiltfile)
2. Your cluster can access locally loaded images:

   ```bash
   # For kind clusters, the Tiltfile automatically loads images using:
   # kind load docker-image <image-name>
   
   # For minikube, you may need to use minikube's docker daemon:
   eval $(minikube docker-env)
   
   # Then restart Tilt so it builds inside minikube's Docker
   ```

### Port Already in Use

If a port is already in use:

```bash
# Find the process using the port
lsof -i :8088

# Kill it or change the port in the Tiltfile
```

### PostgreSQL Connection Issues

If the API server can't connect to PostgreSQL:

```bash
# Check PostgreSQL is running
kubectl get pods -n testkube-dev | grep postgresql

# Check PostgreSQL logs
kubectl logs -n testkube-dev statefulset/testkube-postgresql
```

### Cleaning Up

To completely remove the development environment:

```bash
# Stop Tilt
tilt down

# Delete the namespace
kubectl delete namespace testkube-dev

# Delete kind cluster (if using kind)
kind delete cluster --name testkube-dev
```

## Advanced Topics

### Using a Different Kubernetes Context

The Tiltfile allows the following local Kubernetes contexts by default:
- `docker-desktop`
- `docker-for-desktop`
- `minikube`
- `kind-kind`
- `rancher-desktop`

```bash
# Set the context before running tilt
kubectl config use-context my-cluster

# Or specify in Tilt
tilt up --context my-cluster
```

To allow additional contexts, modify the `allow_k8s_contexts()` call in the Tiltfile.

### Building Images Manually

You can build the development images manually using the Dockerfiles in `build/_local/`:

```bash
# Build the API server image
docker build -t testkube-api-server-dev:latest --target debug \
  -f build/_local/agent-server.Dockerfile .

# Build Test Workflow init container
docker build -t testworkflow-init-dev:latest --target debug \
  -f build/_local/testworkflow-init.Dockerfile .

# Build Test Workflow toolkit
docker build -t testworkflow-toolkit-dev:latest --target debug \
  -f build/_local/testworkflow-toolkit.Dockerfile .

# For kind clusters, load the images:
kind load docker-image testkube-api-server-dev:latest
kind load docker-image testworkflow-init-dev:latest
kind load docker-image testworkflow-toolkit-dev:latest
```

### Running Multiple Instances

If you need multiple development environments:

```bash
# Create a separate Tiltfile or modify NAMESPACE
NAMESPACE = "testkube-dev-2"
```

## Related Documentation

- [Testkube Documentation](https://docs.testkube.io)
- [Tilt Documentation](https://docs.tilt.dev)
- [Helm Chart README](./k8s/helm/testkube/README.md)
- [Contributing Guide](./CONTRIBUTING.md)
