// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"errors"
	"fmt"
	"maps"
	math2 "math"
)

type operator string

const (
	operatorEquals         operator = "="
	operatorEqualsAlias    operator = "=="
	operatorNotEquals      operator = "!="
	operatorNotEqualsAlias operator = "<>"
	operatorGt             operator = ">"
	operatorGte            operator = ">="
	operatorLt             operator = "<"
	operatorLte            operator = "<="
	operatorAnd            operator = "&&"
	operatorOr             operator = "||"
	operatorAdd            operator = "+"
	operatorSubtract       operator = "-"
	operatorModulo         operator = "%"
	operatorDivide         operator = "/"
	operatorMultiply       operator = "*"
	operatorPower          operator = "**"
)

func getOperatorPriority(op operator) int {
	switch op {
	case operatorAnd, operatorOr:
		return 0
	case operatorEquals, operatorEqualsAlias, operatorNotEquals, operatorNotEqualsAlias,
		operatorGt, operatorGte, operatorLt, operatorLte:
		return 1
	case operatorAdd, operatorSubtract:
		return 2
	case operatorMultiply, operatorDivide, operatorModulo:
		return 3
	case operatorPower:
		return 4
	}
	panic("unknown operator: " + op)
}

type math struct {
	operator operator
	left     Expression
	right    Expression
}

func newMath(operator operator, left Expression, right Expression) Expression {
	if left == nil {
		left = None
	}
	if right == nil {
		right = None
	}
	return &math{operator: operator, left: left, right: right}
}

func runOp[T interface{}, U interface{}](v1 StaticValue, v2 StaticValue, mapper func(value StaticValue) (T, error), op func(s1, s2 T) U) (StaticValue, error) {
	s1, err1 := mapper(v1)
	if err1 != nil {
		return nil, err1
	}
	s2, err2 := mapper(v2)
	if err2 != nil {
		return nil, err2
	}
	return NewValue(op(s1, s2)), nil
}

func staticString(v StaticValue) (string, error) {
	return v.StringValue()
}

func staticFloat(v StaticValue) (float64, error) {
	return v.FloatValue()
}

func staticBool(v StaticValue) (bool, error) {
	return v.BoolValue()
}

func (s *math) performMath(v1 StaticValue, v2 StaticValue) (StaticValue, error) {
	switch s.operator {
	case operatorEquals, operatorEqualsAlias:
		return runOp(v1, v2, staticString, func(s1, s2 string) bool {
			return s1 == s2
		})
	case operatorNotEquals, operatorNotEqualsAlias:
		return runOp(v1, v2, staticString, func(s1, s2 string) bool {
			return s1 != s2
		})
	case operatorGt:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) bool {
			return s1 > s2
		})
	case operatorLt:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) bool {
			return s1 < s2
		})
	case operatorGte:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) bool {
			return s1 >= s2
		})
	case operatorLte:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) bool {
			return s1 <= s2
		})
	case operatorAnd:
		return runOp(v1, v2, staticBool, func(s1, s2 bool) interface{} {
			if s1 {
				return v2.Value()
			}
			return v1.Value()
		})
	case operatorOr:
		return runOp(v1, v2, staticBool, func(s1, s2 bool) interface{} {
			if s1 {
				return v1.Value()
			}
			return v2.Value()
		})
	case operatorAdd:
		if v1.IsString() || v2.IsString() {
			return runOp(v1, v2, staticString, func(s1, s2 string) string {
				return s1 + s2
			})
		}
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			return s1 + s2
		})
	case operatorSubtract:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			return s1 - s2
		})
	case operatorModulo:
		divideByZero := false
		res, err := runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			if s2 == 0 {
				divideByZero = true
				return 0
			}
			return math2.Mod(s1, s2)
		})
		if divideByZero {
			return nil, errors.New("cannot modulo by zero")
		}
		return res, err
	case operatorDivide:
		divideByZero := false
		res, err := runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			if s2 == 0 {
				divideByZero = true
				return 0
			}
			return s1 / s2
		})
		if divideByZero {
			return nil, errors.New("cannot divide by zero")
		}
		return res, err
	case operatorMultiply:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			return s1 * s2
		})
	case operatorPower:
		return runOp(v1, v2, staticFloat, func(s1, s2 float64) float64 {
			return math2.Pow(s1, s2)
		})
	default:
	}
	return nil, fmt.Errorf("unknown math operator: %s", s.operator)
}

