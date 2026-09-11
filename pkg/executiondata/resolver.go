package executiondata

import (
	"context"
	"fmt"
)

// Resolver turns a reference into the execution it addresses.
//
// The reserved references are resolved first, so nothing a workflow executes can
// shadow them. Everything else is looked up in the registry, so the common case -
// an execution this workflow scheduled itself - costs no network call and keeps
// the data only the registry holds, such as the entry's alias and its position
// within a fan-out. Anything the registry does not know is asked of the control
// plane, which is what serves a raw execution id handed down as configuration.
type Resolver struct {
	// Registry holds the executions the current workflow scheduled. May be nil.
	Registry *Registry
	// Repository resolves executions the registry does not know. May be nil, in which
	// case only registered executions can be referenced.
	Repository ExecutionRepository
	// ParentIds is the chain of executions that led to this one, oldest first.
	ParentIds []string
	// RerunId is the execution this one is a rerun of, empty when it is not one.
	RerunId string
}

// Resolve finds the execution a reference addresses, or explains why it cannot.
func (r Resolver) Resolve(ctx context.Context, ref string, index int64) (Execution, error) {
	id := ref

	if IsReservedRef(ref) {
		// A reserved reference wins over the registry. Looking in the registry
		// first would let a workflow that executed a child aliased "parent" or
		// "rerun" - or simply ran a workflow of that name - shadow the reserved
		// meaning, and the step would read a different execution than it asked
		// for with nothing to say so.
		//
		// The collision is reported rather than resolved either way: silently
		// preferring the reserved meaning would instead make that child
		// unreachable by name, which is the same failure pointing the other way.
		if r.Registry != nil {
			if shadow, ok, err := r.Registry.Lookup(ref, index); err == nil && ok {
				return Execution{}, ShadowedReservedRefError(ref, shadow)
			}
		}

		var err error
		if id, err = r.reservedId(ref); err != nil {
			return Execution{}, err
		}
	} else if r.Registry != nil {
		execution, ok, err := r.Registry.Lookup(ref, index)
		if err != nil {
			return Execution{}, err
		}
		if ok {
			return execution, nil
		}
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
	if IsReservedRef(ref) {
		execution.Alias = ref
	}
	return execution, nil
}

// reservedId is the execution a reserved reference points at, or an explanation
// of why this execution has none.
func (r Resolver) reservedId(ref string) (string, error) {
	switch ref {
	case ParentRef:
		if len(r.ParentIds) == 0 {
			return "", fmt.Errorf("cannot resolve execution(%q): this execution has no parent", ParentRef)
		}
		return r.ParentIds[len(r.ParentIds)-1], nil
	case RerunRef:
		if r.RerunId == "" {
			return "", fmt.Errorf("cannot resolve execution(%q): this execution is not a rerun of another one", RerunRef)
		}
		return r.RerunId, nil
	}
	return ref, nil
}

func (r Resolver) knownRefs() []string {
	if r.Registry == nil {
		return nil
	}
	return r.Registry.Refs()
}
