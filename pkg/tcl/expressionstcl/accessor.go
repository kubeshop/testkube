// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"fmt"
)

type accessor struct {
	name string
}

func newAccessor(name string) Expression {
	return &accessor{name: name}
}

func (s *accessor) Type() Type {
	return TypeUnknown
}

func (s *accessor) String() string {
	return s.name
}

func (s *accessor) SafeString() string {
	return s.String()
}

func (s *accessor) Template() string {
	return "{{" + s.String() + "}}"
}

func (s *accessor) SafeResolve(m ...Machine) (v Expression, changed bool, err error) {
	if m == nil {
		return s, false, nil
	}

	for i := range m {
		result, ok, err := m[i].Get(s.name)
		if err != nil {
			return nil, false, fmt.Errorf("error while accessing %s: %s", s.String(), err.Error())
		}
		if ok {
			return result, true, nil
		}
	}
	return s, false, nil
}

func (s *accessor) Resolve(m ...Machine) (v Expression, err error) {
	return deepResolve(s, m...)
}

func (s *accessor) Static() StaticValue {
	return nil
}

func (s *accessor) Accessors() map[string]struct{} {
	return map[string]struct{}{s.name: {}}
}

func (s *accessor) Functions() map[string]struct{} {
	return nil
}
