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

func newStatic(value interface{}) StaticValue {
	return &static{value: value}
}

func newStaticString(value interface{}) StaticValue {
	v, _ := toString(value)
	return newStatic(v)
}

func (s *static) WillBeString() bool {
	return s.IsString()
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
	if strings.Contains(v, "{{") {
		return "{{" + s.String() + "}}"
	}
	return v
}

func (s *static) Simplify(_ MachineCore) (Expression, error) {
	return s, nil
}

func (s *static) Static() StaticValue {
	return s
}

func (s *static) IsNone() bool {
	return isNone(s.value)
}

func (s *static) IsString() bool {
	return isString(s.value)
}

func (s *static) IsBool() bool {
	return isBool(s.value)
}

func (s *static) IsInt() bool {
	return isInt(s.value)
}

func (s *static) IsNumber() bool {
	return isNumber(s.value)
}

func (s *static) IsMap() bool {
	return isMap(s.value)
}

func (s *static) IsSlice() bool {
	return isSlice(s.value)
}

func (s *static) Value() interface{} {
	return s.value
}

func (s *static) StringValue() (string, error) {
	return toString(s.value)
}

func (s *static) BoolValue() (bool, error) {
	return toBool(s.value)
}

func (s *static) IntValue() (int64, error) {
	return toInt(s.value)
}

func (s *static) FloatValue() (float64, error) {
	return toFloat(s.value)
}

func (s *static) MapValue() (map[string]interface{}, error) {
	return toMap(s.value)
}

func (s *static) SliceValue() ([]interface{}, error) {
	return toSlice(s.value)
}

func (s *static) Accessors() map[string]struct{} {
	return nil
}

func (s *static) Functions() map[string]struct{} {
	return nil
}
