// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func must[T interface{}](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}

func errOnly(_ interface{}, e error) error {
	return e
}

func TestStaticBool(t *testing.T) {
	// Types
	assert.Equal(t, "false", newStatic(false).String())
	assert.Equal(t, "true", newStatic(true).String())
	assert.Equal(t, true, newStatic(false).IsBool())
	assert.Equal(t, true, newStatic(true).IsBool())
	assert.Equal(t, false, newStatic(false).IsNone())
	assert.Equal(t, false, newStatic(true).IsNone())
	assert.Equal(t, false, newStatic(true).IsInt())
	assert.Equal(t, false, newStatic(true).IsNumber())
	assert.Equal(t, false, newStatic(true).IsString())
	assert.Equal(t, false, newStatic(true).IsMap())
	assert.Equal(t, false, newStatic(true).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic(false).BoolValue()))
	assert.Equal(t, true, must(newStatic(true).BoolValue()))
	assert.Error(t, errOnly(newStatic(true).IntValue()))
	assert.Error(t, errOnly(newStatic(true).FloatValue()))
	assert.Equal(t, "true", must(newStatic(true).StringValue()))
	assert.Equal(t, "false", must(newStatic(false).StringValue()))
	assert.Error(t, errOnly(newStatic(true).MapValue()))
	assert.Error(t, errOnly(newStatic(true).SliceValue()))
}

func TestStaticInt(t *testing.T) {
	// Types
	assert.Equal(t, "0", newStatic(0).String())
	assert.Equal(t, "1", newStatic(1).String())
	assert.Equal(t, false, newStatic(0).IsBool())
	assert.Equal(t, false, newStatic(1).IsBool())
	assert.Equal(t, false, newStatic(0).IsNone())
	assert.Equal(t, false, newStatic(1).IsNone())
	assert.Equal(t, true, newStatic(1).IsInt())
	assert.Equal(t, true, newStatic(1).IsNumber())
	assert.Equal(t, false, newStatic(1).IsString())
	assert.Equal(t, false, newStatic(1).IsMap())
	assert.Equal(t, false, newStatic(1).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic(0).BoolValue()))
	assert.Equal(t, true, must(newStatic(1).BoolValue()))
	assert.Equal(t, int64(1), must(newStatic(1).IntValue()))
	assert.Equal(t, 1.0, must(newStatic(1).FloatValue()))
	assert.Equal(t, "1", must(newStatic(1).StringValue()))
	assert.Error(t, errOnly(newStatic(1).MapValue()))
	assert.Error(t, errOnly(newStatic(1).SliceValue()))
}

func TestStaticFloat(t *testing.T) {
	// Types
	assert.Equal(t, "0", newStatic(0.0).String())
	assert.Equal(t, "1.5", newStatic(1.5).String())
	assert.Equal(t, false, newStatic(0.0).IsBool())
	assert.Equal(t, false, newStatic(1.0).IsBool())
	assert.Equal(t, false, newStatic(1.5).IsBool())
	assert.Equal(t, false, newStatic(0.0).IsNone())
	assert.Equal(t, false, newStatic(1.0).IsNone())
	assert.Equal(t, false, newStatic(1.5).IsNone())
	assert.Equal(t, true, newStatic(1.0).IsInt())
	assert.Equal(t, false, newStatic(1.8).IsInt())
	assert.Equal(t, true, newStatic(1.5).IsNumber())
	assert.Equal(t, false, newStatic(1.7).IsString())
	assert.Equal(t, false, newStatic(1.7).IsMap())
	assert.Equal(t, false, newStatic(1.3).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic(0.0).BoolValue()))
	assert.Equal(t, true, must(newStatic(0.5).BoolValue()))
	assert.Equal(t, true, must(newStatic(1.0).BoolValue()))
	assert.Equal(t, true, must(newStatic(1.5).BoolValue()))
	assert.Equal(t, int64(1), must(newStatic(1.8).IntValue()))
	assert.Equal(t, 1.8, must(newStatic(1.8).FloatValue()))
	assert.Equal(t, "1.877778", must(newStatic(1.877778).StringValue()))
	assert.Equal(t, "1.88", must(newStatic(1.88).StringValue()))
	assert.Error(t, errOnly(newStatic(1.8).MapValue()))
	assert.Error(t, errOnly(newStatic(1.8).SliceValue()))
}

