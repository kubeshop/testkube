# Local Development Guide

This guide explains how to set up and use the local development environment for Testkube using [Tilt](https://tilt.dev).

The development environment builds and deploys:
- **testkube-api-server** - The main API server
- **testworkflow-init** - Init container for Test Workflow execution
- **testworkflow-toolkit** - Runtime utilities for Test Workflow containers

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker** with [BuildX](https://docs.docker.com/buildx/working-with-buildx/) support
- **Kubernetes cluster** - one of:
  - [kind](https://kind.sigs.k8s.io/) (recommended)
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
   - Build 3 images: `testkube-api-server`, `testworkflow-init`, `testworkflow-toolkit`
   - Deploy the Testkube helm chart with all dependencies (PostgreSQL, MinIO, NATS)
   - Set up port forwards for easy local access

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

| Image | Description |
|-------|-------------|
| `testkube-api-server-dev` | Main API server handling all Testkube operations |
| `testworkflow-init-dev` | Init container that sets up Test Workflow execution environments |
| `testworkflow-toolkit-dev` | Runtime utilities for artifact collection, parallel execution, etc. |

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

# Change the image name (useful if you have naming conflicts)
API_SERVER_IMAGE = "docker.io/testkube-api-server-dev"
```

## Development Workflow

### Making Code Changes

1. Edit any Go files in `cmd/api-server/`, `pkg/`, `internal/`, or `api/`
2. Tilt automatically detects changes and triggers a rebuild
3. The new image is built using `docker buildx bake` with caching
4. The deployment is updated with the new image
5. The API server restarts with your changes

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

To use production images without Delve (faster startup), change `target="debug"` to `target="dist"` in the Tiltfile:

```python
docker_build(
    API_SERVER_IMAGE,
    ...
    target="dist",  # Production build without Delve
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
   # For kind, images are loaded automatically
   # For minikube, you may need to use minikube's docker daemon:
   eval $(minikube docker-env)
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

```bash
# Set the context before running tilt
kubectl config use-context my-cluster

# Or specify in Tilt
tilt up --context my-cluster
```

### Building Other Components

The `docker-bake.hcl` file defines multiple build targets. To build other components:

```bash
# Build the CLI
docker buildx bake --file docker-bake.hcl cli

# Build Test Workflow init container
docker buildx bake --file docker-bake.hcl tw-init

# Build Test Workflow toolkit
docker buildx bake --file docker-bake.hcl tw-toolkit
```

### Connecting to Testkube Cloud (Pro)

To test cloud connectivity during local development:

```yaml
# tilt-values.yaml
testkube-api:
  cloud:
    key: "your-agent-key"
    url: "agent.testkube.io:443"
    orgId: "your-org-id"
    envId: "your-env-id"
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
