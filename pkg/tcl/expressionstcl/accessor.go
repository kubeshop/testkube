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

func (s *accessor) String() string {
	return s.name
}

func (s *accessor) SafeString() string {
	return s.String()
}

func (s *accessor) Template() string {
	return "{{" + s.String() + "}}"
}

func (s *accessor) Simplify(m MachineCore) (v Expression, err error) {
	if m == nil {
		return s, nil
	}

	result, ok, err := m.Get(s.name)
	if err != nil {
		return nil, fmt.Errorf("error while accessing %s: %s", s.String(), err.Error())
	}
	if ok {
		return result, nil
	}
	return s, nil
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
