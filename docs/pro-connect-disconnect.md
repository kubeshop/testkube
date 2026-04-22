# Switching Between Testkube OSS and Pro Modes

Testkube CLI provides two commands to seamlessly switch your existing Testkube installation between **OSS (standalone)** mode and **Pro (cloud-connected)** mode. You can switch back and forth at any time without losing your data.

---

## `testkube pro connect`

Upgrades your Testkube OSS installation to **Pro mode** by connecting it to the Testkube Control Plane. The command automatically migrates your historical execution data, installs a cloud runner agent, and scales down the OSS services that are no longer needed.

### What happens when you run `pro connect`

1. **Data export** — All TestWorkflow execution data (metadata, logs, and workflow sequence numbers) is exported from the local agent as a `.tar.gz` archive.
2. **Agent installation** — A cloud runner agent is installed into the cluster via Helm, with all capabilities enabled (runner, listener, GitOps, webhooks).
3. **OSS services scale-down** — The standalone API server, MinIO, NATS, and the active database (MongoDB or PostgreSQL) are scaled to zero replicas. Nothing is deleted; the resources stay in the cluster for a potential rollback.
4. **Data import** — The exported archive is uploaded to the Testkube Control Plane. Processing happens asynchronously and may take a few minutes depending on the volume of data.
5. **CLI context switch** — The local CLI configuration is updated to point to the Pro environment.

### Usage

```bash
testkube pro connect [agent-name] \
  --api-key <key> \
  --org-id <organization-id> \
  --env-id <environment-id> \
  --agent-uri <agent-grpc-uri>
```

If `agent-name` is omitted the agent is registered as **default-oss**.

### Key flags

| Flag | Description |
|------|-------------|
| `--api-key`, `-k` | **(Required)** API key for authenticating with Testkube Pro. |
| `--org-id` | **(Required)** Testkube Pro organization ID. |
| `--env-id` | **(Required)** Testkube Pro environment ID. |
| `--agent-uri` | **(Required)** gRPC URI of the Testkube Pro agent endpoint. |
| `--skip-export` | Skip the automatic data export/import step. Your data stays in the local database and can be exported manually later. |
| `--since` | Export only executions created after the specified date. Useful for large datasets that would exceed size limits. Accepts `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SSZ`. Example: `--since 2025-01-01`. |
| `--dry-run` | Render Helm commands without executing them. Skips all post-install steps (scale-down, data import). |
| `--root-domain` | Override the default Testkube Pro root domain (advanced). |
| `--runner` | Enable the runner component (enabled by default during connect). |
| `--listener` | Enable the listener component (enabled by default during connect). |
| `--gitops` | Enable the GitOps capability (enabled by default during connect). |
| `--webhooks` | Enable the webhooks capability (enabled by default during connect). |
| `--global` | Register the agent as a global agent (enabled by default during connect). |
| `--create` | Auto-create the agent in the control plane (enabled by default during connect). |
| `--version` | Specify the agent chart version (defaults to latest). |
| `--execution-namespace`, `-N` | Namespace for running test executions (defaults to the installation namespace). |

### Examples

**Basic connect (exports all data):**

```bash
testkube pro connect \
  --api-key tkcapi_XXXX \
  --org-id tkcorg_XXXX \
  --env-id tkcenv_XXXX \
  --agent-uri agent.testkube.io:443
```

**Connect with data export limited to last 6 months:**

```bash
testkube pro connect \
  --api-key tkcapi_XXXX \
  --org-id tkcorg_XXXX \
  --env-id tkcenv_XXXX \
  --agent-uri agent.testkube.io:443 \
  --since 2025-01-01
```

**Connect without migrating execution history:**

```bash
testkube pro connect \
  --api-key tkcapi_XXXX \
  --org-id tkcorg_XXXX \
  --env-id tkcenv_XXXX \
  --agent-uri agent.testkube.io:443 \
  --skip-export
```

**Dry-run (preview Helm commands only):**

