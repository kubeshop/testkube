# Testkube OSS Agent — Local Development with Tilt
#
# Prerequisites:
#   - Docker (with buildx)
#   - Kubernetes cluster (kind, k3d, minikube, Docker Desktop)
#   - Tilt (https://tilt.dev)
#   - Helm 3.x
#
# Quick Start:
#   ./scripts/tilt-cluster.sh     # Create a kind cluster (one-time, optional)
#   tilt up                       # Start development environment
#
# Options:
#   tilt up -- --debug            # Build with Delve debugger (attach on :56268)
#   tilt up -- --db=mongo         # Use MongoDB instead of PostgreSQL
#   tilt up -- --db=both          # Use both MongoDB and PostgreSQL
#   tilt up -- --no-live-reload   # Force Docker-only builds (skip local Go compile)
#
# CI mode:
#   tilt ci                       # Run with auto smoke tests, exit on success/failure

# =============================================================================
# Configuration
# =============================================================================

load('ext://restart_process', 'docker_build_with_restart', 'custom_build_with_restart')
load('ext://uibutton', 'cmd_button', 'location')

config.define_bool("live-reload", usage="Force live reload on/off (default: auto-detect Go toolchain)")
config.define_bool("debug", usage="Build with debug symbols and expose Delve debugger ports")
config.define_string("db", usage="Database backend: mongo, postgres, both (default: postgres)")
cfg = config.parse()

NAMESPACE = "testkube-dev"
HELM_RELEASE_NAME = "testkube"
HELM_CHART_PATH = "./k8s/helm/testkube"

# Database backend
db_backend = cfg.get('db', 'postgres')
if db_backend not in ['mongo', 'postgres', 'both']:
    fail("Invalid --db value '%s'. Must be: mongo, postgres, both" % db_backend)

# Detect Go toolchain for live reload
has_go = str(local('which go >/dev/null 2>&1 && echo yes || echo no', quiet=True)).strip() == 'yes'
go_arch = ''
compile_env = ''
if has_go:
    go_arch = str(local('go env GOARCH', quiet=True)).strip()
    compile_env = 'CGO_ENABLED=0 GOOS=linux GOARCH=' + go_arch

# Resolve live_reload: explicit flag overrides auto-detect
live_reload_flag = cfg.get('live-reload', None)
if live_reload_flag != None:
    live_reload = bool(live_reload_flag) and has_go
else:
    live_reload = has_go

# Debug mode: Delve debugger, gcflags to disable optimizations
debug = bool(cfg.get('debug', False))
build_target = 'debug' if debug else 'dist'
# live_reload needs 'live' target (busybox) — distroless has no shell for restart_process
live_target = 'debug' if debug else 'live'

if debug and live_reload:
    go_gcflags = '-gcflags="all=-N -l"'
else:
    go_gcflags = ''

# Status
print("""
-----------------------------------------------------------------
  Testkube — OSS Agent Local Development
-----------------------------------------------------------------
""".strip())
print("")
if live_reload:
    print("  Live reload: enabled (GOARCH=" + go_arch + ")")
else:
    print("  Live reload: disabled" + ("" if not has_go else " (use --live-reload to enable)"))
if debug:
    print("  Debug mode:  enabled (Delve on :56268)")
print("  Database:    " + db_backend)

# =============================================================================
# Safety Checks
# =============================================================================

allow_k8s_contexts([
    "docker-desktop",
    "docker-for-desktop",
    "minikube",
    "kind-kind",
    "kind-testkube-dev",
    "k3d-testkube-dev",
    "rancher-desktop",
])

docker_prune_settings(
    disable=False,
    max_age_mins=120,
    num_builds=5,
    keep_recent=2,
)

# Prevent Tiltfile reload loops from helm dependency downloads
watch_settings(ignore=[
    "k8s/helm/testkube/charts/*.tgz",
    "k8s/helm/testkube/charts/**",
    "k8s/helm/testkube/Chart.lock",
])

# =============================================================================
# Docker Builds
# Image names match what the Helm chart produces so Tilt auto-injects them.
# =============================================================================

# --- Agent server: live reload (binary sync) vs full Docker build ---

