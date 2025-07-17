# TestWorkflow Toolkit

The TestWorkflow Toolkit is a helper binary that runs inside test containers to provide advanced functionality like artifact collection, service management, and test execution coordination.

## Overview

The toolkit is injected into every test container and provides commands that can be called during test execution. It communicates with the TestWorkflow Init process to coordinate test execution across multiple containers and steps.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 Test Pod                        │
├─────────────────────────────────────────────────┤
│  Init Container 0                               │
│  └── testworkflow-init (orchestrator)           │
├─────────────────────────────────────────────────┤
│  Init Container 1                               │
│  ├── testworkflow-toolkit                       │
│  └── Your test code                             │
├─────────────────────────────────────────────────┤
│  Test Container 1                               │
│  ├── testworkflow-toolkit                       │
│  └── Your test code                             │
└─────────────────────────────────────────────────┘
```

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

### 1. Artifacts Command
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

### 2. Clone Command
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

### 3. Tarball Command
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

### 4. Transfer Command
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

### 1. Execute Command (Pro)
```bash
testworkflow-toolkit execute <command> [args...]
```
Runs a command and captures its output, exit code, and execution metadata.

### 2. Services Command (Pro)
```bash
testworkflow-toolkit services <action> [service-name]
```
Manages test services (start, stop, check status).

Actions:
- `start <name>`: Start a service
- `stop <name>`: Stop a service  
- `status <name>`: Check service status

### 3. Parallel Command (Pro)
```bash
testworkflow-toolkit parallel [flags]
```
Executes multiple operations in parallel.

### 4. Kill Command (Pro)
```bash
testworkflow-toolkit kill [flags]
```
Terminates processes or services.

## Integration with Init Process

The toolkit communicates with the init process through:

1. **State File**: Located at `/.tktw/state` by default (read-only access)
2. **Environment Variables**: Configuration passed via TK_CFG and TK_REF
3. **Process Coordination**: Synchronization between containers

### State File Structure

The state file uses short JSON keys to minimize size. Here's what each key means:

```json
{
  "a": [[]],                    // Actions groups (array of arrays)
  "C": {},                      // Internal configuration (from TK_CFG)
  "g": 0,                       // Current group index (which group is running)
  "c": "step-1",               // Current reference (TK_REF value)
  "s": "passed",               // Current status expression
  "o": {                       // Outputs (key-value pairs)
    "result": "\"success\"",
    "score": "42"
  },
  "S": {                       // Steps data (execution details)
    "step-1": {
      "_": "step-1",          // Step reference
      "s": "passed",          // Status
      "e": 0,                 // Exit code
      "S": "2024-01-01T00:00:00Z", // Start time
      "c": "passed",          // Condition
      "p": []                 // Parent step references
    }
  },
  "R": {                       // Resource configuration
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

```yaml
steps:
- name: start-database
  run:
    shell: |
      # Start PostgreSQL service (Pro feature)
      testworkflow-toolkit services start postgres
      
      # Wait for service to be ready
      testworkflow-toolkit services wait postgres
      
- name: run-tests
  run:
    shell: |
      npm test
      
- name: cleanup
  run:
    shell: |
      # Stop service
      testworkflow-toolkit services stop postgres
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

## Debugging

Enable debug mode for verbose logging:

```bash
export DEBUG=1
testworkflow-toolkit execute echo "test"
```

This will show:
- Configuration loading
- Command execution details
- File operations
- Network requests

## Best Practices

1. **Always collect artifacts** for important test outputs
2. **Use meaningful step references** (TK_REF) for debugging
3. **Handle service lifecycle properly** - always stop services you start
4. **Check exit codes** when executing commands
5. **Use glob patterns** for flexible artifact collection

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