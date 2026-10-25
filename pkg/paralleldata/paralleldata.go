// Package paralleldata exposes what the workers of a parallel step published to the
// expression language, so that a later step can aggregate per-worker results:
//
//	{{ parallel("shard", 0).outputs.duration }}
//
// A worker writes to its outputs directory as any step does. Its own init scans the
// directory and prints an outputs instruction, which the parallel step reads from the
// worker's status stream. When the workers end, the parallel step prints one hint per
// step, named after the step id, holding every worker's status and outputs.
//
// The hint is stored in the state of the pod running the parallel step, where the later
// steps read it. It is not saved on the execution record: a workflow that wants to publish
// an aggregate writes it to its own outputs directory.
//
// Sensitive values never leave the worker: its init replaces them with withheld markers
// before printing the instruction, so a consumer that refuses markers keeps refusing them.
package paralleldata

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	// ParallelFn is the expression function reading the data of a parallel worker.
	ParallelFn = "parallel"

	// InstructionPrefix starts the name of the hint holding the workers of a parallel step.
	// It is not "parallel", which is the name of the output reporting worker status to the UI.
	InstructionPrefix = "paralleldata."
)

// Worker is what a parallel worker published.
type Worker struct {
	// Index is the index of the worker, the `index` the worker itself reads.
	Index int64 `json:"index"`
	// Status is the final status of the worker, empty for a worker that did not start.
	Status testkube.TestWorkflowStatus `json:"status,omitempty"`
	// Outputs are the outputs of every step of the worker, where a later step wins.
	Outputs map[string]string `json:"outputs,omitempty"`
}

// AsMap is what the expression sees for a worker.
func (w Worker) AsMap() map[string]interface{} {
	outputs := make(map[string]interface{}, len(w.Outputs))
	for key, value := range w.Outputs {
		outputs[key] = value
	}
	return map[string]interface{}{
		"index":   w.Index,
		"status":  string(w.Status),
		"outputs": outputs,
	}
}

// Group holds every worker of one parallel step.
type Group struct {
	// Count is the number of workers the step expected, so an index past the workers
	// that started still reads as a worker rather than an unknown one.
	Count   int64    `json:"count"`
	Workers []Worker `json:"workers"`
}

// Worker returns the worker at the index. A worker the step expected that has no entry,
// for example one that fail-fast never started, resolves with no status and no outputs.
func (g Group) Worker(index int64) (Worker, bool) {
	if index < 0 || index >= g.Count {
		return Worker{}, false
	}
	for _, worker := range g.Workers {
		if worker.Index == index {
			return worker, true
		}
	}
	return Worker{Index: index}, true
}

// InstructionName is the name of the hint holding the workers of the step with the id.
func InstructionName(stepId string) string {
	return InstructionPrefix + stepId
}

// NewMachine registers the parallel() function over the groups, keyed by the id of the
// parallel step.
func NewMachine(groups map[string]Group) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction(ParallelFn, func(values ...expressions.StaticValue) (interface{}, bool, error) {
			ref, index, err := parseArgs(values)
			if err != nil {
				return nil, true, err
			}
			group, ok := groups[ref]
			if !ok {
				return nil, true, unknownRefError(ref, groups)
			}
			worker, ok := group.Worker(index)
			if !ok {
				return nil, true, fmt.Errorf("%q function: parallel step %q has %d workers, there is no worker %d", ParallelFn, ref, group.Count, index)
			}
			return worker.AsMap(), true, nil
		})
}

// parseArgs reads the (ref, index) argument pair. The index is required: the function
// addresses one worker, and none of them is a natural default.
func parseArgs(values []expressions.StaticValue) (string, int64, error) {
	if len(values) != 2 {
		return "", 0, fmt.Errorf("%q function expects 2 arguments, %d provided", ParallelFn, len(values))
	}
	if !values[0].IsString() {
		return "", 0, fmt.Errorf("%q function expects the reference to be a string, %s provided", ParallelFn, values[0].String())
	}
	ref, _ := values[0].StringValue()
	if ref == "" {
		return "", 0, fmt.Errorf("%q function expects a non-empty reference", ParallelFn)
	}
	index, err := values[1].IntValue()
	if err != nil {
		return "", 0, fmt.Errorf("%q function expects the index to be a number, %s provided", ParallelFn, values[1].String())
	}
	return ref, index, nil
}

func unknownRefError(ref string, groups map[string]Group) error {
	if len(groups) == 0 {
		return fmt.Errorf("%q function: unknown parallel step %q, no parallel step has finished yet", ParallelFn, ref)
	}
	refs := make([]string, 0, len(groups))
	for key := range groups {
		refs = append(refs, key)
	}
	sort.Strings(refs)
	return fmt.Errorf("%q function: unknown parallel step %q, available parallel steps are %s", ParallelFn, ref, strings.Join(refs, ", "))
}