func TestStaticString(t *testing.T) {
	// Types
	assert.Equal(t, `""`, newStatic("").String())
	assert.Equal(t, `"value"`, newStatic("value").String())
	assert.Equal(t, `"v\"alue"`, newStatic("v\"alue").String())
	assert.Equal(t, false, newStatic("").IsBool())
	assert.Equal(t, false, newStatic("value").IsBool())
	assert.Equal(t, false, newStatic("").IsNone())
	assert.Equal(t, false, newStatic("value").IsNone())
	assert.Equal(t, false, newStatic("5").IsInt())
	assert.Equal(t, false, newStatic("value").IsInt())
	assert.Equal(t, false, newStatic("5").IsNumber())
	assert.Equal(t, false, newStatic("value").IsNumber())
	assert.Equal(t, true, newStatic("").IsString())
	assert.Equal(t, true, newStatic("value").IsString())
	assert.Equal(t, false, newStatic("value").IsMap())
	assert.Equal(t, false, newStatic("value").IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic("").BoolValue()))
	assert.Equal(t, false, must(newStatic("0").BoolValue()))
	assert.Equal(t, false, must(newStatic("off").BoolValue()))
	assert.Equal(t, false, must(newStatic("false").BoolValue()))
	assert.Equal(t, true, must(newStatic("False").BoolValue()))
	assert.Equal(t, true, must(newStatic("true").BoolValue()))
	assert.Equal(t, true, must(newStatic("on").BoolValue()))
	assert.Equal(t, true, must(newStatic("1").BoolValue()))
	assert.Equal(t, true, must(newStatic("something").BoolValue()))
	assert.Equal(t, int64(1), must(newStatic("1").IntValue()))
	assert.Equal(t, int64(1), must(newStatic("1.5").IntValue()))
	assert.Error(t, errOnly(newStatic("").IntValue()))
	assert.Error(t, errOnly(newStatic("5 apples").IntValue()))
	assert.Equal(t, 1.0, must(newStatic("1").FloatValue()))
	assert.Equal(t, 1.5, must(newStatic("1.5").FloatValue()))
	assert.Error(t, errOnly(newStatic("").FloatValue()))
	assert.Error(t, errOnly(newStatic("5 apples").FloatValue()))
	assert.Equal(t, "", must(newStatic("").StringValue()))
	assert.Equal(t, "value", must(newStatic("value").StringValue()))
	assert.Equal(t, `v"alu\e`, must(newStatic(`v"alu\e`).StringValue()))
	assert.Error(t, errOnly(newStatic("").MapValue()))
	assert.Error(t, errOnly(newStatic("v").MapValue()))
	assert.Error(t, errOnly(newStatic("").SliceValue()))
	assert.Error(t, errOnly(newStatic("v").SliceValue()))
}

func TestStaticMap(t *testing.T) {
	// Types
	assert.Equal(t, "{}", newStatic(map[string]interface{}(nil)).String())
	assert.Equal(t, "{}", newStatic(map[string]string(nil)).String())
	assert.Equal(t, `{"a":"b"}`, newStatic(map[string]string{"a": "b"}).String())
	assert.Equal(t, `{"3":"b"}`, newStatic(map[int]string{3: "b"}).String())
	assert.Equal(t, false, newStatic(map[string]interface{}(nil)).IsBool())
	assert.Equal(t, false, newStatic(map[string]interface{}{}).IsBool())
	assert.Equal(t, false, newStatic(map[string]interface{}{"a": "b"}).IsBool())
	assert.Equal(t, false, newStatic(map[string]interface{}(nil)).IsNone())
	assert.Equal(t, false, newStatic(map[string]interface{}{}).IsNone())
	assert.Equal(t, false, newStatic(map[string]interface{}{"a": "b"}).IsNone())
	assert.Equal(t, false, newStatic(map[int]interface{}{3: "3"}).IsInt())
	assert.Equal(t, false, newStatic(map[int]interface{}{3: "3"}).IsNumber())
	assert.Equal(t, false, newStatic(map[int]interface{}{3: "3"}).IsString())
	assert.Equal(t, true, newStatic(map[string]interface{}(nil)).IsMap())
	assert.Equal(t, true, newStatic(map[string]interface{}{}).IsMap())
	assert.Equal(t, true, newStatic(map[string]interface{}{"a": "b"}).IsMap())
	assert.Equal(t, false, newStatic(map[string]interface{}(nil)).IsSlice())
	assert.Equal(t, false, newStatic(map[string]interface{}{}).IsSlice())
	assert.Equal(t, false, newStatic(map[string]interface{}{"a": "b"}).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic(map[string]string{}).BoolValue()))
	assert.Equal(t, false, must(newStatic(map[string]string(nil)).BoolValue()))
	assert.Equal(t, true, must(newStatic(map[string]string{"a": "b"}).BoolValue()))
	assert.Error(t, errOnly(newStatic(map[string]string(nil)).IntValue()))
	assert.Error(t, errOnly(newStatic(map[string]string{}).IntValue()))
	assert.Error(t, errOnly(newStatic(map[string]string{"a": "b"}).IntValue()))
	assert.Error(t, errOnly(newStatic(map[string]string(nil)).FloatValue()))
	assert.Error(t, errOnly(newStatic(map[string]string{}).FloatValue()))
	assert.Error(t, errOnly(newStatic(map[string]string{"a": "b"}).FloatValue()))
	assert.Equal(t, "{}", must(newStatic(map[string]string(nil)).StringValue()))
	assert.Equal(t, "{}", must(newStatic(map[string]string{}).StringValue()))
	assert.Equal(t, `{"a":"b"}`, must(newStatic(map[string]string{"a": "b"}).StringValue()))
	assert.Equal(t, map[string]interface{}{}, must(newStatic(map[string]string(nil)).MapValue()))
	assert.Equal(t, map[string]interface{}{}, must(newStatic(map[string]string{}).MapValue()))
	assert.Equal(t, map[string]interface{}{"a": "b"}, must(newStatic(map[string]string{"a": "b"}).MapValue()))
	assert.Error(t, errOnly(newStatic(map[string]string(nil)).SliceValue()))
	assert.Error(t, errOnly(newStatic(map[int]string{}).SliceValue()))
	assert.Error(t, errOnly(newStatic(map[int]string{3: "a"}).SliceValue()))
}