if live_reload:
    local_resource('compile:agent-server',
        cmd=compile_env + ' go build ' + go_gcflags + ' -o build/_local/agent-server ./cmd/api-server',
        deps=['cmd/api-server', 'pkg', 'internal'],
        ignore=['**/*_test.go', '**/testdata/**'],
        labels=['compile'],
    )

    # restart_file must NOT be under /tmp — the Helm chart mounts an emptyDir there,
    # which hides the sentinel file the extension creates at Docker build time.
    custom_build_with_restart('kubeshop/testkube-api-server',
        command='docker build -t $EXPECTED_REF -f build/_local/agent-server.Dockerfile --target ' + live_target + ' .',
        deps=['build/_local/agent-server', 'build/_local/agent-server.Dockerfile', 'go.mod', 'go.sum'],
        entrypoint=['/testkube/agent-server'],
        live_update=[sync('build/_local/agent-server', '/testkube/agent-server')],
        restart_file='/.restart-proc',
    )
else:
    docker_build('kubeshop/testkube-api-server', '.',
        dockerfile='build/_local/agent-server.Dockerfile',
        target=build_target,
        only=['cmd/api-server', 'pkg', 'internal', 'go.mod', 'go.sum', 'LICENSE'],
    )

# --- TestWorkflow images: always full Docker build ---
# match_in_env_vars=True because the API server references these via env vars, not pod specs.
# tw-init/tw-toolkit import across the entire monorepo so only= is impractical.
docker_build('kubeshop/testkube-tw-init', '.',
    dockerfile='build/_local/testworkflow-init.Dockerfile',
    target=build_target,
    match_in_env_vars=True,
)

docker_build('kubeshop/testkube-tw-toolkit', '.',
    dockerfile='build/_local/testworkflow-toolkit.Dockerfile',
    target=build_target,
    match_in_env_vars=True,
)

# =============================================================================
# Namespace Setup
# =============================================================================

local_resource(
    "create-namespace",
    cmd="kubectl create namespace {} --dry-run=client -o yaml | kubectl apply -f -".format(NAMESPACE),
    labels=["setup"],
)

# =============================================================================
# Helm Deployment
# =============================================================================

local("helm dependency build {} 2>/dev/null || helm dependency update {}".format(HELM_CHART_PATH, HELM_CHART_PATH), quiet=True, echo_off=True)

use_mongo = db_backend in ['mongo', 'both']
use_postgres = db_backend in ['postgres', 'both']

helm_sets = [
    "testkube-api.analyticsEnabled=false",
    "testkube-api.enableDebugMode=true",
    "testkube-api.next.controllers.enabled=true",
    # Standalone mode (no cloud connection)
    "testkube-api.cloud.key=",
    "testkube-api.testConnection.enabled=false",
]

if use_mongo:
    helm_sets += [
        "mongodb.enabled=true",
        "testkube-api.mongodb.enabled=true",
    ]
else:
    helm_sets += [
        "mongodb.enabled=false",
        "testkube-api.mongodb.enabled=false",
    ]

if use_postgres:
    helm_sets += [
        "postgresql.enabled=true",
        "testkube-api.postgresql.enabled=true",
        "testkube-api.postgresql.dsn=postgres://testkube:postgres5432@testkube-postgresql:5432/backend?sslmode=disable",
    ]
else:
    helm_sets += [
        "postgresql.enabled=false",
        "testkube-api.postgresql.enabled=false",
    ]

# Optional local overrides (not committed)
values_files = []
if os.path.exists("./tilt-values.yaml"):
    values_files = ["./tilt-values.yaml"]

k8s_yaml(
    helm(
        HELM_CHART_PATH,
        name=HELM_RELEASE_NAME,
        namespace=NAMESPACE,
        values=values_files,
        set=helm_sets,
    )
)

# =============================================================================
# Resource Configuration
# =============================================================================

# API Server
api_port_forwards = [
    port_forward(8088, 8088, name="HTTP API"),
    port_forward(8089, 8089, name="gRPC"),
]
if debug:
    api_port_forwards.append(port_forward(56268, 56268, name="Delve Debugger"))

api_deps = ["create-namespace"]
if live_reload:
    api_deps.append("compile:agent-server")

k8s_resource(
    "testkube-api-server",
    port_forwards=api_port_forwards,
    labels=["testkube"],
    resource_deps=api_deps,
)

