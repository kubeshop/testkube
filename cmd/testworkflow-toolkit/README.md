# TestWorkflow Toolkit

The TestWorkflow Toolkit is a CLI tool that runs inside Kubernetes containers to orchestrate TestWorkflow execution. It's the runtime component that makes TestWorkflows actually work.

## What is it?

The toolkit is a binary that gets injected into test containers to provide:
- **Parallel execution** - Run multiple test instances with matrix/sharding support
- **Artifact management** - Handle test outputs and logs
- **Service orchestration** - Manage sidecar services and dependencies
- **File transfers** - Share files between workflow steps
- **Execution control** - Pause, resume, and coordinate test execution

## Architecture

```
TestWorkflow Definition (YAML)
        ↓
TestWorkflow Controller (compiles to Job)
        ↓
Toolkit (runs inside Pod Container)
        ↓
Test Execution
```

The toolkit acts as the bridge between Kubernetes orchestration and actual test execution. It handles the complexity of distributed execution while presenting a simple interface to tests.

## Environment Variables

The toolkit reads configuration from these environment variables:

### Required Variables

- **`TK_REF`**: Unique reference ID for the current step/container. Used to identify which part of the workflow is executing.
  - Maps to state file: `.c` (current reference)
  - Example: `"step-1"`, `"run-tests"`

- **`TK_CFG`**: JSON-encoded internal configuration containing workflow metadata, execution details, and worker configuration.
  - Maps to state file: `.C` (internal config)
  - Contains the full configuration object (see below)

### Optional Variables

- **`TK_IP`**: IP address for network operations
- **`TK_FF_JUNIT_REPORT`**: Enable JUnit report parsing (default: false)
- **`DEBUG`**: Enable debug logging (set to "1")

### Internal Configuration (TK_CFG)

The `TK_CFG` environment variable contains a JSON object with:

```json
{
  "workflow": {
    "name": "test-workflow-name"
  },
  "execution": {
    "id": "execution-id",
    "name": "execution-name", 
    "number": 1,
    "scheduledAt": "2024-01-01T00:00:00Z",
    "disableWebhooks": false,
    "debug": false,
    "tags": {"env": "prod"}
  },
  "worker": {
    "namespace": "testkube"
  }
}
```

## Commands

The toolkit provides several commands accessible via CLI:

### Core Commands (Open Source)

#### 1. Artifacts Command
```bash
testworkflow-toolkit artifacts [flags] <paths...>
```
Collects files and directories as test artifacts. Supports glob patterns.

Flags:
- `--id`: Custom artifact ID
- `--compress`: Compression type (tar.gz, tgz, none)
- `--compress-cache`: Cache path for compression
- `--unpack`: Auto-extract archives in cloud storage
- `--mount`: Additional mount paths to check

Example:
```bash
# Collect all test reports
testworkflow-toolkit artifacts "reports/*.xml" "logs/*.log"

# Collect with compression
testworkflow-toolkit artifacts --compress=tar.gz "test-results/"
```

#### 2. Clone Command
```bash
testworkflow-toolkit clone <repository> [flags]
```
Clones git repositories with authentication support.

Flags:
- `--branch`: Specific branch/tag/commit to checkout
- `--paths`: Sparse checkout paths (comma-separated)
- `--auth-type`: Authentication type (header, token, username-password, ssh-key)
- `--token`: Authentication token
- `--username`: Git username
- `--password`: Git password  
- `--ssh-key`: SSH private key path

Example:
```bash
# Clone with authentication
testworkflow-toolkit clone https://github.com/private/repo.git \
  --token $GITHUB_TOKEN \
  --branch main
```

#### 3. Tarball Command
```bash
testworkflow-toolkit tarball <operation> [args...]
```
Creates or extracts tarball archives.

Operations:
- `create <archive> <paths...>`: Create a tarball
- `extract <archive> [destination]`: Extract a tarball

Flags:
- `--compress`: Enable gzip compression (for create)
- `--mount`: Additional mount paths to check

Example:
```bash
# Create compressed tarball
testworkflow-toolkit tarball create --compress archive.tar.gz src/ tests/

# Extract tarball
testworkflow-toolkit tarball extract archive.tar.gz /tmp/extracted/
```

#### 4. Transfer Command
```bash
testworkflow-toolkit transfer <source:patterns=url> [...]
```
Transfers files between locations using pattern matching.

Format: `source_path:pattern1,pattern2=destination_url`

Example:
```bash
# Transfer specific file types
testworkflow-toolkit transfer "/data:*.txt,*.log=http://storage/upload"

# Multiple transfers
testworkflow-toolkit transfer \
  "/results:*.xml=http://storage/results" \
  "/logs:**/*.log=http://storage/logs"
```

### Pro Commands (Licensed)

These commands require a Testkube Pro license:

