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

## Key Design Principles

1. **Ephemeral by design** - Runs in short-lived containers, not long-running services
2. **Expression-based configuration** - Uses template expressions (`{{index}}`, `{{env.VAR}}`) for dynamic configuration
3. **Two-stage resolution** - Structural expressions resolved at scheduling, environment expressions in workers
4. **Non-blocking operations** - Prevents deadlocks in distributed scenarios
5. **Graceful degradation** - Continues execution even if some workers fail

## Core Commands

### parallel
Orchestrates parallel test execution with support for:
- Matrix testing (test combinations like OS × Browser × Version)
- Sharding (distribute test files across workers)
- Simple replication (run N copies)

### transfer
Manages file sharing between workflow steps using an internal HTTP server.

### execute
Runs individual test commands with proper environment setup.

### artifacts
Handles test output collection and storage.

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

The toolkit uses base64 encoding for complex arguments to prevent premature expression resolution by testworkflow-init.

**Problem**: testworkflow-init resolves ALL expressions in command arguments before execution. This fails for expressions that need context only available in the toolkit command (like `{{ matrix.browser }}` or `{{ index + 1 }}`).

**Solution**: Base64 encode arguments containing expressions:

```go
// In processor (operations.go):
encoded := base64.StdEncoding.EncodeToString(jsonData)
stage.Container().SetArgs("--base64", encoded)

// In toolkit command:
if base64Encoded {
    decoded, _ := base64.StdEncoding.DecodeString(args[0])
    // Now expressions can be resolved with proper context
}
```

**Commands using this pattern**:
- `parallel` - for matrix/shard/index expressions
- `services` - for service matrix expressions  
- `execute` - for workflow index/count expressions

This is not a workaround but an architectural boundary between:
- **testworkflow-init**: Basic container lifecycle and expression resolution
- **toolkit commands**: Domain-specific logic with specialized expression contexts

### Non-blocking Updates
```go
// Prevent deadlocks during status updates
select {
case updates <- status:
    // sent successfully
default:
    // channel full, skip non-critical update
}
```

### Resource Cleanup
```go
defer func() {
    // Always cleanup, even on panic
    worker.Destroy()
    logs.Save()
}()
```
