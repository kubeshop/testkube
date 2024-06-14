// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressions

import (
	"fmt"
	"maps"
	"strings"
)

type call struct {
	name string
	args []callArgument
}

type callArgument struct {
	expr   Expression
	spread bool
}

func newCall(name string, args []callArgument) Expression {
	for i := range args {
		if args[i].expr == nil {
			args[i].expr = None
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
		args[i] = arg.expr.String()
		if arg.spread {
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
			args[i] = a.expr.Template()
		}
		return strings.Join(args, "")
	}
	return "{{" + s.String() + "}}"
}

func (s *call) isResolved() bool {
	for i := range s.args {
		if s.args[i].expr.Static() == nil {
			return false
		}
	}
	return true
}

func (s *call) resolvedArgs() ([]StaticValue, error) {
	v := make([]StaticValue, 0)
	for _, vv := range s.args {
		value := vv.expr.Static()
		if vv.spread {
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

func (s *call) SafeResolve(m ...Machine) (v Expression, changed bool, err error) {
	var ch bool
	for i := range s.args {
		s.args[i].expr, ch, err = s.args[i].expr.SafeResolve(m...)
		changed = changed || ch
		if err != nil {
			return nil, changed, err
		}
	}
	if s.isResolved() {
		args, err := s.resolvedArgs()
		if err != nil {
			return nil, true, err
		}
		result, ok, err := StdLibMachine.Call(s.name, args...)
		if ok {
			if err != nil {
				return nil, true, fmt.Errorf("error while calling %s: %s", s.String(), err.Error())
			}
			return result, true, nil
		}
		for i := range m {
			result, ok, err = m[i].Call(s.name, args...)
			if err != nil {
				return nil, true, fmt.Errorf("error while calling %s: %s", s.String(), err.Error())
			}
			if ok {
				return result, true, nil
			}
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
		maps.Copy(result, s.args[i].expr.Accessors())
	}
	return result
}

func (s *call) Functions() map[string]struct{} {
	result := make(map[string]struct{})
	for i := range s.args {
		maps.Copy(result, s.args[i].expr.Functions())
	}
	result[s.name] = struct{}{}
	return result
}
