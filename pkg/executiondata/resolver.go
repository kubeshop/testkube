package executiondata

import (
	"context"
	"fmt"
)

// Resolver turns a reference into the execution it addresses.
//
// It looks in the registry first, so the common case - an execution this workflow
// scheduled itself - costs no network call and keeps the data only the registry holds,
// such as the entry's alias and its position within a fan-out. Anything the registry does
// not know is asked of the control plane, which is what serves a raw execution id handed
// down as configuration, and the reserved "parent" reference.
type Resolver struct {
	// Registry holds the executions the current workflow scheduled. May be nil.
	Registry *Registry
	// Repository resolves executions the registry does not know. May be nil, in which
	// case only registered executions can be referenced.
	Repository ExecutionRepository
	// ParentIds is the chain of executions that led to this one, oldest first.
	ParentIds []string
}

// Resolve finds the execution a reference addresses, or explains why it cannot.
func (r Resolver) Resolve(ctx context.Context, ref string, index int64) (Execution, error) {
	if r.Registry != nil {
		execution, ok, err := r.Registry.Lookup(ref, index)
		if err != nil {
			return Execution{}, err
		}
		if ok {
			return execution, nil
		}
	}

	id := ref
	if ref == ParentRef {
		if len(r.ParentIds) == 0 {
			return Execution{}, fmt.Errorf("cannot resolve execution(%q): this execution has no parent", ParentRef)
		}
		id = r.ParentIds[len(r.ParentIds)-1]
	}

	// Fan-out indexes only exist within the local registry - anything resolved
	// through the control plane is a single execution addressed by its id.
	if index != 0 {
		return Execution{}, UnknownRefError(ref, index, r.knownRefs())
	}

	if r.Repository == nil {
		return Execution{}, UnknownRefError(ref, index, r.knownRefs())
	}

	execution, err := r.Repository.Get(ctx, id)
	if err != nil {
		if ref == ParentRef {
			return Execution{}, fmt.Errorf("reading parent execution %s: %w", id, err)
		}
		return Execution{}, fmt.Errorf("reading execution %q: %w", ref, err)
	}
	if execution.Id == "" {
		return Execution{}, UnknownRefError(ref, index, r.knownRefs())
	}
	if ref == ParentRef {
		execution.Alias = ParentRef
	}
	return execution, nil
}

func (r Resolver) knownRefs() []string {
	if r.Registry == nil {
		return nil
	}
	return r.Registry.Refs()
}
