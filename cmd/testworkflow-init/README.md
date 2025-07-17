# TestWorkflow Init Process

The TestWorkflow Init process is the orchestrator that runs as an init container in Kubernetes pods to set up and coordinate test execution across multiple containers.

## Overview

The init process is responsible for:
1. Setting up the test environment
2. Preparing the state file
3. Copying necessary binaries
4. Coordinating multi-container test execution
5. Managing the overall workflow lifecycle

## How It Works

### Understanding Groups

TestWorkflow organizes actions into **groups**. A group is a collection of actions that run together in the same container. Each group has an **index** (starting from 0):

- **Group 0**: Always the setup phase (runs in init container)
- **Group 1+**: Test phases (run in separate containers)

Think of groups as "stages" of your test execution.

### Startup Sequence

1. **Parse Arguments**: Init expects exactly one argument - the group index
2. **Initialize Orchestration**: Sets up the execution environment
3. **Run Actions**: Executes all actions in the specified group
4. **Save State**: Persists execution state for subsequent containers

### Execution Flow

```
testworkflow-init 0          # Group 0: Setup phase (init container)
    ↓
testworkflow-init 1          # Group 1: First test phase
    ↓  
testworkflow-init 2          # Group 2: Second test phase
    ↓
... (more groups as needed)
```

## Environment Variables

### Path Configuration

- **`TESTKUBE_TW_INTERNAL_PATH`**: Root directory for internal files (default: `/.tktw`)
- **`TESTKUBE_TW_STATE_PATH`**: Path to state file (default: `/.tktw/state`)
- **`TESTKUBE_TW_TERMINATION_LOG_PATH`**: Kubernetes termination log (default: `/dev/termination-log`)
- **`TESTKUBE_TW_INIT_BINARY_PATH`**: Path to init binary to copy
- **`TESTKUBE_TW_TOOLKIT_BINARY_PATH`**: Path to toolkit binary to copy
- **`TESTKUBE_TW_BUSYBOX_BINARY_PATH`**: Path to busybox utilities

### Binary Path Configuration

These are used during setup to locate binaries to copy (mainly for testing):

- **`TESTKUBE_TW_INIT_BINARY_PATH`**: Path to init binary (default: `/init`)
- **`TESTKUBE_TW_TOOLKIT_BINARY_PATH`**: Path to toolkit binary (default: `/toolkit`)
- **`TESTKUBE_TW_BUSYBOX_BINARY_PATH`**: Path to busybox utilities (default: `/.tktw-bin`)

### Image Information

- **`TESTKUBE_TW_INIT_IMAGE`**: Init container image name (used in setup data)
- **`TESTKUBE_TW_TOOLKIT_IMAGE`**: Toolkit container image name (used in setup data)

### Debug Configuration

- **`DEBUG`**: Enable debug logging (set to "1")

### Internal Environment Variables

These are used internally by the init process:

- **`TKI_N`**: Node name (from Kubernetes downward API)
- **`TKI_P`**: Pod name (from Kubernetes downward API)  
- **`TKI_S`**: Namespace name (from Kubernetes downward API)
- **`TKI_A`**: Service account name (from Kubernetes downward API)

### Execution Configuration

- **`TK_CFG`**: JSON-encoded configuration passed to all containers
  - Contains the complete workflow and execution configuration
  - This is stored in the state file at path `.C`
  - Set by the init process when executing actions

- **`TK_REF`**: Current step reference 
  - Set by the init process when running a step
  - Example: "rw7ckazs"
  - Used by toolkit to identify which step is running

## State File

The state file (default: `/.tktw/state`) is the central coordination mechanism between containers.

### State File Structure

The state file uses abbreviated keys to save space. Here's what each field means:

