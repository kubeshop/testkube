package expressions

import (
	"fmt"
	"strings"
)

//go:generate go tool mockgen -destination=./mock_machine.go -package=expressions "github.com/kubeshop/testkube/pkg/expressions" Machine
type Machine interface {
	Get(name string) (Expression, bool, error)
	Call(name string, args []CallArgument) (Expression, bool, error)
}

type MachineAccessorExt = func(name string) (interface{}, bool, error)
type MachineAccessor = func(name string) (interface{}, bool)
type MachineFn = func(values ...StaticValue) (interface{}, bool, error)
type MachineFnExt = func(args []CallArgument) (interface{}, bool, error)

type machine struct {
	accessors []MachineAccessorExt
	functions map[string]MachineFnExt
}

func NewMachine() *machine {
	return &machine{
		accessors: make([]MachineAccessorExt, 0),
		functions: make(map[string]MachineFnExt),
	}
}

func (m *machine) Register(name string, value interface{}) *machine {
	return m.RegisterAccessor(func(n string) (interface{}, bool) {
		if n == name {
			return value, true
		}
		return nil, false
	})
}

func (m *machine) RegisterStringMap(prefix string, value map[string]string) *machine {
	if len(prefix) > 0 {
		prefix += "."
	}
	return m.RegisterAccessor(func(n string) (interface{}, bool) {
		if !strings.HasPrefix(n, prefix) {
			return nil, false
		}
		v, ok := value[n[len(prefix):]]
		return v, ok
	})
}

func (m *machine) RegisterMap(prefix string, value map[string]interface{}) *machine {
	if len(prefix) > 0 {
		prefix += "."
	}
	return m.RegisterAccessor(func(n string) (interface{}, bool) {
		if !strings.HasPrefix(n, prefix) {
			return nil, false
		}
		v, ok := value[n[len(prefix):]]
		return v, ok
	})
}

func (m *machine) RegisterAccessorExt(fn MachineAccessorExt) *machine {
	m.accessors = append(m.accessors, fn)
	return m
}

func (m *machine) RegisterAccessor(fn MachineAccessor) *machine {
	return m.RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		v, ok := fn(name)
		return v, ok, nil
	})
}

func areArgsResolved(args []CallArgument) bool {
	for i := range args {
		if args[i].Static() == nil {
			return false
		}
	}
	return true
}

func resolveArgs(args []CallArgument) ([]StaticValue, error) {
	v := make([]StaticValue, 0)
	for _, vv := range args {
		value := vv.Static()
		if vv.Spread {
			if value.IsNone() {
				continue
			}
			items, err := value.SliceValue()
			if err != nil {
				return nil, fmt.Errorf("spread operator (...) used against non-list parameter: %s", value)
			}
			staticItems := make([]StaticValue, len(items))
			for i := range items {
				staticItems[i] = NewValue(items[i])
			}
			v = append(v, staticItems...)
		} else {
			v = append(v, value)
		}
	}
	return v, nil
}

func ToMachineFunctionExt(fn MachineFn) MachineFnExt {
	return func(args []CallArgument) (interface{}, bool, error) {
		if !areArgsResolved(args) {
			return nil, false, nil
		}
		v, err := resolveArgs(args)
		if err != nil {
			return nil, true, err
		}
		return fn(v...)
	}
}

func (m *machine) RegisterFunction(name string, fn MachineFn) *machine {
	m.functions[name] = ToMachineFunctionExt(fn)
	return m
}

func (m *machine) RegisterFunctionExt(name string, fn MachineFnExt) *machine {
	m.functions[name] = fn
	return m
}

func (m *machine) Get(name string) (Expression, bool, error) {
	for i := range m.accessors {
		r, ok, err := m.accessors[i](name)
		if err != nil {
			return nil, true, err
		}
		if ok {
			if v, ok := r.(Expression); ok {
				return v, true, nil
			}
			return NewValue(r), true, nil
		}
	}
	return nil, false, nil
}

func (m *machine) Call(name string, args []CallArgument) (Expression, bool, error) {
	fn, ok := m.functions[name]
	if !ok {
		return nil, false, nil
	}
	r, ok, err := fn(args)
	if !ok || err != nil {
		return nil, ok, err
	}
	if v, ok := r.(Expression); ok {
		return v, true, nil
	}
	return NewValue(r), true, nil
}
