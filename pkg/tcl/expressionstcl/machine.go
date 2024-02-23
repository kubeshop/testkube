// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

//go:generate mockgen -destination=./mock_machinecore.go -package=expressionstcl "github.com/kubeshop/testkube/pkg/tcl/expressionstcl" MachineCore
type MachineCore interface {
	Get(name string) (Expression, bool, error)
	Call(name string, args ...StaticValue) (Expression, bool, error)
}

//go:generate mockgen -destination=./mock_machine.go -package=expressionstcl "github.com/kubeshop/testkube/pkg/tcl/expressionstcl" Machine
type Machine interface {
	MachineCore
	Finalizer() MachineCore
}

type MachineAccessor = func(name string) (interface{}, bool)
type MachineFn = func(values ...StaticValue) (interface{}, bool, error)

type machine struct {
	accessors []MachineAccessor
	functions map[string]MachineFn
	finalizer *finalizer
}

func NewMachine() *machine {
	m := &machine{
		accessors: make([]MachineAccessor, 0),
		functions: make(map[string]MachineFn),
	}
	m.finalizer = &finalizer{machine: m}
	return m
}

func (m *machine) Register(name string, value interface{}) *machine {
	return m.RegisterAccessor(func(n string) (interface{}, bool) {
		if n == name {
			return value, true
		}
		return nil, false
	})
}

func (m *machine) RegisterAccessor(fn MachineAccessor) *machine {
	m.accessors = append(m.accessors, fn)
	return m
}

func (m *machine) RegisterFunction(name string, fn MachineFn) *machine {
	m.functions[name] = fn
	return m
}

func (m *machine) Get(name string) (Expression, bool, error) {
	for i := range m.accessors {
		r, ok := m.accessors[i](name)
		if ok {
			if v, ok := r.(Expression); ok {
				return v, true, nil
			}
			return NewValue(r), true, nil
		}
	}
	return nil, false, nil
}

func (m *machine) Call(name string, args ...StaticValue) (Expression, bool, error) {
	fn, ok := m.functions[name]
	if !ok {
		return nil, false, nil
	}
	r, ok, err := fn(args...)
	if !ok || err != nil {
		return nil, ok, err
	}
	if v, ok := r.(Expression); ok {
		return v, true, nil
	}
	return NewValue(r), true, nil
}

func (m *machine) Finalizer() MachineCore {
	return m.finalizer
}
