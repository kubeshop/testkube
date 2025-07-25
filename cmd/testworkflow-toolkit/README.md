# TestWorkflow Toolkit

The TestWorkflow Toolkit provides helper utilities for test workflows. It's injected into test containers and communicates with the init process to coordinate test execution, artifact collection, and service management.

## Overview

The toolkit works with the TestWorkflow Init process to provide comprehensive test orchestration:

- **Init Process**: Orchestrates test execution (entrypoint, retry logic, state management)
- **Toolkit**: Provides utilities for tests (artifacts, git operations, services, parallel execution)

## Environment Configuration

The toolkit receives configuration via environment variables:

| Variable | Purpose                     | Source              |
|----------|-----------------------------|---------------------|
| `TK_REF` | Current step reference ID   | Set by init process |
| `TK_CFG` | JSON workflow configuration | Set by init process |
| `DEBUG`  | Enable debug logging        | Optional            |

### TK_CFG Structure

Contains workflow metadata and execution details:
```json
{
  "workflow": {"name": "test-workflow"},
  "execution": {"id": "exec-123", "namespace": "testkube"},
  "worker": {"namespace": "testkube"}
}
```

## Commands

### Open Source Commands

#### `artifacts <paths...>`
Collect test artifacts with optional compression and cloud upload.

```bash
# Basic usage
testworkflow-toolkit artifacts "reports/*.xml" "logs/*.log"

# With compression  
testworkflow-toolkit artifacts --compress=tar.gz "test-results/"

# Custom artifact ID
testworkflow-toolkit artifacts --id=custom-id "output/"
```

**Flags**:
- `--compress`: Compression type (`tar.gz`, `tgz`, `none`)
- `--id`: Custom artifact identifier
- `--unpack`: Auto-extract archives in cloud storage
- `--mount`: Additional mount paths

#### `clone <repository>`
Clone git repositories with authentication support.

```bash
# Basic clone
testworkflow-toolkit clone https://github.com/user/repo.git

# With authentication and branch
testworkflow-toolkit clone https://github.com/private/repo.git \
  --token $GITHUB_TOKEN --branch main

# Sparse checkout
testworkflow-toolkit clone https://github.com/user/repo.git \
  --paths "src/,tests/" --branch develop
```

**Authentication Types**:
- `--token`: Token-based (GitHub, GitLab)
- `--username`/`--password`: Basic auth
- `--ssh-key`: SSH key path

#### `tarball <operation> [args...]`
Create or extract tarball archives.

```bash
# Create compressed tarball
testworkflow-toolkit tarball create --compress archive.tar.gz src/ tests/

# Extract tarball
testworkflow-toolkit tarball extract archive.tar.gz /destination/
```

#### `transfer <source:patterns=url>`
Transfer files using pattern matching.

```bash
# Single transfer
testworkflow-toolkit transfer "/data:*.txt,*.log=http://storage/upload"

# Multiple transfers
testworkflow-toolkit transfer \
  "/results:*.xml=http://storage/results" \
  "/logs:**/*.log=http://storage/logs"
```

### Pro Commands (Testkube Pro License Required)

#### `execute`
Execute other tests or workflows from within a workflow.

```bash
# Execute tests with matrix/sharding support
testworkflow-toolkit execute --test='{"name":"api-test","count":3}' \
  --workflow='{"name":"e2e-suite","matrix":{"browser":["chrome","firefox"]}}'

# Parallel execution with custom parallelism
testworkflow-toolkit execute --parallelism=5 --async \
  --test='{"name":"load-test"}' --workflow='{"name":"integration-suite"}'
```

**Features**:
- Matrix and sharding support for scaling tests
- Parallel execution with configurable parallelism
- Async mode for fire-and-forget execution
- Transfer server for file sharing between executions

#### `services <ref>`
Manage accompanying services (databases, APIs, etc.) alongside tests.

```bash
# Start services for a group
testworkflow-toolkit services --group=test-group \
  "db=<service-spec>" "api=<service-spec>"
```

**Service Management**:
- Automatic readiness probing
- Resource management and cleanup
- IP assignment and networking
- Parallel service startup with configurable policies

#### `parallel <spec>`
Execute multiple operations in parallel with advanced orchestration.

```bash
# Run parallel workflows with matrix
testworkflow-toolkit parallel '<parallel-spec-json>'
```

**Features**:
- Matrix and sharding support
- Configurable parallelism
- Transfer server integration
- Log collection with conditional saving
- Automatic resource cleanup
- Resume/pause orchestration

#### `kill <ref>`
Terminate and clean up services or parallel operations.

```bash
# Kill services with log collection
testworkflow-toolkit kill service-group --logs="db=failed" --logs="api=always"
```

**Cleanup Features**:
- Selective log collection based on conditions
- Graceful resource cleanup
- Error reporting and status tracking

## Integration with Init Process

The toolkit integrates with the init process through:

1. **Environment Variables**: Configuration passed via `TK_CFG` and `TK_REF`
2. **Shared Filesystem**: Access to volumes for artifacts and state
3. **State Coordination**: Synchronization through the state file at `/.tktw/state`

## Exit Codes

- **0**: Success
- **1**: General error 
- **137**: Terminated by signal (SIGINT/SIGTERM)

## Internal Paths

The toolkit uses these paths for coordination:

| Path               | Purpose                                  |
|--------------------|------------------------------------------|
| `/.tktw/`          | Root directory for TestWorkflow files    |
| `/.tktw/state`     | State file for init process coordination |
| `/.tktw/transfer/` | Temporary directory for file transfers   |
