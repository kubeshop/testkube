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

func (r *Registry) Lookup(ref string, index int64) (Execution, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, execution := range r.executions {
		if execution.Index != index {
			continue
		}
		for _, candidate := range execution.Refs() {
			if candidate == ref {
				return execution, true
			}
		}
	}
	return Execution{}, false
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

// UnknownRefError explains that a reference could not be resolved, listing what could.
func UnknownRefError(ref string, index int64, known []string) error {
	position := ""
	if index != 0 {
		position = fmt.Sprintf(" at index %d", index)
	}
	if len(known) == 0 {
		return fmt.Errorf("unknown execution %q%s: this workflow has not executed any test workflow yet - run it in an earlier step", ref, position)
	}
	return fmt.Errorf("unknown execution %q%s: available executions are %s", ref, position, strings.Join(known, ", "))
}
