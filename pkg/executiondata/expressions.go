package executiondata

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	// ExecutionFn is the expression function reading another execution's data.
	ExecutionFn = "execution"
	// ReadArtifactFn is the expression function reading another execution's artifact.
	ReadArtifactFn = "read_artifact"
)

// MachineOptions configures the executiondata expression machine.
type MachineOptions struct {
	// Registry holds the executions the current workflow scheduled. May be nil.
	Registry *Registry
	// Repository resolves executions the registry does not know. May be nil, in
	// which case only registered executions can be referenced.
	Repository ExecutionRepository
	// ParentIds is the chain of executions that led to this one, oldest first.
	ParentIds []string
	// ArtifactClient downloads from object storage, which read_artifact() talks to
	// directly rather than through the control plane. May be nil, in which case storage
	// certificates are verified - see NewArtifactClient.
	ArtifactClient *http.Client
}

// NewMachine registers the functions reading data from other executions.
//
// The `execution` function does not collide with the `execution.*` accessor
// registered by testworkflowconfig: the expression machine keeps functions and
// accessors in separate namespaces.
func NewMachine(opts MachineOptions) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction(ExecutionFn, func(values ...expressions.StaticValue) (interface{}, bool, error) {
			ref, index, err := parseRefArgs(ExecutionFn, values)
			if err != nil {
				return nil, true, err
			}
			execution, err := opts.resolve(ref, index)
			if err != nil {
				return nil, true, err
			}
			return execution.AsMap(), true, nil
		}).
		RegisterFunction(ReadArtifactFn, func(values ...expressions.StaticValue) (interface{}, bool, error) {
			if len(values) != 2 {
				return nil, true, fmt.Errorf("%q function expects 2 arguments, %d provided", ReadArtifactFn, len(values))
			}
			ref, _, err := parseRefArgs(ReadArtifactFn, values[:1])
			if err != nil {
				return nil, true, err
			}
			if !values[1].IsString() {
				return nil, true, fmt.Errorf("%q function expects the path to be a string, %s provided", ReadArtifactFn, values[1].String())
			}
			path, _ := values[1].StringValue()
			if path == "" {
				return nil, true, fmt.Errorf("%q function expects a non-empty path", ReadArtifactFn)
			}

			execution, err := opts.resolve(ref, 0)
			if err != nil {
				return nil, true, err
			}
			content, err := ReadArtifact(context.Background(), opts.Repository, opts.ArtifactClient, execution.Id, path)
			if err != nil {
				return nil, true, fmt.Errorf("reading artifact of execution %q: %w", ref, err)
			}
			return content, true, nil
		})
}

// resolve finds an execution by reference, preferring the local registry so that
// the common case costs no network call.
func (o MachineOptions) resolve(ref string, index int64) (Execution, error) {
	if o.Registry != nil {
		execution, ok, err := o.Registry.Lookup(ref, index)
		if err != nil {
			return Execution{}, err
		}
		if ok {
			return execution, nil
		}
	}

	id := ref
	if ref == ParentRef {
		if len(o.ParentIds) == 0 {
			return Execution{}, fmt.Errorf("cannot resolve execution(%q): this execution has no parent", ParentRef)
		}
		id = o.ParentIds[len(o.ParentIds)-1]
	}

	// Fan-out indexes only exist within the local registry - anything resolved
	// through the control plane is a single execution addressed by its id.
	if index != 0 {
		return Execution{}, UnknownRefError(ref, index, o.knownRefs())
	}

	if o.Repository == nil {
		return Execution{}, UnknownRefError(ref, index, o.knownRefs())
	}

	execution, err := o.Repository.Get(context.Background(), id)
	if err != nil {
		if ref == ParentRef {
			return Execution{}, fmt.Errorf("reading parent execution %s: %w", id, err)
		}
		return Execution{}, fmt.Errorf("reading execution %q: %w", ref, err)
	}
	if execution.Id == "" {
		return Execution{}, UnknownRefError(ref, index, o.knownRefs())
	}
	if ref == ParentRef {
		execution.Alias = ParentRef
	}
	return execution, nil
}

func (o MachineOptions) knownRefs() []string {
	if o.Registry == nil {
		return nil
	}
	return o.Registry.Refs()
}

// parseRefArgs reads the (ref) or (ref, index) argument pair shared by the
// executiondata functions that address another execution.
func parseRefArgs(fn string, values []expressions.StaticValue) (string, int64, error) {
	if len(values) < 1 || len(values) > 2 {
		return "", 0, fmt.Errorf("%q function expects 1-2 arguments, %d provided", fn, len(values))
	}
	if !values[0].IsString() {
		return "", 0, fmt.Errorf("%q function expects the reference to be a string, %s provided", fn, values[0].String())
	}
	ref, _ := values[0].StringValue()
	if ref == "" {
		return "", 0, fmt.Errorf("%q function expects a non-empty reference", fn)
	}
	if len(values) == 1 {
		return ref, 0, nil
	}
	index, err := values[1].IntValue()
	if err != nil {
		return "", 0, fmt.Errorf("%q function expects the index to be a number, %s provided", fn, values[1].String())
	}
	return ref, index, nil
}
