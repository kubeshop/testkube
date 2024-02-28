// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"encoding/json"
	"strings"
)

type static struct {
	value interface{}
}

var none *static
var None StaticValue = none

func NewValue(value interface{}) StaticValue {
	if value == noneValue {
		return None
	}
	return &static{value: value}
}

func NewStringValue(value interface{}) StaticValue {
	v, _ := toString(value)
	return NewValue(v)
}

func (s *static) Type() Type {
	if s == nil {
		return TypeUnknown
	}
	switch s.value.(type) {
	case int64:
		return TypeInt64
	case float64:
		return TypeFloat64
	case string:
		return TypeString
	case bool:
		return TypeBool
	default:
		return TypeUnknown
	}
}

func (s *static) String() string {
	if s.IsNone() {
		return "null"
	}
	b, _ := json.Marshal(s.value)
	if len(b) == 0 {
		return "null"
	}
	r := string(b)
	if s.IsMap() && r == "null" {
		return "{}"
	}
	if s.IsSlice() && r == "null" {
		return "[]"
	}
	return r
}

func (s *static) SafeString() string {
	return s.String()
}

func (s *static) Template() string {
	if s.IsNone() {
		return ""
	}
	v, _ := s.StringValue()
	return strings.ReplaceAll(v, "{{", "{{\"{{\"}}")
}

func (s *static) SafeResolve(_ ...Machine) (Expression, bool, error) {
	return s, false, nil
}

func (s *static) Resolve(_ ...Machine) (Expression, error) {
	return s, nil
}

func (s *static) Static() StaticValue {
	return s
}

func (s *static) IsNone() bool {
	return s == nil
}

func (s *static) IsString() bool {
	return !s.IsNone() && isString(s.value)
}

func (s *static) IsBool() bool {
	return !s.IsNone() && isBool(s.value)
}

func (s *static) IsInt() bool {
	return !s.IsNone() && isInt(s.value)
}

func (s *static) IsNumber() bool {
	return !s.IsNone() && isNumber(s.value)
}

func (s *static) IsMap() bool {
	return !s.IsNone() && isMap(s.value)
}

func (s *static) IsSlice() bool {
	return !s.IsNone() && isSlice(s.value)
}

func (s *static) Value() interface{} {
	if s.IsNone() {
		return noneValue
	}
	return s.value
}

func (s *static) StringValue() (string, error) {
	return toString(s.Value())
}

func (s *static) BoolValue() (bool, error) {
	return toBool(s.Value())
}

func (s *static) IntValue() (int64, error) {
	return toInt(s.Value())
}

func (s *static) FloatValue() (float64, error) {
	return toFloat(s.Value())
}

func (s *static) MapValue() (map[string]interface{}, error) {
	return toMap(s.Value())
}

func (s *static) SliceValue() ([]interface{}, error) {
	return toSlice(s.Value())
}

func (s *static) Accessors() map[string]struct{} {
	return nil
}

func (s *static) Functions() map[string]struct{} {
	return nil
}
