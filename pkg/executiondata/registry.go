package executiondata

import (
	"fmt"
	"strings"
	"sync"
)

// Registry holds the executions a workflow scheduled, so they can be addressed
// by alias or workflow name instead of by opaque execution id.
type Registry struct {
	mu         sync.RWMutex
	executions []Execution
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Add registers an execution, replacing an earlier entry for the same group and index.
func (r *Registry) Add(execution Execution) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.executions {
		if r.executions[i].Key() == execution.Key() && r.executions[i].Index == execution.Index {
			r.executions[i] = execution
			return
		}
	}
	r.executions = append(r.executions, execution)
}

// Lookup finds the execution a reference addresses at a position.
//
// A reference may be a group key, a workflow name or an execution id, and an aliased
// entry stays addressable by the workflow it ran - so more than one execution can answer
// to the same reference. An aliased selector covering workflow "a" and a separate entry
// running "a" without an alias both answer to execution("a", 0). Returning either would
// hand the caller data from an arbitrary child, so that is reported as an error instead
// of resolved by insertion order.
//
// An execution id is matched before anything else and without regard to the index. An id
// names one execution outright, whereas an alias or a workflow name names a group whose
// members are told apart by index - so filtering an id by index would hide every member of
// a fan-out but the first. A caller passing an id together with an index therefore gets the
// execution the id names, and the index is not consulted; the index cannot contradict it,
// because an unspecified index and an explicit 0 are the same argument here.
func (r *Registry) Lookup(ref string, index int64) (Execution, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ids come from the control plane and each execution is registered under a single
	// key, so two entries cannot share one - this reports a collision rather than
	// picking, to keep that an invariant instead of an assumption.
	var byId Execution
	foundById := false
	for _, execution := range r.executions {
		if execution.Id == "" || execution.Id != ref {
			continue
		}
		if foundById {
			return Execution{}, false, AmbiguousRefError(ref, index, byId, execution)
		}
		byId, foundById = execution, true
	}
	if foundById {
		return byId, true, nil
	}

	var found Execution
	matched := false
	for _, execution := range r.executions {
		if execution.Index != index {
			continue
		}
		for _, candidate := range execution.Refs() {
			if candidate != ref {
				continue
			}
			if matched {
				return Execution{}, false, AmbiguousRefError(ref, index, found, execution)
			}
			found, matched = execution, true
			break
		}
	}
	return found, matched, nil
}

func (r *Registry) Refs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{}, len(r.executions))
	refs := make([]string, 0, len(r.executions))
	for _, execution := range r.executions {
		key := execution.Key()
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, key)
	}
	return refs
}

// Reset drops every execution registered under a primary reference.
func (r *Registry) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := r.executions[:0]
	for _, execution := range r.executions {
		if execution.Key() != key {
			kept = append(kept, execution)
		}
	}
	r.executions = kept
}

// Group returns every execution sharing a primary reference, ordered by index.
// This is the unit the parent publishes as a single output instruction.
func (r *Registry) Group(key string) []Execution {
	r.mu.RLock()
	defer r.mu.RUnlock()

	group := make([]Execution, 0, 1)
	for _, execution := range r.executions {
		if execution.Key() == key {
			group = append(group, execution)
		}
	}
	return group
}

// ExecutionInstructionName is the output instruction name carrying a single group.
func ExecutionInstructionName(key string) string {
	return ExecutionInstructionPrefix + instructionSafe(key)
}

// instructionSafe strips characters the instruction name grammar does not accept.
// Aliases and workflow names are Kubernetes names, so this is a safety net rather
// than an expected transformation.
func instructionSafe(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
}

// AmbiguousRefError explains that a reference addresses more than one execution, and how
// to say which one was meant.
func AmbiguousRefError(ref string, index int64, first, second Execution) error {
	position := ""
	if index != 0 {
		position = fmt.Sprintf(" at index %d", index)
	}
	return fmt.Errorf("ambiguous execution %q%s: it addresses both %s and %s - address them by their own 'as' alias, or set a unique 'as' on one of them",
		ref, position, describeExecution(first), describeExecution(second))
}

// describeExecution names an execution the way its author would recognise it.
func describeExecution(e Execution) string {
	workflow := e.Workflow
	if workflow == "" {
		workflow = "unnamed workflow"
	}
	if e.Alias != "" {
		return fmt.Sprintf("the %q run aliased %q", workflow, e.Alias)
	}
	return fmt.Sprintf("the unaliased %q run", workflow)
}

// UnknownRefError explains that a reference could not be resolved, listing what could.
func UnknownRefError(ref string, index int64, known []string) error {
	position := ""
	if index != 0 {
		position = fmt.Sprintf(" at index %d", index)
	}
	// A workflow only knows the test workflows it ran itself. Anything else - a
	// sibling of the same suite, for instance - has to be addressed by execution id,
	// which the parent can pass down as configuration.
	if len(known) == 0 {
		return fmt.Errorf("unknown execution %q%s: this workflow has not executed any test workflow yet - run it in an earlier step, or pass an execution id", ref, position)
	}
	return fmt.Errorf("unknown execution %q%s: available executions are %s (anything else must be addressed by execution id)", ref, position, strings.Join(known, ", "))
}
