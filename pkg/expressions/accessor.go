package expressions

import (
	"fmt"
	"strings"
)

type accessor struct {
	name     string
	fallback *Expression
}

func newAccessor(name string) Expression {
	// Map values based on wildcard
	segments := strings.Split(name, ".*")
	if len(segments) > 1 {
		return newCall("map", []CallArgument{
			{Expression: newAccessor(strings.Join(segments[0:len(segments)-1], ".*"))},
			{Expression: NewStringValue("_.value" + segments[len(segments)-1])},
		})
	}

	// Prepare fallback based on the segments
	segments = strings.Split(name, ".")
	var fallback *Expression
	if len(segments) > 1 {
		f := newPropertyAccessor(
			newAccessor(strings.Join(segments[0:len(segments)-1], ".")),
			segments[len(segments)-1],
		)
		fallback = &f
	}

	return &accessor{name: name, fallback: fallback}
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
		if ok && err == nil {
			return result, true, nil
		}
		if s.fallback != nil {
			var err2 error
			result, ok, err2 = (*s.fallback).SafeResolve(m...)
			if ok && err2 == nil {
				return result, true, nil
			}
		}
		if err != nil {
			return nil, false, fmt.Errorf("error while accessing %s: %w", s.String(), err)
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
