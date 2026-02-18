---
name: cli-build-and-test
description: Build, test, and run the Testkube CLI (kubectl-testkube) locally. Use when developing CLI commands, testing against local or cloud clusters, building release binaries, or debugging CLI behavior. Covers build commands, version injection, client modes, and local development workflow.
metadata:
  author: testkube
  version: "1.0"
---

# CLI Build & Test

Entry point: `cmd/kubectl-testkube/main.go`. Commands registered in `cmd/kubectl-testkube/commands/root.go`.

---

## Building

### Quick build (current platform)

```bash
make build-testkube-cli        # → bin/app/testkube
```

Or during development, skip the build entirely:

```bash
go run cmd/kubectl-testkube/main.go <command> [flags]
```

### Other build targets

| Target | Output |
|--------|--------|
| `make build-testkube-cli` | `bin/app/testkube` |
| `make build-kubectl-testkube-cli` | `bin/app/kubectl-testkube` |
| `make rebuild-kubectl-testkube-cli` | Deletes and rebuilds |
| `make build-all-platforms` | Cross-compile for linux/darwin/windows × amd64/arm64 |
| `make docker-build-cli` | Docker image via goreleaser |

### Version injection

Build flags inject version info via ldflags:

```
-X main.version=<VERSION>
-X main.commit=<COMMIT>
-X main.date=<DATE>
-X main.builtBy=<USER>
```

**Dev builds** default to `999.0.0-dev` which **bypasses cluster version upgrade checks** — this is intentional for local development.

---

## Testing

### Run tests

```bash
make unit-tests                # All unit tests
make test                      # Alias for unit-tests
make cover                     # Unit tests + HTML coverage report
```

### CLI tests are standard Go tests

Tests use `testing` + `testify/assert` with table-driven patterns. Key test files:
- `cmd/kubectl-testkube/commands/testworkflows/run_test.go` — workflow status mapping, config parsing
- `cmd/kubectl-testkube/commands/common/artifacts_test.go` — file copy utilities
- `cmd/kubectl-testkube/config/config_test.go` — config persistence

No dedicated CLI e2e test suite exists — test commands manually against a cluster.

---

## Client modes

The CLI supports 4 connection modes (`--client` flag):

| Mode | Flag | Description |
|------|------|-------------|
| **proxy** (default) | `--client=proxy` | kubectl port-forward to API server |
| **direct** | `--client=direct` | Direct HTTP to API server URL |
| **cluster** | `--client=cluster` | In-cluster client (for pods) |
| **cloud** | (set via `testkube login`) | Through Cloud/Pro control plane with OAuth |

Client factory: `cmd/kubectl-testkube/commands/common/client.go`.

---

## Local development workflow

### Option 1: Port-forward to cluster API

```bash
# Terminal 1: forward API server port
make port-forward-api          # kubectl port-forward svc/testkube-api-server 8088

# Terminal 2: use CLI (default proxy mode)
bin/app/testkube get testworkflows
```

### Option 2: Direct connection

```bash
bin/app/testkube --client=direct --api-uri=http://localhost:8088 get testworkflows
```

### Option 3: Cloud/Pro connection

```bash
bin/app/testkube login                    # or: testkube pro login
bin/app/testkube context set --org <org> --env <env>
bin/app/testkube get testworkflows
```

### Option 4: Local control plane (tk-dev)

```bash
make login-local    # Logs into local Control Plane (localhost:8099)
make devbox         # Starts dev agent in "devbox" namespace
```

---

## Debug commands

```bash
testkube debug agent           # Pods, services, logs from agent namespace
testkube debug controlplane    # Control plane component status
testkube debug oss             # OSS installation diagnostics
testkube diagnostics           # Full installation + license diagnostics
```

---

## Key environment variables

| Variable | Purpose |
|----------|---------|
| `TESTKUBE_API_URI` | Override API URI |
| `TESTKUBE_ANALYTICS_ENABLED` | Enable/disable telemetry |
| `NAMESPACE` | K8s namespace for port-forwarding (default: `testkube`) |
| `DEVBOX_NAMESPACE` | Namespace for devbox agent (default: `devbox`) |

## Config file

Stored at `~/.testkube/config.json`. Managed via `testkube config` subcommands. Contains namespace, API URI, context type, cloud context (org/env/tokens).
