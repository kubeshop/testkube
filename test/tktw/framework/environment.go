package framework

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// testWorkflowActions defines the workflow actions for testing.
// This JSON structure represents two groups of actions:
// - Group 0: Setup actions (initialization, definitions, references)
// - Group 1: Execution actions (shell command that tests resource usage)
//
// Action types:
// - "_": Initialize (i=interrupts, b=buffered)
// - "d": Define step (c=condition, r=ref, p=parents)
// - "r": Set result (r=ref, v=value)
// - "S": Start step (value=ref or empty for global)
// - "s": Set status (value=status or ref)
// - "c": Execute container (r=ref, c=container spec)
// - "e": Execute step (r=ref, v=value)
// - "E": End step (value=ref or empty for global)
var testWorkflowActions = `[
  [
    {
      "_": {
        "i": true,
        "b": true
      }
    },
    {
      "d": {
        "c": "true",
        "r": "root"
      }
    },
    {
      "d": {
        "c": "true",
        "r": "r6lxv49",
        "p": ["root"]
      }
    },
    {
      "r": {
        "r": "root",
        "v": "true"
      }
    },
    {
      "r": {
        "r": "r6lxv49",
        "v": "true"
      }
    },
    {
      "S": ""
    },
    {
      "s": "true"
    },
    {
      "S": "root"
    },
    {
      "s": "root"
    }
  ],
  [
    {
      "c": {
        "r": "r6lxv49",
        "c": {
          "command": ["/bin/sh"],
          "args": [
            "-c",
            "echo 'generating ~10MB random data in memory...' && random_data=\"$(head -c 10485760 /dev/urandom | base64)\" && echo \"data generated, length: ${#random_data} characters\" && echo 'processing data (CPU work)...' && echo \"$random_data\" | tr -d '[:alnum:]' | wc -c && echo 'sleeping for 5s...' && sleep 5 && echo 'done.' && echo 'TestWorkflow execution successful' > /tmp/test_execution.txt"
          ]
        }
      }
    },
    {
      "S": "r6lxv49"
    },
    {
      "e": {
        "r": "r6lxv49",
        "v": "true"
      }
    },
    {
      "E": "r6lxv49"
    },
    {
      "E": "root"
    },
    {
      "E": ""
    }
  ]
]`

// testWorkflowSignature describes the workflow steps.
// Each entry defines a step with its reference, name, and category.
var testWorkflowSignature = `[
  {
    "ref": "r6lxv49",
    "name": "memory-cpu-test",
    "category": "Run shell command"
  }
]`

// SetupTestEnvironment sets up environment variables for e2e testing.
func SetupTestEnvironment() func() {
	config := &testworkflowconfig.InternalConfig{
		Execution: testworkflowconfig.ExecutionConfig{
			Id:      "6877e9b024044bf1cf0d3f54",
			GroupId: "6877e9b024044bf1cf0d3f54",
			Name:    "alpine-memory-load-1",
		},
		Workflow: testworkflowconfig.WorkflowConfig{
			Name: "alpine-memory-load",
		},
		Resource: testworkflowconfig.ResourceConfig{
			Id: "6877e9b024044bf1cf0d3f54",
		},
		Worker: testworkflowconfig.WorkerConfig{
			Namespace: "testkube",
		},
	}

	// Serialize the config
	jsonData, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal config: %v", err))
	}

	// Set required environment variables for the init process.
	// These simulate the environment variables that would be set by Kubernetes.
	envVars := map[string]string{
		// Control plane configuration (group 00)
		"_00_TKI_N": "testkube-control-plane",
		"_00_TKI_P": "6877e9b024044bf1cf0d3f54-zhzhf",
		"_00_TKI_S": "testkube",
		"_00_TKI_A": "testkube-api-server-tests-job",

		// Workflow actions specification (group 01)
		"_01_TKI_I": testWorkflowActions,

		// Internal configuration (group 03)
		"_03_TKI_C": string(jsonData),
		"_03_TKI_G": testWorkflowSignature,

		// Resource limits and requests (group 04)
		"_04_TKI_R_R_C": "100m",
		"_04_TKI_R_R_M": "128Mi",
		"_04_TKI_R_L_C": "200m",
		"_04_TKI_R_L_M": "256Mi",

		// Runtime information (group 05)
		"_05_TKI_O": "2",

		// Common variables
		"TK_REF": "r6lxv49",
		"TK_IP":  "127.0.0.1",
		"TK_CFG": string(jsonData),
	}

	// Save original values for restoration
	originals := make(map[string]string)
	for key := range envVars {
		if original, exists := os.LookupEnv(key); exists {
			originals[key] = original
		}
	}

	// Set the environment variables
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			panic(fmt.Sprintf("failed to set env var %s: %v", key, err))
		}
	}

	// Return cleanup function
	return func() {
		for key := range envVars {
			if original, exists := originals[key]; exists {
				if err := os.Setenv(key, original); err != nil {
					// Log error but don't panic in cleanup
					fmt.Printf("failed to restore env var %s: %v\n", key, err)
				}
			} else {
				if err := os.Unsetenv(key); err != nil {
					// Log error but don't panic in cleanup
					fmt.Printf("failed to unset env var %s: %v\n", key, err)
				}
			}
		}
	}
}

// SetupResourceTestEnvironment sets up environment for resource testing
func SetupResourceTestEnvironment() func() {
	cleanup := SetupTestEnvironment()

	// Override for resource testing
	if err := os.Setenv("TK_REF", "resource-test-step"); err != nil {
		panic(fmt.Sprintf("failed to set TK_REF: %v", err))
	}
	if err := os.Setenv("_04_TKI_R_R_C", "500m"); err != nil {
		panic(fmt.Sprintf("failed to set CPU request: %v", err))
	}
	if err := os.Setenv("_04_TKI_R_R_M", "512Mi"); err != nil {
		panic(fmt.Sprintf("failed to set memory request: %v", err))
	}

	return cleanup
}