func TestStaticSlice(t *testing.T) {
	// Types
	assert.Equal(t, "[]", newStatic([]interface{}(nil)).String())
	assert.Equal(t, "[]", newStatic([]string(nil)).String())
	assert.Equal(t, `["a","b"]`, newStatic([]string{"a", "b"}).String())
	assert.Equal(t, `[3]`, newStatic([]int{3}).String())
	assert.Equal(t, false, newStatic([]interface{}(nil)).IsBool())
	assert.Equal(t, false, newStatic([]interface{}{}).IsBool())
	assert.Equal(t, false, newStatic([]interface{}{"a", "b"}).IsBool())
	assert.Equal(t, false, newStatic([]interface{}(nil)).IsNone())
	assert.Equal(t, false, newStatic([]interface{}{}).IsNone())
	assert.Equal(t, false, newStatic([]interface{}{"a", "b"}).IsNone())
	assert.Equal(t, false, newStatic([]interface{}{3: "3"}).IsInt())
	assert.Equal(t, false, newStatic([]interface{}{3: "3"}).IsNumber())
	assert.Equal(t, false, newStatic([]interface{}{3: "3"}).IsString())
	assert.Equal(t, false, newStatic([]interface{}(nil)).IsMap())
	assert.Equal(t, false, newStatic([]interface{}{}).IsMap())
	assert.Equal(t, false, newStatic([]interface{}{"a", "b"}).IsMap())
	assert.Equal(t, true, newStatic([]interface{}(nil)).IsSlice())
	assert.Equal(t, true, newStatic([]interface{}{}).IsSlice())
	assert.Equal(t, true, newStatic([]interface{}{"a", "b"}).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic([]string{}).BoolValue()))
	assert.Equal(t, false, must(newStatic([]string(nil)).BoolValue()))
	assert.Equal(t, true, must(newStatic([]string{"a", "b"}).BoolValue()))
	assert.Error(t, errOnly(newStatic([]string(nil)).IntValue()))
	assert.Error(t, errOnly(newStatic([]string{}).IntValue()))
	assert.Error(t, errOnly(newStatic([]string{"a", "b"}).IntValue()))
	assert.Error(t, errOnly(newStatic([]string(nil)).FloatValue()))
	assert.Error(t, errOnly(newStatic([]string{}).FloatValue()))
	assert.Error(t, errOnly(newStatic([]string{"a", "b"}).FloatValue()))
	assert.Equal(t, "", must(newStatic([]string(nil)).StringValue()))
	assert.Equal(t, "", must(newStatic([]string{}).StringValue()))
	assert.Equal(t, `a,b`, must(newStatic([]string{"a", "b"}).StringValue()))
	assert.Equal(t, map[string]interface{}{}, must(newStatic([]string(nil)).MapValue()))
	assert.Equal(t, map[string]interface{}{}, must(newStatic([]string{}).MapValue()))
	assert.Equal(t, map[string]interface{}{"0": "a", "1": "b"}, must(newStatic([]string{"a", "b"}).MapValue()))
	assert.Equal(t, []interface{}{}, must(newStatic([]string(nil)).SliceValue()))
	assert.Equal(t, []interface{}{}, must(newStatic([]string{}).SliceValue()))
	assert.Equal(t, slice("a"), must(newStatic([]string{"a"}).SliceValue()))
}

func TestStaticNone(t *testing.T) {
	// Types
	assert.Equal(t, "null", newStatic(noneValue).String())
	assert.Equal(t, false, newStatic(noneValue).IsBool())
	assert.Equal(t, true, newStatic(noneValue).IsNone())
	assert.Equal(t, false, newStatic(noneValue).IsInt())
	assert.Equal(t, false, newStatic(noneValue).IsNumber())
	assert.Equal(t, false, newStatic(noneValue).IsString())
	assert.Equal(t, false, newStatic(noneValue).IsMap())
	assert.Equal(t, false, newStatic(noneValue).IsSlice())

	// Conversion
	assert.Equal(t, false, must(newStatic(noneValue).BoolValue()))
	assert.Equal(t, int64(0), must(newStatic(noneValue).IntValue()))
	assert.Equal(t, 0.0, must(newStatic(noneValue).FloatValue()))
	assert.Equal(t, "", must(newStatic(noneValue).StringValue()))
	assert.Equal(t, map[string]interface{}(nil), must(newStatic(noneValue).MapValue()))
	assert.Equal(t, []interface{}(nil), must(newStatic(noneValue).SliceValue()))
}
