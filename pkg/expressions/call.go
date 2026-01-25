package expressions

import (
	"fmt"
	"maps"
	"strings"
)

type call struct {
	name string
	args []CallArgument
}

type CallArgument struct {
	Expression
	Spread bool
}

func newCall(name string, args []CallArgument) Expression {
	for i := range args {
		if args[i].Expression == nil {
			args[i].Expression = None
		}
	}
	return &call{name: name, args: args}
}

func (s *call) Type() Type {
	if IsStdFunction(s.name) {
		return GetStdFunctionReturnType(s.name)
	}
	return TypeUnknown
}

func (s *call) String() string {
	args := make([]string, len(s.args))
	for i, arg := range s.args {
		args[i] = arg.String()
		if arg.Spread {
			args[i] += "..."
		}
	}
	return fmt.Sprintf("%s(%s)", s.name, strings.Join(args, ","))
}

func (s *call) SafeString() string {
	return s.String()
}

func (s *call) Template() string {
	if s.name == stringCastStdFn {
		args := make([]string, len(s.args))
		for i, a := range s.args {
			args[i] = a.Template()
		}
		return strings.Join(args, "")
	}
	return "{{" + s.String() + "}}"
}

func (s *call) SafeResolve(m ...Machine) (v Expression, changed bool, err error) {
	var ch bool
	for i := range s.args {
		s.args[i].Expression, ch, err = s.args[i].SafeResolve(m...)
		changed = changed || ch
		if err != nil {
			return nil, changed, err
		}
	}
	if changed {
		return s, true, nil
	}
	result, ok, err := StdLibMachine.Call(s.name, s.args)
	if ok {
		if err != nil {
			return nil, true, fmt.Errorf("error while calling %s: %s", s.String(), err.Error())
		}
		return result, true, nil
	}
	for i := range m {
		result, ok, err = m[i].Call(s.name, s.args)
		if err != nil {
			return nil, true, fmt.Errorf("error while calling %s: %s", s.String(), err.Error())
		}
		if ok {
			return result, true, nil
		}
	}
	return s, changed, nil
}

func (s *call) Resolve(m ...Machine) (v Expression, err error) {
	return deepResolve(s, m...)
}

func (s *call) Static() StaticValue {
	return nil
}

func (s *call) Accessors() map[string]struct{} {
	result := make(map[string]struct{})
	for i := range s.args {
		maps.Copy(result, s.args[i].Accessors())
	}
	return result
}

func (s *call) Functions() map[string]struct{} {
	result := make(map[string]struct{})
	for i := range s.args {
		maps.Copy(result, s.args[i].Functions())
	}
	result[s.name] = struct{}{}
	return result
}
