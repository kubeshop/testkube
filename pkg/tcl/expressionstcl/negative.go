// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import "fmt"

type negative struct {
	expr Expression
}

func newNegative(expr Expression) Expression {
	return &negative{expr: expr}
}

func (s *negative) String() string {
	return fmt.Sprintf("!%s", s.expr.SafeString())
}

func (s *negative) SafeString() string {
	return s.String()
}

func (s *negative) Template() string {
	return "{{" + s.String() + "}}"
}

func (s *negative) Simplify(m MachineCore) (v Expression, err error) {
	s.expr, err = s.expr.Simplify(m)
	if err != nil {
		return nil, err
	}
	st := s.expr.Static()
	if st == nil {
		return s, nil
	}

	vv, err := st.BoolValue()
	if err != nil {
		return nil, err
	}
	return newStatic(!vv), nil
}

func (s *negative) Static() StaticValue {
	// FIXME: it should get environment to call
	return nil
}

func (s *negative) Accessors() map[string]struct{} {
	return s.expr.Accessors()
}

func (s *negative) Functions() map[string]struct{} {
	return s.expr.Functions()
}