# Dependencies
if use_postgres:
    k8s_resource(
        "testkube-postgresql",
        port_forwards=[port_forward(5432, 5432, name="PostgreSQL")],
        labels=["dependencies"],
    )

if use_mongo:
    k8s_resource(
        "testkube-mongodb",
        port_forwards=[port_forward(27017, 27017, name="MongoDB")],
        labels=["dependencies"],
    )

k8s_resource(
    "testkube-minio-testkube-dev",
    port_forwards=[
        port_forward(9000, 9000, name="MinIO API"),
        port_forward(9001, 9090, name="MinIO Console"),
    ],
    labels=["dependencies"],
)

k8s_resource(
    "testkube-nats",
    port_forwards=[port_forward(4222, 4222, name="NATS")],
    labels=["dependencies"],
)

# MinIO bucket creation — compensates for a race condition where the API server starts
# before MinIO is ready, causing its startup bucket creation to fail silently.
local_resource(
    "create-minio-buckets",
    cmd="""
        set -e
        echo "Waiting for MinIO to be ready..."
        kubectl wait --for=condition=ready pod -l app=testkube-minio-testkube-dev -n {} --timeout=120s
        sleep 3
        echo "Creating MinIO buckets..."
        kubectl delete job minio-bucket-setup -n {} --ignore-not-found=true
        kubectl apply -n {} -f - <<'YAML'
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
        kubectl wait --for=condition=complete job/minio-bucket-setup -n {} --timeout=60s
        echo "MinIO buckets setup complete."
    """.format(NAMESPACE, NAMESPACE, NAMESPACE, NAMESPACE),
    labels=["setup"],
    resource_deps=["testkube-minio-testkube-dev"],
)

# =============================================================================
# Verification
# =============================================================================

local_resource(
    'health-check',
    cmd='curl -sf http://localhost:8088/health && echo "HEALTHY" || { echo "NOT READY: API server is not responding on :8088"; exit 1; }',
    labels=['verify'],
    auto_init=False,
    resource_deps=['testkube-api-server'],
)

is_ci = config.tilt_subcommand == 'ci'

local_resource(
    'smoke-test',
    cmd='curl -sf http://localhost:8088/health || { echo "FAILED: health endpoint not responding"; exit 1; } && curl -sf http://localhost:8088/v1/test-workflows > /dev/null || { echo "FAILED: /v1/test-workflows not responding"; exit 1; } && echo "SMOKE TEST PASSED"',
    labels=['verify'],
    auto_init=is_ci,
    trigger_mode=TRIGGER_MODE_AUTO if is_ci else TRIGGER_MODE_MANUAL,
    resource_deps=['testkube-api-server'],
)

if not is_ci:
    cmd_button('health-check:run',
        argv=['sh', '-c', 'curl -sf http://localhost:8088/health && echo "HEALTHY" || { echo "UNHEALTHY"; exit 1; }'],
        resource='testkube-api-server',
        icon_name='favorite',
        text='Health Check',
    )

# =============================================================================
# Development Utilities
# =============================================================================

local_resource("make test", cmd="make test", labels=["dev"], auto_init=False, trigger_mode=TRIGGER_MODE_MANUAL)
local_resource("make lint", cmd="make lint", labels=["dev"], auto_init=False, trigger_mode=TRIGGER_MODE_MANUAL)

# Code generation
for target in ["generate", "generate-protobuf", "generate-openapi", "generate-mocks", "generate-sqlc", "generate-crds"]:
    local_resource(
        "make " + target,
        cmd="make " + target,
        labels=["generate"],
        auto_init=False,
        trigger_mode=TRIGGER_MODE_MANUAL,
    )

# =============================================================================
# Output
# =============================================================================

services_text = """
Ports:
  Tilt UI:     http://localhost:10350
  API:         http://localhost:8088
  gRPC:        localhost:8089
  MinIO:       http://localhost:9000 (minio/minio123)
  NATS:        localhost:4222"""

if use_postgres:
    services_text += "\n  PostgreSQL:  localhost:5432 (testkube/postgres5432)"
if use_mongo:
    services_text += "\n  MongoDB:     localhost:27017"
if debug:
    services_text += "\n  Delve:       localhost:56268"

print(services_text)
print("""
Quick Start:
  testkube config api-server-uri http://localhost:8088
  testkube get testworkflows
  testkube run testworkflow <name>
""")