```bash
testkube pro connect \
  --api-key tkcapi_XXXX \
  --org-id tkcorg_XXXX \
  --env-id tkcenv_XXXX \
  --agent-uri agent.testkube.io:443 \
  --dry-run
```

### Error handling

- **Export too large** — If the export archive exceeds the server size limit, the CLI suggests using `--since` to narrow the date range. You are prompted to continue without the export.
- **Export or import failure** — The CLI warns you and asks whether to continue connecting. The data remains in your local database and can be exported later. If the import fails after connecting, the archive path is printed so you can retry manually.
- **Scale-down failure** — Failures when scaling down individual services are reported but do not abort the overall connect flow.

---

## `testkube pro disconnect`

Reverts your Testkube installation from **Pro mode** back to **OSS (standalone) mode**. The command uninstalls the cloud runner agent, removes the agent record from the control plane, restores the previously scaled-down OSS services, and resets the CLI context.

### What happens when you run `pro disconnect`

1. **Runner uninstall** — The cloud runner Helm release that was installed by `pro connect` is removed.
2. **Agent cleanup** — The agent record is deleted from the Testkube Control Plane.
3. **OSS services restore** — The API server, MinIO, NATS, and the active database are scaled back up to their configured replica counts.
4. **CLI context reset** — The local CLI configuration is switched back to kubeconfig mode and all Pro-related settings are cleared.

### Usage

```bash
testkube pro disconnect
```

### Key flags

| Flag | Default | Description |
|------|---------|-------------|
| `--minio-replicas` | `1` | Number of MinIO replicas to restore. |
| `--mongo-replicas` | `1` | Number of MongoDB replicas to restore. |
| `--postgres-replicas` | `1` | Number of PostgreSQL replicas to restore. |

> **Tip:** If your OSS installation used custom replica counts before connecting, pass the original values so that disconnect restores them correctly. For example: `--mongo-replicas 3`.

### Examples

**Basic disconnect:**

```bash
testkube pro disconnect
```

**Disconnect with custom replica counts:**

```bash
testkube pro disconnect \
  --mongo-replicas 3 \
  --minio-replicas 2
```

### Error handling

- **Runner uninstall failure** — Reported as a warning; disconnect continues to restore OSS services.
- **Agent deletion failure** — Reported as a warning; disconnect continues.
- **Scale-up failure** — Failures when restoring individual services are reported but do not abort the disconnect.
- **Config save failure** — If the CLI config file cannot be updated, you are given instructions to manually remove the Pro context fields.

---

## Data safety

Both commands are designed to be non-destructive:

- **`pro connect`** scales down OSS services to zero replicas but does **not** delete any Kubernetes resources, PersistentVolumeClaims, or data. Your local databases and object storage remain intact.
- **`pro disconnect`** scales the same services back up so you can resume using Testkube in OSS mode with all original data in place.
- Execution data migrated to the control plane during `pro connect` is kept separately from any new executions created in Pro mode.

You can switch between modes as many times as needed.

---

## Prerequisites

| Requirement | Details |
|-------------|---------|
| **Testkube CLI** | Latest version with `pro connect` / `pro disconnect` support. |
| **kubectl** | Configured and pointing to the target cluster. |
| **helm** | Required for agent installation and uninstallation (v3+). |
| **Testkube OSS** | An existing Testkube OSS installation in the cluster. |
| **Pro credentials** | API key, organization ID, and environment ID from your Testkube Pro account (for `pro connect` only). |

---

## Quick reference

| Action | Command |
|--------|---------|
| Connect to Pro (full data migration) | `testkube pro connect --api-key <key> --org-id <org> --env-id <env> --agent-uri <uri>` |
| Connect to Pro (skip data migration) | `testkube pro connect --api-key <key> --org-id <org> --env-id <env> --agent-uri <uri> --skip-export` |
| Connect to Pro (recent data only) | `testkube pro connect --api-key <key> --org-id <org> --env-id <env> --agent-uri <uri> --since 2025-01-01` |
| Disconnect from Pro | `testkube pro disconnect` |
| Disconnect with custom replicas | `testkube pro disconnect --mongo-replicas 3 --minio-replicas 2` |
