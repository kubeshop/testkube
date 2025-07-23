# TestWorkflow Init Process

The TestWorkflow Init process is a wrapper that orchestrates test execution in containers.
It runs in every container to provide consistent metrics collection, retry logic, timeout handling, and state management across all test workflow steps.

## Architecture

The init process runs as the entrypoint in EVERY container, first to set up the environment (Group 0) and then to wrap each test step (Groups 1+) with consistent behavior.

## Overview

The init process serves two distinct roles:

### 1. Group 0: Initial Setup (Always First)
The first execution (Group 0) always performs the initialization:
- **Copy binaries** - Copies `/init` and `/toolkit` binaries to the shared volume
- **Setup shell** - Configures busybox utilities for script execution
- **Initialize state** - Creates the state file for sharing data between steps
- **Prepare environment** - Sets up the execution environment for subsequent steps

### 2. Groups 1+: Test Step Execution
All subsequent groups (1, 2, 3...) wrap actual test steps:
- **Git operations** - Cloning repositories, checking out branches
- **Test execution** - Running unit tests, integration tests, e2e tests
- **Build steps** - Compiling code, building containers
- **Cleanup operations** - Teardown and cleanup
- **Any custom commands** - Any step defined in the TestWorkflow

For ALL groups, the init process provides:
- **Metrics collection** - CPU, memory, disk, and network usage monitoring
- **Retry logic** - Automatic retries with configurable policies
- **Timeout handling** - Graceful timeout with process cleanup
- **Output filtering** - Sensitive data obfuscation
- **State persistence** - Sharing data between steps
- **Pause/resume control** - Via HTTP control server

### High-Level Flow

```
Pod Creation
    ↓
Init Container (Group 0 - Setup)
    ├─ testworkflow-init 0
    ├─ Copy /init and /toolkit binaries
    ├─ Setup busybox shell utilities  
    └─ Initialize state file
    ↓
Container 1 (Group 1 - e.g., Git Clone)
    ├─ testworkflow-init 1
    ├─ Load state from Group 0
    ├─ Start metrics collection
    ├─ Execute git clone with retries
    └─ Save state and metrics
    ↓
Container 2 (Group 2 - e.g., Run Tests)
    ├─ testworkflow-init 2
    ├─ Load state from previous groups
    ├─ Start metrics collection
    ├─ Execute test command
    └─ Save state and metrics
    ↓
... (more steps as needed)
```

### Why "Init" Process for Everything?

- **Binary distribution**: Group 0 copies the init binary to a shared volume so all subsequent containers can use it
- **Consistency**: Every step gets identical retry logic, timeout handling, and metrics
- **Efficiency**: One binary provides all orchestration features
- **State Management**: Seamless state sharing between all containers via the state file

### Core Concepts

- **Actions**: Atomic operations (execute command, set timeout, declare dependencies)
- **Steps**: Logical groupings of actions with retry/timeout policies
- **Groups**: Collections of actions that run in the same container
- **State**: Shared data structure persisted to disk between containers

The runner processes actions sequentially within a group, managing:
- Retry logic with configurable conditions
- Timeout monitoring with graceful shutdown
- Pause/resume via control server on port 30107
- Sensitive data obfuscation in logs
- Metrics collection for resource usage

### Example Workflow

```yaml
# Example TestWorkflow with multiple steps

# Group 0 is automatically created - the testworkflow-init process sets up binaries
spec:
  content:
    # Group 1 - run git clone via the testworkflow-toolkit by the testworkflow-init wrapper
    git:
      uri: https://github.com/kubeshop/testkube.git
      mountPath: /custom/mount/path
  steps:
    # Group 2: Dependency installation wrapped by testworkflow-init
    - name: dependencies 
      run:
        image: golang:1.21
        command: ["go", "mod", "download"]
    # Group 3: Test execution wrapped by testworkflow-init
    - name: unit-tests
      run:
        image: golang:1.21
        command: ["go", "test", "./..."]
      retry:              # Init process handles retry logic
        count: 3
        until: passed
      timeout: 10m        # Init process enforces timeout
    # Group 4: Cleanup wrapped by testworkflow-init
    - name: cleanup      
      run:
        image: alpine
        command: ["rm", "-rf", "/tmp/test-artifacts"]
```

Note: Group 0 (setup) is automatically injected before your steps to copy binaries and initialize the environment.

## How It Works

### Understanding Groups

TestWorkflow organizes actions into **groups**. A group is a collection of actions that run together in the same container. Each group has an **index**:

- **Group 0**: ALWAYS the setup phase (automatically injected)
  - Runs the ActionTypeSetup action
  - Copies `/init` and `/toolkit` binaries to shared volume
  - Sets up busybox shell utilities
  - Initializes the state file
  - Runs as a Kubernetes init container

- **Group 1+**: Your actual test steps
  - Each step from your TestWorkflow becomes a group
  - Group 1 = First step in your workflow
  - Group 2 = Second step in your workflow
  - And so on...

Each group/step:
- Runs in its own container
- Uses testworkflow-init as its entrypoint
- Gets its own metrics collection
- Can have retry and timeout policies
- Shares state with other steps via the state file
- Has access to binaries copied by Group 0

### Startup Sequence

1. **Parse Arguments**: Init expects exactly one argument - the group index
2. **Initialize Orchestration**: Sets up the execution environment
3. **Run Actions**: Executes all actions in the specified group
4. **Save State**: Persists execution state for subsequent containers

## Environment Variables