func (s *math) Type() Type {
	l := s.left.Type()
	r := s.right.Type()
	switch s.operator {
	case operatorAnd, operatorOr:
		if l == r {
			return l
		}
		return TypeUnknown
	case operatorPower, operatorModulo, operatorSubtract, operatorMultiply, operatorDivide:
		return TypeFloat64
	case operatorAdd:
		if l == TypeString || r == TypeString {
			return TypeString
		}
		return TypeFloat64
	case operatorEquals, operatorNotEquals, operatorEqualsAlias, operatorNotEqualsAlias, operatorGt, operatorLt, operatorGte, operatorLte:
		return TypeBool
	default:
		return TypeUnknown
	}
}

func (s *math) itemString(v Expression) string {
	if vv, ok := v.(*math); ok {
		if getOperatorPriority(vv.operator) >= getOperatorPriority(s.operator) {
			return v.String()
		}
	}
	return v.SafeString()
}

func (s *math) String() string {
	return s.itemString(s.left) + string(s.operator) + s.itemString(s.right)
}

func (s *math) SafeString() string {
	return "(" + s.String() + ")"
}

func (s *math) Template() string {
	// Simplify the template when it is possible
	if s.operator == operatorAdd && s.Type() == TypeString {
		return s.left.Template() + s.right.Template()
	}
	return "{{" + s.String() + "}}"
}

func (s *math) SafeResolve(m ...Machine) (v Expression, changed bool, err error) {
	var ch bool
	s.left, ch, err = s.left.SafeResolve(m...)
	changed = changed || ch
	if err != nil {
		return
	}

	// Fast track for cutting dead paths
	if s.left.Static() != nil {
		if s.operator == operatorAnd {
			b, err := s.left.Static().BoolValue()
			if err == nil && !b {
				return s.left, true, nil
			} else if err == nil {
				return s.right, true, nil
			}
		} else if s.operator == operatorOr {
			b, err := s.left.Static().BoolValue()
			if err == nil && b {
				return s.left, true, nil
			} else if err == nil {
				return s.right, true, nil
			}
		}
	}

	s.right, ch, err = s.right.SafeResolve(m...)
	changed = changed || ch
	if err != nil {
		return
	}

	// Fast track for cutting dead paths
	t := s.left.Type()
	if s.left.Static() == nil && s.right.Static() != nil && t != TypeUnknown && t == s.right.Type() && t == TypeBool {
		if s.operator == operatorAnd {
			b, err := s.right.Static().BoolValue()
			if err == nil && !b {
				return s.right, true, nil
			} else if err == nil {
				return s.left, true, nil
			}
		} else if s.operator == operatorOr {
			b, err := s.right.Static().BoolValue()
			if err == nil && b {
				return s.right, true, nil
			} else if err == nil {
				return s.left, true, nil
			}
		}
	}

	if s.left.Static() != nil && s.right.Static() != nil {
		res, err := s.performMath(s.left.Static(), s.right.Static())
		if err != nil {
			return nil, changed, fmt.Errorf("error while performing math: %s: %s", s.String(), err)
		}
		return res, true, nil
	}
	return s, changed, nil
}

func (s *math) Resolve(m ...Machine) (v Expression, err error) {
	return deepResolve(s, m...)
}

func (s *math) Static() StaticValue {
	return nil
}

func (s *math) Accessors() map[string]struct{} {
	result := make(map[string]struct{})
	maps.Copy(result, s.left.Accessors())
	maps.Copy(result, s.right.Accessors())
	return result
}

func (s *math) Functions() map[string]struct{} {
	result := make(map[string]struct{})
	maps.Copy(result, s.left.Functions())
	maps.Copy(result, s.right.Functions())
	return result
}
