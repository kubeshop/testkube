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
	"strings"

	"github.com/pkg/errors"
)

type propertyAccessor struct {
	value Expression
	path  []string
}

func newPropertyAccessor(value Expression, path string) Expression {
	return &propertyAccessor{value: value, path: strings.Split(path, ".")}
}

func (s *propertyAccessor) Type() Type {
	return TypeUnknown
}

func (s *propertyAccessor) String() string {
	return fmt.Sprintf("%s.%s", s.value.SafeString(), strings.Join(s.path, "."))
}

func (s *propertyAccessor) SafeString() string {
	return s.String()
}

func (s *propertyAccessor) Template() string {
	return "{{" + s.String() + "}}"
}

func (s *propertyAccessor) SafeResolve(m ...Machine) (v Expression, changed bool, err error) {
	if s.value.Static() == nil {
		s.value, changed, err = s.value.SafeResolve(m...)
		if !changed || err != nil || s.value.Static() == nil {
			return s, changed, err
		}
	}
	current := s.value
	for i := 0; i < len(s.path); i++ {
		current, err = CallStdFunction("at", current, s.path[i])
		if err != nil {
			return nil, changed, errors.Wrap(err, strings.Join(s.path[:i+1], "."))
		}
	}
	return current, true, nil
}

func (s *propertyAccessor) Resolve(m ...Machine) (v Expression, err error) {
	return deepResolve(s, m...)
}

func (s *propertyAccessor) Static() StaticValue {
	return nil
}

func (s *propertyAccessor) Accessors() map[string]struct{} {
	return nil
}

func (s *propertyAccessor) Functions() map[string]struct{} {
	return nil
}