#### 1. Execute Command (Pro)
```bash
testworkflow-toolkit execute [flags]
```
Executes other TestWorkflows or Tests with support for matrix, sharding, and parallel execution.

Flags:
- `--test, -t`: Test names to execute (can specify multiple)
- `--workflow, -w`: TestWorkflow names to execute (can specify multiple)
- `--parallelism, -p`: Number of executions to run simultaneously (default: 100)
- `--async`: Don't wait for results, just schedule executions
- `--base64`: Input is base64 encoded (used by processor to preserve expressions)

The command accepts execution data in two formats:
1. **Base64 encoded JSON** (with --base64): Preserves expressions like `{{ index + 1 }}` for runtime evaluation
2. **Command line flags**: Direct specification via --test and --workflow flags

Features:
- Execute multiple Tests and TestWorkflows in a single command
- Support for matrix parameters, sharding, and count for each execution
- Configurable parallelism to control resource usage
- Async mode for fire-and-forget execution
- Transfer files via tarball to executed workflows
- Automatic retry on API failures

Example usage:
```yaml
steps:
- name: run-integration-tests
  execute:
  - test: api-test
    matrix:
      endpoint: [users, products, orders]
  - workflow: e2e-test
    parallelism: 5

#### 2. Services Command (Pro)
```bash
testworkflow-toolkit services <ref> [flags]
```
Starts accompanying services that run alongside your tests. Services are defined in the TestWorkflow spec and support matrix, sharding, and parallel execution.

Flags:
- `--group, -g`: Services group reference (required)
- `--base64`: Input is base64 encoded (used by processor to preserve expressions)

The command accepts service definitions in two formats:
1. **Base64 encoded JSON** (preferred): Preserves expressions like `{{ matrix.browser }}` for runtime evaluation
2. **Legacy format**: `name=spec` pairs (backward compatibility)

Service instances support:
- Matrix parameters for running multiple variations
- Sharding for distributed workloads
- Readiness probes to ensure services are ready before tests start
- Timeout configuration
- Resource limits and requests

Example service spec in TestWorkflow:
```yaml
services:
  postgres:
    image: postgres:15
    env:
    - name: POSTGRES_PASSWORD
      value: mysecret
    readinessProbe:
      tcpSocket:
        port: 5432
      periodSeconds: 1
  redis:
    image: redis:7
    matrix:
      version: ["6", "7"]  # Creates 2 Redis instances

#### 3. Parallel Command (Pro)
```bash
testworkflow-toolkit parallel <spec> [flags]
```
Executes parallel workers with advanced orchestration capabilities.

Flags:
- `--base64`: Spec is base64 encoded (used by processor to preserve expressions)

The parallel command accepts a JSON specification that defines:
- **Matrix**: Parameter combinations for test variations
- **Shards**: Data partitioning across workers
- **Count/MaxCount**: Number of worker instances
- **Parallelism**: Maximum concurrent workers
- **Steps**: Workflow steps to execute in each worker
- **Transfer/Fetch**: File sharing between parent and workers
- **Logs**: Conditional log collection (always/never/failed)

Key features:
- Dynamic worker creation based on matrix/shard calculations
- Synchronized pause/resume across all workers
- Individual worker resource isolation
- Conditional log collection based on execution results
- Non-blocking status updates to prevent deadlocks
- Automatic cleanup of worker resources

Example parallel spec:
```yaml
parallel:
  matrix:
    browser: [chrome, firefox, safari]
    version: [latest, stable]
  parallelism: 3
  steps:
  - name: run-browser-test
    run:
      shell: |
        echo "Testing {{ matrix.browser }} {{ matrix.version }}"
  logs: "failed"  # Only collect logs for failed workers

#### 4. Kill Command (Pro)
```bash
testworkflow-toolkit kill <ref> [flags]
```
Terminates and cleans up service groups, with optional log collection before termination.

Flags:
- `--logs, -l`: Fetch logs for specific services using `name=expression` pairs

The kill command:
- Destroys all services in the specified group reference
- Optionally collects logs before termination based on conditions
- Supports conditional log collection using expressions
- Cleans up all Kubernetes resources (pods, jobs) for the group

Log collection expressions can access:
- `index`: The service instance index
- `count`: Total number of instances for that service
- Service state from the state machine

Example usage:
```bash
# Kill all services in group "test-services"
testworkflow-toolkit kill test-services

# Kill services and collect logs for failed instances
testworkflow-toolkit kill test-services --logs "db=index < 2" --logs "api=true"