```json
{
  "a": [                       // Actions groups array
    [                          // Group 0 (always setup/init)
      {
        "name": "setup",
        "type": "setup"
      }
    ],
    [                          // Group 1 (first test container)
      {
        "name": "run-tests",
        "type": "execute",
        "command": ["npm", "test"]
      }
    ]
  ],
  "C": {                       // Configuration (from TK_CFG)
    "workflow": {              // Workflow metadata
      "name": "my-tests"       
    },
    "execution": {             // Execution details
      "id": "abc123",          
      "name": "my-tests-5",    
      "number": 5,             
      "scheduledAt": "...",    
      "debug": false,          
      "disableWebhooks": false 
    },
    "worker": {
      "namespace": "testkube"  
    }
  },
  "g": 0,                      // Current group index being executed
  "c": "step-1",               // Current reference (set when TK_REF is set)
  "s": "passed",               // Current status expression
  "o": {                       // Outputs (set by toolkit)
    "result": "\"success\"",   // JSON-encoded values
    "metrics": "{\"cpu\":0.5}"
  },
  "S": {                       // Steps execution data
    "step-1": {
      "_": "step-1",           // Step reference (ref)
      "c": "passed",           // Condition - when to run this step
      "s": "passed",           // Status - execution result
      "e": 0,                  // Exit code
      "p": [],                 // Parent step references
      "S": "2024-01-01T00:00:00Z", // Start time
      "t": "30s",              // Timeout duration (optional)
      "P": false,              // Paused on start
      "r": {                   // Retry policy (optional)
        "count": 3,
        "until": "passed"
      },
      "R": "result-value",     // Result value (optional)
      "i": 0                   // Iteration number
    }
  },
  "R": {                      // Container resources configuration
    "requests": {
      "cpu": "100m",          
      "memory": "128Mi"        
    },
    "limits": {
      "cpu": "1000m",         
      "memory": "1Gi"          
    }
  },
  "G": [...]                  // Signature configs (for services)
}
```

### State Management

The state is:
- Loaded at startup from the state file
- Updated during execution
- Saved after each action completes
- Shared between all containers in the pod

## Action Types

Actions are operations that the init process can perform. They are organized into groups (arrays), and each container executes one group.

The init process executes different types of actions:

### 1. Setup Action
```json
{
  "type": "setup",
  "name": "initialize"
}
```
Prepares the environment, copies binaries, sets up directories.

### 2. Execute Action
```json
{
  "type": "execute",
  "name": "run-test",
  "command": ["sh", "-c", "npm test"],
  "workingDir": "/workspace"
}
```
Runs commands in the container.

### 3. Start Action
```json
{
  "type": "start",
  "ref": "step-1"
}
```
Marks the start of a step execution.

### 4. End Action
```json
{
  "type": "end",
  "ref": "step-1"
}
```
Marks the end of a step execution.

### 5. Retry Action
```json
{
  "type": "retry",
  "ref": "step-1",
  "count": 3,
  "until": "passed"
}
```
Configures retry policy for a step.

### 6. Declare Action
```json
{
  "type": "declare",
  "ref": "step-1",
  "condition": "passed",
  "parents": ["parent-step"]
}
```
Declares step dependencies and conditions.

### 7. Pause Action
```json
{
  "type": "pause",
  "ref": "step-1"
}
```
Marks a step to pause on start.

### 8. Result Action
```json
{
  "type": "result",
  "ref": "step-1",
  "value": "some-result"
}
```
Sets a result value for a step.

### 9. Timeout Action
```json
{
  "type": "timeout",
  "ref": "step-1",
  "timeout": "30s"
}
```
Sets timeout for a step.

### 10. Container Transition Action
```json
{
  "type": "container",
  "config": {
    "command": ["/bin/sh"],
    "args": ["-c", "echo hello"]
  }
}
```
Transitions to a new container configuration.

### 11. Current Status Action
```json
{
  "type": "status",
  "status": "passed"
}
```
Sets the current execution status expression.

## Exit Codes

The init process uses specific exit codes:

- **0**: Success
- **137**: Aborted (SIGKILL/timeout) - from constants.CodeAborted
- **155**: Input validation error - from constants.CodeInputError
- **190**: Internal error - from constants.CodeInternal

## Directory Structure

The init process creates this directory structure:

```
/.tktw/
├── state              # State file (JSON)
├── bin/               # Binary tools directory  
│   └── sh            # Shell from busybox
├── init              # Init binary copy
└── toolkit           # Toolkit binary copy
```

## Orchestration Process

### Phase 1: Init Container (Always Group 0)

The init container always runs group index 0, which contains setup actions:

1. Create directory structure (/.tktw/)
2. Copy binaries (init, toolkit, shell)
3. Initialize state file with empty structure
4. Execute any setup actions defined in group 0
5. Save state for next containers

### Phase 2+: Test Containers (Groups 1, 2, 3...)

All steps are run as init containers except the last one, which runs as a regular container.

Each test container runs a specific group index (1 or higher):

1. Load existing state from previous containers
2. Execute all actions in its assigned group
3. Update step status (passed/failed)
4. Save any outputs produced
5. Update state file for next containers

## Integration with Toolkit

The init process and toolkit work together:

1. **Init** creates the environment and state
2. **Toolkit** reads state and updates outputs
3. **Init** coordinates between containers
4. **Toolkit** provides utilities for tests

### State Persistence

State is saved:
- After each action completes
- Before exit (success or failure)
- On panic recovery
