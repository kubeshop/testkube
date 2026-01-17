# -*- mode: Python -*-
# Tiltfile for local development of the standalone Testkube Agent (testkube-api-server)
#
# Prerequisites:
#   - Docker with buildx support
#   - Kubernetes cluster (e.g., kind, minikube, Docker Desktop)
#   - Tilt installed (https://tilt.dev)
#   - Helm 3.x installed
#
# Usage:
#   tilt up
#
# This will:
#   1. Build the testkube images (api-server, testworkflow-init, testworkflow-toolkit)
#   2. Deploy the testkube helm chart to the testkube-dev namespace
#   3. Automatically rebuild and update when Go files change

# ============================================================================
# Configuration
# ============================================================================
NAMESPACE = "testkube-dev"
HELM_RELEASE_NAME = "testkube"
HELM_CHART_PATH = "./k8s/helm/testkube"

# Image references - must match what helm template produces
# Using local-only names (no docker.io/ prefix) to avoid Docker Hub pull attempts
API_SERVER_IMAGE = "testkube-api-server-dev"

# Test Workflow images (built as local resources, not tracked by Tilt)
TW_INIT_IMAGE = "testworkflow-init-dev:latest"
TW_TOOLKIT_IMAGE = "testworkflow-toolkit-dev:latest"

# ============================================================================
# Cluster Configuration
# ============================================================================
# Allow common local Kubernetes contexts
allow_k8s_contexts([
    "docker-desktop",
    "docker-for-desktop",
    "minikube",
    "kind-kind",
    "rancher-desktop",
])

# ============================================================================
# Namespace Setup
# ============================================================================
local_resource(
    "create-namespace",
    cmd="kubectl create namespace {} --dry-run=client -o yaml | kubectl apply -f -".format(NAMESPACE),
    labels=["setup"],
)

# ============================================================================
# Build Configuration
# ============================================================================
# Common ignore patterns for all builds
COMMON_IGNORE = [
    ".git",
    ".github",
    "docs",
    "test",
    "assets",
    "choco",
    "tmp",
    "js",
    "proto",
    "scripts",
    "k8s/helm",
    "*.md",
    "*.yaml",
    "*.json",
    "Makefile",
    "Tiltfile",
]

# Build the testkube-api-server image (with Delve debugger)
# Delve listens on port 56268
docker_build(
    API_SERVER_IMAGE,
    context=".",
    dockerfile="build/_local/agent-server.Dockerfile",
    target="debug",  # Enable Delve debugging
    build_args={
        "VERSION": "dev",
        "GIT_SHA": "local",
    },
    ignore=COMMON_IGNORE,
)

# Build Test Workflow images as local resources
# These images are NOT directly used in k8s manifests - they're passed as configuration
# to the API server which spawns them dynamically for Test Workflow executions.
# We build them separately and tag them so they're available in Docker.
# For kind clusters, we also need to load them into the cluster.

# Detect if running on kind cluster
IS_KIND = "kind-" in str(local("kubectl config current-context 2>/dev/null || echo ''", quiet=True, echo_off=True))

local_resource(
    "build-tw-init",
    cmd="docker build -t testworkflow-init-dev:latest --target debug -f build/_local/testworkflow-init.Dockerfile . && " +
        ("kind load docker-image testworkflow-init-dev:latest" if IS_KIND else "true"),
    deps=[
        "cmd/testworkflow-init/",
        "pkg/",
        "go.mod",
        "go.sum",
        "build/_local/testworkflow-init.Dockerfile",
    ],
    labels=["build"],
    resource_deps=["create-namespace"],
)

local_resource(
    "build-tw-toolkit",
    cmd="docker build -t testworkflow-toolkit-dev:latest --target debug -f build/_local/testworkflow-toolkit.Dockerfile . && " +
        ("kind load docker-image testworkflow-toolkit-dev:latest" if IS_KIND else "true"),
    deps=[
        "cmd/testworkflow-toolkit/",
        "cmd/testworkflow-init/",
        "pkg/",
        "go.mod",
        "go.sum",
        "build/_local/testworkflow-toolkit.Dockerfile",
    ],
    labels=["build"],
    resource_deps=["create-namespace"],
)

