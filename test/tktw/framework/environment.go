package framework

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const (
	// DefaultStepRef is the default step reference ID used in test workflows
	DefaultStepRef = "r6lxv49"

	// defaultExecutionID is the test execution ID
	defaultExecutionID = "6877e9b024044bf1cf0d3f54"

	// defaultPodName is the test pod name
	defaultPodName = "6877e9b024044bf1cf0d3f54-zhzhf"
)

// WorkflowOptions allows customization of the test workflow
type WorkflowOptions struct {
	// Custom workflow actions JSON (if not provided, uses default)
	Actions string
	// Custom workflow signature JSON (if not provided, uses default)
	Signature string
	// Step reference name (default: r6lxv49)
	StepRef string
	// Resource limits and requests
	Resources struct {
		Requests struct {
			CPU    string // default: 100m
			Memory string // default: 128Mi
		}
		Limits struct {
			CPU    string // default: 200m
			Memory string // default: 256Mi
		}
	}
}

// testWorkflowActions defines the workflow actions for testing.
// Group 0: Setup actions (initialization, definitions, references)
// Group 1: Execution actions (shell command that tests resource usage)
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
	return SetupTestEnvironmentWithOptions(WorkflowOptions{})
}

// SetupTestEnvironmentWithOptions sets up environment variables with custom options
func SetupTestEnvironmentWithOptions(opts WorkflowOptions) func() {
	config := &testworkflowconfig.InternalConfig{
		Execution: testworkflowconfig.ExecutionConfig{
			Id:      defaultExecutionID,
			GroupId: defaultExecutionID,
			Name:    "alpine-memory-load-1",
		},
		Workflow: testworkflowconfig.WorkflowConfig{
			Name: "alpine-memory-load",
		},
		Resource: testworkflowconfig.ResourceConfig{
			Id: defaultExecutionID,
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
		"_00_TKI_P": defaultPodName,
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
		"TK_REF": DefaultStepRef,
		"TK_IP":  "127.0.0.1",
		"TK_CFG": string(jsonData),
	}

	// Apply custom options
	if opts.Actions != "" {
		envVars["_01_TKI_I"] = opts.Actions
	}
	if opts.Signature != "" {
		envVars["_03_TKI_G"] = opts.Signature
	}
	if opts.StepRef != "" {
		envVars["TK_REF"] = opts.StepRef
	}
	if opts.Resources.Requests.CPU != "" {
		envVars["_04_TKI_R_R_C"] = opts.Resources.Requests.CPU
	}
	if opts.Resources.Requests.Memory != "" {
		envVars["_04_TKI_R_R_M"] = opts.Resources.Requests.Memory
	}
	if opts.Resources.Limits.CPU != "" {
		envVars["_04_TKI_R_L_C"] = opts.Resources.Limits.CPU
	}
	if opts.Resources.Limits.Memory != "" {
		envVars["_04_TKI_R_L_M"] = opts.Resources.Limits.Memory
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

// ActionBuilder helps create workflow actions for testing
type ActionBuilder struct {
	Groups [][]map[string]interface{}
}

// NewActionBuilder creates a new action builder
func NewActionBuilder() *ActionBuilder {
	return &ActionBuilder{
		Groups: make([][]map[string]interface{}, 0),
	}
}

// AddInitGroup adds the standard initialization group (group 0)
func (b *ActionBuilder) AddInitGroup(stepRef string) *ActionBuilder {
	group := []map[string]interface{}{
		{"_": map[string]interface{}{"i": true, "b": true}},
		{"d": map[string]interface{}{"c": "true", "r": "root"}},
		{"d": map[string]interface{}{"c": "true", "r": stepRef, "p": []string{"root"}}},
		{"r": map[string]interface{}{"r": "root", "v": "true"}},
		{"r": map[string]interface{}{"r": stepRef, "v": "true"}},
		{"S": ""},
		{"s": "true"},
		{"S": "root"},
		{"s": "root"},
	}
	b.Groups = append(b.Groups, group)
	return b
}

// AddExecutionGroup adds a custom execution group
func (b *ActionBuilder) AddExecutionGroup(stepRef string, command []string, args []string) *ActionBuilder {
	group := []map[string]interface{}{
		{
			"c": map[string]interface{}{
				"r": stepRef,
				"c": map[string]interface{}{
					"command": command,
					"args":    args,
				},
			},
		},
		{"S": stepRef},
		{"e": map[string]interface{}{"r": stepRef, "v": "true"}},
		{"E": stepRef},
		{"E": "root"},
		{"E": ""},
	}
	b.Groups = append(b.Groups, group)
	return b
}

// Build returns the JSON string of the actions
func (b *ActionBuilder) Build() (string, error) {
	data, err := json.Marshal(b.Groups)
	if err != nil {
		return "", fmt.Errorf("failed to marshal actions: %v", err)
	}
	return string(data), nil
}

// CreateSimpleTestActions creates actions for a simple test that runs a command
func CreateSimpleTestActions(stepRef string, duration int, message string) (string, error) {
	command := fmt.Sprintf("echo '%s' && sleep %d && echo 'Test completed'", message, duration)

	return NewActionBuilder().
		AddInitGroup(stepRef).
		AddExecutionGroup(stepRef, []string{"/bin/sh"}, []string{"-c", command}).
		Build()
}
