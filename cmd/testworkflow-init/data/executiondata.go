package data

import (
	"encoding/json"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// ExecutionRegistry rebuilds the registry of test workflows this workflow has
// executed. Each entry was recorded as an output instruction by the `execute`
// step, so it survives across steps and across processes.
func ExecutionRegistry() *executiondata.Registry {
	registry := executiondata.NewRegistry()
	for _, raw := range GetState().GetOutputsWithPrefix(executiondata.ExecutionInstructionPrefix) {
		var entries []executiondata.Execution
		if err := json.Unmarshal([]byte(raw), &entries); err != nil {
			output.Std.Warnf("warn: could not read executed test workflow: %s\n", err.Error())
			continue
		}
		for _, entry := range entries {
			registry.Add(entry)
		}
	}
	return registry
}

// ExecutionDataMachine resolves data produced by other executions - the ones this
// workflow scheduled, and the one that scheduled this workflow.
func ExecutionDataMachine() expressions.Machine {
	return ExecutionDataMachineFor(ExecutionRegistry())
}

// ExecutionDataMachineFor is ExecutionDataMachine over a caller-owned registry.
// The `execute` step uses it to keep resolving against a registry it keeps growing
// as it schedules more test workflows.
func ExecutionDataMachineFor(registry *executiondata.Registry) expressions.Machine {
	return executiondata.NewMachine(executiondata.MachineOptions{
		Registry:   registry,
		Repository: ExecutionDataRepository(),
		ParentIds:  ParentExecutionIds(),
	})
}

// ExecutionDataRepository reads executions and their artifacts from the control plane.
func ExecutionDataRepository() executiondata.ExecutionRepository {
	cfg := GetState().InternalConfig.Execution
	return executiondata.NewExecutionRepository(CloudClient, cfg.EnvironmentId)
}

// ParentExecutionIds returns the chain of executions that led to this one, oldest
// first. The internal config stores it as a slash-joined string.
func ParentExecutionIds() []string {
	raw := GetState().InternalConfig.Execution.ParentIds
	if raw == "" {
		return nil
	}
	ids := make([]string, 0)
	for _, id := range strings.Split(raw, "/") {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