# Manual rebuild of the API server image (bypasses automatic rebuild on file changes)
local_resource(
    "build-api-server",
    cmd="docker build -t {}:latest --target debug --build-arg VERSION=dev --build-arg GIT_SHA=local -f build/_local/agent-server.Dockerfile . && ".format(API_SERVER_IMAGE) +
        ("kind load docker-image {}:latest && ".format(API_SERVER_IMAGE) if IS_KIND else "") +
        "kubectl rollout restart deployment/testkube-api-server -n {}".format(NAMESPACE),
    labels=["build"],
    auto_init=True,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# ============================================================================
# Helm Dependency Update
# ============================================================================
# Ignore the downloaded chart archives to prevent Tiltfile reload loops
watch_settings(ignore=[
    "k8s/helm/testkube/charts/*.tgz",
    "k8s/helm/testkube/charts/**",
    "k8s/helm/testkube/Chart.lock",
])

# Run helm dependency build synchronously during Tiltfile load
local("helm dependency build {} 2>/dev/null || helm dependency update {}".format(HELM_CHART_PATH, HELM_CHART_PATH), quiet=True, echo_off=True)

# ============================================================================
# Helm Deployment
# ============================================================================
# Check if custom values file exists
values_files = []
if os.path.exists("./tilt-values.yaml"):
    values_files = ["./tilt-values.yaml"]

# Deploy the testkube helm chart
k8s_yaml(
    helm(
        HELM_CHART_PATH,
        name=HELM_RELEASE_NAME,
        namespace=NAMESPACE,
        values=values_files,
        set=[
            # API Server image (local-only, no registry prefix)
            "testkube-api.image.registry=",
            "testkube-api.image.repository=testkube-api-server-dev",
            "testkube-api.image.tag=latest",
            "testkube-api.image.pullPolicy=Never",
            # Test Workflow Init image (local-only, no registry prefix)
            "testkube-api.imageTwInit.registry=",
            "testkube-api.imageTwInit.repository=testworkflow-init-dev",
            "testkube-api.imageTwInit.tag=latest",
            # Test Workflow Toolkit image (local-only, no registry prefix)
            "testkube-api.imageTwToolkit.registry=",
            "testkube-api.imageTwToolkit.repository=testworkflow-toolkit-dev",
            "testkube-api.imageTwToolkit.tag=latest",
            # Development settings
            "testkube-api.analyticsEnabled=false",
            "testkube-api.enableDebugMode=true",
            # Enable K8s controllers (for TestWorkflowExecution CRD watching)
            "testkube-api.next.controllers.enabled=true",
            # Standalone mode (no cloud)
            "testkube-api.cloud.key=",
            # Database: Use PostgreSQL instead of MongoDB
            "mongodb.enabled=false",
            "postgresql.enabled=true",
            "testkube-api.mongodb.enabled=false",
            "testkube-api.postgresql.enabled=true",
            "testkube-api.postgresql.dsn=postgres://testkube:postgres5432@testkube-postgresql:5432/backend?sslmode=disable",
            # Other dependencies
            "testkube-operator.enabled=false",
            "testkube-api.testConnection.enabled=false",
        ],
    )
)

# ============================================================================
# Resource Configuration
# ============================================================================

# Testkube API Server
k8s_resource(
    "testkube-api-server",
    port_forwards=[
        port_forward(8088, 8088, name="HTTP API"),
        port_forward(8089, 8089, name="gRPC"),
        port_forward(56268, 56268, name="Delve Debugger"),
    ],
    labels=["testkube"],
    resource_deps=["create-namespace", "build-tw-init", "build-tw-toolkit"],
)

# PostgreSQL
k8s_resource(
    "testkube-postgresql",
    port_forwards=[
        port_forward(5432, 5432, name="PostgreSQL"),
    ],
    labels=["dependencies"],
)

# MinIO (artifact storage)
k8s_resource(
    "testkube-minio-testkube-dev",
    port_forwards=[
        port_forward(9000, 9000, name="MinIO API"),
        port_forward(9001, 9001, name="MinIO Console"),
    ],
    labels=["dependencies"],
)

# Create MinIO buckets after MinIO is ready
# This compensates for a race condition where the API server starts before MinIO is ready,
# causing its startup bucket creation to fail silently. By creating buckets here,
# they exist when the runner needs them for saving logs/artifacts.
local_resource(
    "create-minio-buckets",
    cmd="""
        set -e
        echo "Waiting for MinIO to be ready..."
        kubectl wait --for=condition=ready pod -l app=testkube-minio-testkube-dev -n testkube-dev --timeout=120s
        sleep 3
        echo "Creating MinIO buckets..."
        # Delete old job if exists
        kubectl delete job minio-bucket-setup -n testkube-dev --ignore-not-found=true
        # Create job to setup buckets (using correct service name from helm template)
        kubectl apply -n testkube-dev -f - <<'YAML'
apiVersion: batch/v1
kind: Job
metadata:
  name: minio-bucket-setup
spec:
  ttlSecondsAfterFinished: 60
  template:
    spec:
      containers:
      - name: mc
        image: minio/mc:latest
        command:
        - /bin/sh
        - -c
        - |
          mc alias set minio http://testkube-minio-service-testkube-dev:9000 minio minio123
          mc mb minio/testkube-artifacts --ignore-existing
          mc mb minio/testkube-logs --ignore-existing
          echo "Buckets created successfully"
      restartPolicy: Never
  backoffLimit: 3
YAML
        # Wait for job to complete
        kubectl wait --for=condition=complete job/minio-bucket-setup -n testkube-dev --timeout=60s
        echo "MinIO buckets setup complete."
    """,
    labels=["setup"],
    resource_deps=["testkube-minio-testkube-dev"],
)

# NATS
k8s_resource(
    "testkube-nats",
    port_forwards=[
        port_forward(4222, 4222, name="NATS"),
    ],
    labels=["dependencies"],
)


# ============================================================================
# Development Utilities
# ============================================================================

# Local resource to run tests via Make
local_resource(
    "make test",
    cmd="make test",
    labels=["dev"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to run linting via Make
local_resource(
    "make lint",
    cmd="make lint",
    labels=["dev"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# ============================================================================
# Code Generation
# ============================================================================

# Local resource to generate all code
local_resource(
    "make generate",
    cmd="make generate",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to generate protobuf code
local_resource(
    "make generate-protobuf",
    cmd="make generate-protobuf",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to generate OpenAPI models
local_resource(
    "make generate-openapi",
    cmd="make generate-openapi",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to generate mock files
local_resource(
    "make generate-mocks",
    cmd="make generate-mocks",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to generate sqlc queries
local_resource(
    "make generate-sqlc",
    cmd="make generate-sqlc",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# Local resource to generate Kubernetes CRDs
local_resource(
    "make generate-crds",
    cmd="make generate-crds",
    labels=["generate"],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL,
)

# ============================================================================
# Startup Banner
# ============================================================================
print("""
╔══════════════════════════════════════════════════════════════════════════════╗
║                    Testkube Local Development Environment                    ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  Namespace:    {namespace}                                                   ║
║                                                                              ║
║  Images Built (with Delve debugging, local-only names):                      ║
║    • testkube-api-server-dev      (API Server)         Delve: 56268          ║
║    • testworkflow-init-dev        (TW Init Container)  Delve: 56268          ║
║    • testworkflow-toolkit-dev     (TW Toolkit)         Delve: 56300          ║
║                                                                              ║
║  Services:                                                                   ║
║    API URL:      http://localhost:8088                                       ║
║    gRPC URL:     localhost:8089                                              ║
║    Delve:        localhost:56268 (API Server debugger)                       ║
║    PostgreSQL:   localhost:5432 (testkube/postgres5432)                      ║
║    MinIO:        http://localhost:9000 (minio/minio123)                      ║
║    NATS:         localhost:4222                                              ║
║                                                                              ║
║  Quick Start:                                                                ║
║    testkube config api-server-uri http://localhost:8088                      ║
║    testkube get testworkflows                                                ║
║    testkube run testworkflow <name>                                          ║
║                                                                              ║
║  Debug (VSCode/GoLand): Connect to localhost:56268                           ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
""".format(namespace=NAMESPACE))