| Variable                    | Purpose                                                  | Default        |
|-----------------------------|----------------------------------------------------------|----------------|
| `TESTKUBE_TW_INTERNAL_PATH` | Root directory for internal files                        | `/.tktw`       |
| `TESTKUBE_TW_STATE_PATH`    | Path to state file                                       | `/.tktw/state` |
| `DEBUG`                     | Enable debug logging                                     | -              |
| `TK_CFG`                    | JSON workflow configuration                              | -              |
| `TK_REF`                    | Current step reference                                   | -              |
| `TKI_N/P/S/A`               | Kubernetes metadata (node/pod/namespace/service account) | -              |

## State File

The state file (default: `/.tktw/state`) is the central coordination mechanism between containers.

### State File Structure

The state file uses abbreviated keys for efficiency:

| Key | Content       | Description                                |
|-----|---------------|--------------------------------------------|
| `a` | Action groups | Arrays of actions for each container       |
| `C` | Configuration | Workflow metadata from TK_CFG              |
| `S` | Steps data    | Execution status, results, timing per step |
| `o` | Outputs       | Results from toolkit operations            |
| `g` | Current group | Index of executing group                   |
| `c` | Current ref   | Active step reference                      |
| `s` | Status        | Current execution status                   |

State is loaded at startup, updated during execution, and saved after each action.

## Action Types

Actions are operations organized into groups, with each container executing one group:

| Type        | Purpose                                    | Key Fields                    |
|-------------|--------------------------------------------|-------------------------------|
| `setup`     | Environment initialization, binary copying | -                             |
| `execute`   | Run commands                               | `command`, `workingDir`       |
| `start/end` | Mark step boundaries                       | `ref`                         |
| `retry`     | Configure retry policy                     | `ref`, `count`, `until`       |
| `declare`   | Set dependencies/conditions                | `ref`, `condition`, `parents` |
| `timeout`   | Set step timeout                           | `ref`, `timeout`              |
| `pause`     | Mark step for pausing                      | `ref`                         |
| `result`    | Set step result value                      | `ref`, `value`                |
| `container` | Change container config                    | `config`                      |
| `status`    | Set current status                         | `status`                      |

## Exit Codes

The init process uses specific exit codes:

- **0**: Success
- **137**: Aborted (SIGKILL/timeout) - from constants.CodeAborted
- **155**: Input validation error - from constants.CodeInputError
- **190**: Internal error - from constants.CodeInternal

## Key Implementation Details

### Group 0 Setup
- Copies `/init` and `/toolkit` binaries to shared volume
- Sets up busybox utilities and shell environment
- Creates state file with 0777 permissions for container sharing

### Retry Logic
- Configurable count and conditions ("passed", "failed", etc.)
- Parent timeout prevents children from retrying
- State reset between attempts for accurate tracking

### Metrics & State
- Background metrics collection (CPU, memory, disk, network)
- State persisted after each action for crash recovery
- JSON format with abbreviated keys for efficiency

### State Persistence

State is saved:
- After each action completes
- Before exit (success or failure)
- On panic recovery

## Debugging

- **Debug mode**: `export DEBUG=1`
- **State inspection**: `cat /.tktw/state`
- **State persistence**: Saved after each action, before exit, on panic recovery

## Expression System

The init process uses Testkube's expression engine for dynamic configuration and conditional logic. Expressions are used throughout the workflow for conditions, retry logic, and variable interpolation.

### Expression Usage in Init Process

#### 1. Step Conditions
Determine when a step should execute:
```yaml
steps:
  - name: deploy
    condition: "env.ENVIRONMENT == 'production'"
  - name: cleanup
    condition: "always"  # Alias for 'true'
```

#### 2. Retry Conditions
Control when retries should stop:
```yaml
retry:
  count: 3
  until: "passed"  # Retry until the step passes
  # until: "error"  # Retry only on errors, not on failures
```

#### 3. Status Expressions
Evaluate workflow status:
```yaml
# Common status checks
"passed"              # Expands to: status == "passed"
"failed"              # Expands to: status != "passed" && status != "skipped"
"self.passed"         # Current step passed
"self.failed"         # Current step failed
"parent-step.passed"  # Named step passed
```

### Expression Machines

The init process uses several execution contexts:

| Machine           | Variables                                    | Purpose                 |
|-------------------|----------------------------------------------|-------------------------|
| LocalMachine      | `status`                                     | Current step status     |
| StateMachine      | `self.status`, `output.*`, `services.*`      | Workflow state access   |
| EnvMachine        | `env.*`                                      | Environment variables   |
| RefSuccessMachine | `step-name`                                  | Check if step succeeded |
| AliasMachine      | `always`→`true`, `passed`→`status=='passed'` | Common aliases          |

### Common Expression Patterns

```yaml
# Conditions
condition: "env.ENVIRONMENT == 'production'"
condition: "setup-step.failed || parent1.failed"

# Retry logic
retry:
  count: 3
  until: "passed"  # or "self.exitCode != 1"

# Output references
condition: "output.testResult == 'success'"
command: ["deploy", "--version", "{{output.version}}"]
```

### Expression Evaluation Context

Expressions are evaluated at different times with different contexts:

1. **Step Start**: Condition expressions determine if step should run
2. **Step End**: Result expressions determine final status
3. **After Each Attempt**: Retry conditions check if should retry
4. **During Execution**: Status expressions track current state

See the [Expression Engine Documentation](../../pkg/expressions/README.md).