package data

import (
	"encoding/json"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/paralleldata"
)

// ParallelGroups rebuilds what the workers of the finished parallel steps published,
// keyed by the step id. Each group was recorded as a hint by the `parallel` step, so it
// survives across steps and across processes.
func ParallelGroups() map[string]paralleldata.Group {
	groups := make(map[string]paralleldata.Group)
	for stepId, raw := range GetState().GetOutputsWithPrefix(paralleldata.InstructionPrefix) {
		var group paralleldata.Group
		if err := json.Unmarshal([]byte(raw), &group); err != nil {
			output.Std.Warnf("warn: could not read the workers of parallel step %q: %s\n", stepId, err.Error())
			continue
		}
		groups[stepId] = group
	}
	return groups
}

// ParallelDataMachine resolves parallel(), the data published by the workers of the
// parallel steps that finished before.
func ParallelDataMachine() expressions.Machine {
	return paralleldata.NewMachine(ParallelGroups())
}
