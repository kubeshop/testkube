package expressions

import "strings"

type limitedMachine struct {
	prefix  string
	machine Machine
}

func PrefixMachine(prefix string, machine Machine) Machine {
	return &limitedMachine{
		prefix:  prefix,
		machine: machine,
	}
}

func (m *limitedMachine) Get(name string) (Expression, bool, error) {
	if strings.HasPrefix(name, m.prefix) {
		return m.machine.Get(name)
	}
	return nil, false, nil
}

func (m *limitedMachine) Call(name string, args []CallArgument) (Expression, bool, error) {
	if strings.HasPrefix(name, m.prefix) {
		return m.machine.Call(name, args)
	}
	return nil, false, nil
}

type combinedMachine struct {
	machines []Machine
}

func CombinedMachines(machines ...Machine) Machine {
	return &combinedMachine{machines: machines}
}

func (m *combinedMachine) Get(name string) (Expression, bool, error) {
	for i := range m.machines {
		v, ok, err := m.machines[i].Get(name)
		if err != nil || ok {
			return v, ok, err
		}
	}
	return nil, false, nil
}

func (m *combinedMachine) Call(name string, args []CallArgument) (Expression, bool, error) {
	for i := range m.machines {
		v, ok, err := m.machines[i].Call(name, args)
		if err != nil || ok {
			return v, ok, err
		}
	}
	return nil, false, nil
}

func ReplacePrefixMachine(from string, to string) Machine {
	return NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
		if strings.HasPrefix(name, from) {
			return newAccessor(to + name[len(from):]), true
		}
		return nil, false
	})
}