# Conditional log collection
testworkflow-toolkit kill test-services --logs "web={{ failures > 0 }}"
```

## How Parallel Execution Works

1. **Spec parsing** - Parses the parallel configuration (matrix, shards, count)
2. **Worker generation** - Creates individual worker specifications
3. **Resource allocation** - Spawns Kubernetes jobs for each worker
4. **Execution monitoring** - Tracks worker status and collects outputs
5. **Synchronization** - Coordinates paused workers for synchronized resume
6. **Cleanup** - Collects logs and removes Kubernetes resources

### Key Interfaces

The toolkit uses interface-based design for testability:
- `WorkerRegistry` - Tracks worker lifecycle
- `ExecutionWorker` - Kubernetes job management
- `ArtifactStorage` - Output persistence

## Common Patterns

### Expression Resolution
```go
// Preserve environment expressions for worker context
machine := spawn.CreateBaseMachineWithoutEnv()

// Resolve in worker
{{env.DATABASE_URL}} → resolved with worker's environment
```

### Base64 Argument Encoding Pattern

testworkflow-init resolves ALL expressions in command arguments before execution. This fails for expressions that need context only available in the toolkit command (like `{{ matrix.browser }}` or `{{ index + 1 }}`).

The toolkit uses base64 encoding for complex arguments to prevent premature expression resolution by testworkflow-init.

**Commands using this pattern**:
- `parallel` - for matrix/shard/index expressions
- `services` - for service matrix expressions  
- `execute` - for workflow index/count expressions

This is not a workaround but an architectural boundary between:
- **testworkflow-init**: Basic container lifecycle and expression resolution
- **toolkit commands**: Domain-specific logic with specialized expression contexts

## Integration with Init Process

The toolkit communicates with the init process through:

1. **State File**: Located at `/.tktw/state` by default (read-only access)
2. **Environment Variables**: Configuration passed via TK_CFG and TK_REF
3. **Process Coordination**: Synchronization between containers

### State File Structure

The state file uses short JSON keys to minimize size. Here's what each key means:

```json
{
  "a": [[]],                   // Actions groups (array of arrays)
  "C": {},                     // Internal configuration (from TK_CFG)
  "g": 0,                      // Current group index (which group is running)
  "c": "step-1",               // Current reference (TK_REF value)
  "s": "passed",               // Current status expression
  "o": {                       // Outputs (key-value pairs)
    "result": "\"success\"",
    "score": "42"
  },
  "S": {                      // Steps data (execution details)
    "step-1": {
      "_": "step-1",          // Step reference
      "s": "passed",          // Status
      "e": 0,                 // Exit code
      "S": "2024-01-01T00:00:00Z", // Start time
      "c": "passed",          // Condition
      "p": []                 // Parent step references
    }
  },
  "R": {                      // Resource configuration
    "requests": {"cpu": "100m", "memory": "128Mi"},
    "limits": {"cpu": "1000m", "memory": "1Gi"}
  },
  "G": [{...}]                // Signature configs (service definitions)
}
```

## Usage Examples

### Running Tests with Artifact Collection

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: example-test
spec:
  container:
    image: node:18
  steps:
  - name: run-tests
    run:
      shell: |
        npm test
        
        # Collect test results
        testworkflow-toolkit artifacts "test-results/*.xml"
```

### Managing Services (Pro)

Services are defined at the TestWorkflow spec level and are started automatically:

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: test-with-services
spec:
  services:
    postgres:
      image: postgres:15
      env:
      - name: POSTGRES_PASSWORD
        value: mysecret
      readinessProbe:
        tcpSocket:
          port: 5432
        periodSeconds: 1
    redis:
      image: redis:7
      readinessProbe:
        tcpSocket:
          port: 6379
  steps:
  - name: run-tests
    run:
      shell: |
        # Services are automatically available
        # Access them by service name (postgres, redis)
        psql -h postgres -U postgres -c "SELECT 1"
        redis-cli -h redis ping
```

Services with matrix parameters:
```yaml
services:
  browser:
    matrix:
      driver: [chrome, firefox, safari]
      version: ["latest", "beta"]
    image: selenium/standalone-{{ matrix.driver }}:{{ matrix.version }}
    readinessProbe:
      httpGet:
        path: /wd/hub/status
        port: 4444
```

### Cloning Private Repositories

```bash
# Clone with authentication
testworkflow-toolkit clone https://github.com/private/repo.git \
  --token $GITHUB_TOKEN \
  --branch main \
  --depth 1
```

## Error Handling

The toolkit uses exit codes to indicate different types of failures:

- **0**: Success
- **1**: General error
- **2**: Configuration error
- **15**: Internal error
- **16**: Input validation error

## Internal Paths

The toolkit uses these internal paths:

- `/.tktw/`: Root directory for TestWorkflow files
- `/.tktw/state`: State file for coordination
- `/.tktw/bin/`: Binary tools
- `/.tktw/transfer/`: Temporary transfer directory
- `/dev/termination-log`: Kubernetes termination log

These can be overridden with environment variables:
- `TESTKUBE_TW_INTERNAL_PATH`
- `TESTKUBE_TW_STATE_PATH`
- `TESTKUBE_TW_TERMINATION_LOG_PATH`