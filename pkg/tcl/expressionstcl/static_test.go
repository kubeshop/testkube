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
	assert.Equal(t, "false", NewValue(false).String())
	assert.Equal(t, "true", NewValue(true).String())
	assert.Equal(t, true, NewValue(false).IsBool())
	assert.Equal(t, true, NewValue(true).IsBool())
	assert.Equal(t, false, NewValue(false).IsNone())
	assert.Equal(t, false, NewValue(true).IsNone())
	assert.Equal(t, false, NewValue(true).IsInt())
	assert.Equal(t, false, NewValue(true).IsNumber())
	assert.Equal(t, false, NewValue(true).IsString())
	assert.Equal(t, false, NewValue(true).IsMap())
	assert.Equal(t, false, NewValue(true).IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue(false).BoolValue()))
	assert.Equal(t, true, must(NewValue(true).BoolValue()))
	assert.Error(t, errOnly(NewValue(true).IntValue()))
	assert.Error(t, errOnly(NewValue(true).FloatValue()))
	assert.Equal(t, "true", must(NewValue(true).StringValue()))
	assert.Equal(t, "false", must(NewValue(false).StringValue()))
	assert.Error(t, errOnly(NewValue(true).MapValue()))
	assert.Error(t, errOnly(NewValue(true).SliceValue()))
}

func TestStaticInt(t *testing.T) {
	// Types
	assert.Equal(t, "0", NewValue(0).String())
	assert.Equal(t, "1", NewValue(1).String())
	assert.Equal(t, false, NewValue(0).IsBool())
	assert.Equal(t, false, NewValue(1).IsBool())
	assert.Equal(t, false, NewValue(0).IsNone())
	assert.Equal(t, false, NewValue(1).IsNone())
	assert.Equal(t, true, NewValue(1).IsInt())
	assert.Equal(t, true, NewValue(1).IsNumber())
	assert.Equal(t, false, NewValue(1).IsString())
	assert.Equal(t, false, NewValue(1).IsMap())
	assert.Equal(t, false, NewValue(1).IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue(0).BoolValue()))
	assert.Equal(t, true, must(NewValue(1).BoolValue()))
	assert.Equal(t, int64(1), must(NewValue(1).IntValue()))
	assert.Equal(t, 1.0, must(NewValue(1).FloatValue()))
	assert.Equal(t, "1", must(NewValue(1).StringValue()))
	assert.Error(t, errOnly(NewValue(1).MapValue()))
	assert.Error(t, errOnly(NewValue(1).SliceValue()))
}

func TestStaticFloat(t *testing.T) {
	// Types
	assert.Equal(t, "0", NewValue(0.0).String())
	assert.Equal(t, "1.5", NewValue(1.5).String())
	assert.Equal(t, false, NewValue(0.0).IsBool())
	assert.Equal(t, false, NewValue(1.0).IsBool())
	assert.Equal(t, false, NewValue(1.5).IsBool())
	assert.Equal(t, false, NewValue(0.0).IsNone())
	assert.Equal(t, false, NewValue(1.0).IsNone())
	assert.Equal(t, false, NewValue(1.5).IsNone())
	assert.Equal(t, true, NewValue(1.0).IsInt())
	assert.Equal(t, false, NewValue(1.8).IsInt())
	assert.Equal(t, true, NewValue(1.5).IsNumber())
	assert.Equal(t, false, NewValue(1.7).IsString())
	assert.Equal(t, false, NewValue(1.7).IsMap())
	assert.Equal(t, false, NewValue(1.3).IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue(0.0).BoolValue()))
	assert.Equal(t, true, must(NewValue(0.5).BoolValue()))
	assert.Equal(t, true, must(NewValue(1.0).BoolValue()))
	assert.Equal(t, true, must(NewValue(1.5).BoolValue()))
	assert.Equal(t, int64(1), must(NewValue(1.8).IntValue()))
	assert.Equal(t, 1.8, must(NewValue(1.8).FloatValue()))
	assert.Equal(t, "1.877778", must(NewValue(1.877778).StringValue()))
	assert.Equal(t, "1.88", must(NewValue(1.88).StringValue()))
	assert.Error(t, errOnly(NewValue(1.8).MapValue()))
	assert.Error(t, errOnly(NewValue(1.8).SliceValue()))
}

func TestStaticString(t *testing.T) {
	// Types
	assert.Equal(t, `""`, NewValue("").String())
	assert.Equal(t, `"value"`, NewValue("value").String())
	assert.Equal(t, `"v\"alue"`, NewValue("v\"alue").String())
	assert.Equal(t, false, NewValue("").IsBool())
	assert.Equal(t, false, NewValue("value").IsBool())
	assert.Equal(t, false, NewValue("").IsNone())
	assert.Equal(t, false, NewValue("value").IsNone())
	assert.Equal(t, false, NewValue("5").IsInt())
	assert.Equal(t, false, NewValue("value").IsInt())
	assert.Equal(t, false, NewValue("5").IsNumber())
	assert.Equal(t, false, NewValue("value").IsNumber())
	assert.Equal(t, true, NewValue("").IsString())
	assert.Equal(t, true, NewValue("value").IsString())
	assert.Equal(t, false, NewValue("value").IsMap())
	assert.Equal(t, false, NewValue("value").IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue("").BoolValue()))
	assert.Equal(t, false, must(NewValue("0").BoolValue()))
	assert.Equal(t, false, must(NewValue("off").BoolValue()))
	assert.Equal(t, false, must(NewValue("false").BoolValue()))
	assert.Equal(t, true, must(NewValue("False").BoolValue()))
	assert.Equal(t, true, must(NewValue("true").BoolValue()))
	assert.Equal(t, true, must(NewValue("on").BoolValue()))
	assert.Equal(t, true, must(NewValue("1").BoolValue()))
	assert.Equal(t, true, must(NewValue("something").BoolValue()))
	assert.Equal(t, int64(1), must(NewValue("1").IntValue()))
	assert.Equal(t, int64(1), must(NewValue("1.5").IntValue()))
	assert.Error(t, errOnly(NewValue("").IntValue()))
	assert.Error(t, errOnly(NewValue("5 apples").IntValue()))
	assert.Equal(t, 1.0, must(NewValue("1").FloatValue()))
	assert.Equal(t, 1.5, must(NewValue("1.5").FloatValue()))
	assert.Error(t, errOnly(NewValue("").FloatValue()))
	assert.Error(t, errOnly(NewValue("5 apples").FloatValue()))
	assert.Equal(t, "", must(NewValue("").StringValue()))
	assert.Equal(t, "value", must(NewValue("value").StringValue()))
	assert.Equal(t, `v"alu\e`, must(NewValue(`v"alu\e`).StringValue()))
	assert.Error(t, errOnly(NewValue("").MapValue()))
	assert.Error(t, errOnly(NewValue("v").MapValue()))
	assert.Error(t, errOnly(NewValue("").SliceValue()))
	assert.Error(t, errOnly(NewValue("v").SliceValue()))
}

func TestStaticMap(t *testing.T) {
	// Types
	assert.Equal(t, "{}", NewValue(map[string]interface{}(nil)).String())
	assert.Equal(t, "{}", NewValue(map[string]string(nil)).String())
	assert.Equal(t, `{"a":"b"}`, NewValue(map[string]string{"a": "b"}).String())
	assert.Equal(t, `{"3":"b"}`, NewValue(map[int]string{3: "b"}).String())
	assert.Equal(t, false, NewValue(map[string]interface{}(nil)).IsBool())
	assert.Equal(t, false, NewValue(map[string]interface{}{}).IsBool())
	assert.Equal(t, false, NewValue(map[string]interface{}{"a": "b"}).IsBool())
	assert.Equal(t, false, NewValue(map[string]interface{}(nil)).IsNone())
	assert.Equal(t, false, NewValue(map[string]interface{}{}).IsNone())
	assert.Equal(t, false, NewValue(map[string]interface{}{"a": "b"}).IsNone())
	assert.Equal(t, false, NewValue(map[int]interface{}{3: "3"}).IsInt())
	assert.Equal(t, false, NewValue(map[int]interface{}{3: "3"}).IsNumber())
	assert.Equal(t, false, NewValue(map[int]interface{}{3: "3"}).IsString())
	assert.Equal(t, true, NewValue(map[string]interface{}(nil)).IsMap())
	assert.Equal(t, true, NewValue(map[string]interface{}{}).IsMap())
	assert.Equal(t, true, NewValue(map[string]interface{}{"a": "b"}).IsMap())
	assert.Equal(t, false, NewValue(map[string]interface{}(nil)).IsSlice())
	assert.Equal(t, false, NewValue(map[string]interface{}{}).IsSlice())
	assert.Equal(t, false, NewValue(map[string]interface{}{"a": "b"}).IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue(map[string]string{}).BoolValue()))
	assert.Equal(t, false, must(NewValue(map[string]string(nil)).BoolValue()))
	assert.Equal(t, true, must(NewValue(map[string]string{"a": "b"}).BoolValue()))
	assert.Error(t, errOnly(NewValue(map[string]string(nil)).IntValue()))
	assert.Error(t, errOnly(NewValue(map[string]string{}).IntValue()))
	assert.Error(t, errOnly(NewValue(map[string]string{"a": "b"}).IntValue()))
	assert.Error(t, errOnly(NewValue(map[string]string(nil)).FloatValue()))
	assert.Error(t, errOnly(NewValue(map[string]string{}).FloatValue()))
	assert.Error(t, errOnly(NewValue(map[string]string{"a": "b"}).FloatValue()))
	assert.Equal(t, "{}", must(NewValue(map[string]string(nil)).StringValue()))
	assert.Equal(t, "{}", must(NewValue(map[string]string{}).StringValue()))
	assert.Equal(t, `{"a":"b"}`, must(NewValue(map[string]string{"a": "b"}).StringValue()))
	assert.Equal(t, map[string]interface{}{}, must(NewValue(map[string]string(nil)).MapValue()))
	assert.Equal(t, map[string]interface{}{}, must(NewValue(map[string]string{}).MapValue()))
	assert.Equal(t, map[string]interface{}{"a": "b"}, must(NewValue(map[string]string{"a": "b"}).MapValue()))
	assert.Error(t, errOnly(NewValue(map[string]string(nil)).SliceValue()))
	assert.Error(t, errOnly(NewValue(map[int]string{}).SliceValue()))
	assert.Error(t, errOnly(NewValue(map[int]string{3: "a"}).SliceValue()))
}

func TestStaticSlice(t *testing.T) {
	// Types
	assert.Equal(t, "[]", NewValue([]interface{}(nil)).String())
	assert.Equal(t, "[]", NewValue([]string(nil)).String())
	assert.Equal(t, `["a","b"]`, NewValue([]string{"a", "b"}).String())
	assert.Equal(t, `[3]`, NewValue([]int{3}).String())
	assert.Equal(t, false, NewValue([]interface{}(nil)).IsBool())
	assert.Equal(t, false, NewValue([]interface{}{}).IsBool())
	assert.Equal(t, false, NewValue([]interface{}{"a", "b"}).IsBool())
	assert.Equal(t, false, NewValue([]interface{}(nil)).IsNone())
	assert.Equal(t, false, NewValue([]interface{}{}).IsNone())
	assert.Equal(t, false, NewValue([]interface{}{"a", "b"}).IsNone())
	assert.Equal(t, false, NewValue([]interface{}{3: "3"}).IsInt())
	assert.Equal(t, false, NewValue([]interface{}{3: "3"}).IsNumber())
	assert.Equal(t, false, NewValue([]interface{}{3: "3"}).IsString())
	assert.Equal(t, false, NewValue([]interface{}(nil)).IsMap())
	assert.Equal(t, false, NewValue([]interface{}{}).IsMap())
	assert.Equal(t, false, NewValue([]interface{}{"a", "b"}).IsMap())
	assert.Equal(t, true, NewValue([]interface{}(nil)).IsSlice())
	assert.Equal(t, true, NewValue([]interface{}{}).IsSlice())
	assert.Equal(t, true, NewValue([]interface{}{"a", "b"}).IsSlice())

	// Conversion
	assert.Equal(t, false, must(NewValue([]string{}).BoolValue()))
	assert.Equal(t, false, must(NewValue([]string(nil)).BoolValue()))
	assert.Equal(t, true, must(NewValue([]string{"a", "b"}).BoolValue()))
	assert.Error(t, errOnly(NewValue([]string(nil)).IntValue()))
	assert.Error(t, errOnly(NewValue([]string{}).IntValue()))
	assert.Error(t, errOnly(NewValue([]string{"a", "b"}).IntValue()))
	assert.Error(t, errOnly(NewValue([]string(nil)).FloatValue()))
	assert.Error(t, errOnly(NewValue([]string{}).FloatValue()))
	assert.Error(t, errOnly(NewValue([]string{"a", "b"}).FloatValue()))
	assert.Equal(t, "", must(NewValue([]string(nil)).StringValue()))
	assert.Equal(t, "", must(NewValue([]string{}).StringValue()))
	assert.Equal(t, `a,b`, must(NewValue([]string{"a", "b"}).StringValue()))
	assert.Equal(t, map[string]interface{}{}, must(NewValue([]string(nil)).MapValue()))
	assert.Equal(t, map[string]interface{}{}, must(NewValue([]string{}).MapValue()))
	assert.Equal(t, map[string]interface{}{"0": "a", "1": "b"}, must(NewValue([]string{"a", "b"}).MapValue()))
	assert.Equal(t, []interface{}{}, must(NewValue([]string(nil)).SliceValue()))
	assert.Equal(t, []interface{}{}, must(NewValue([]string{}).SliceValue()))
	assert.Equal(t, slice("a"), must(NewValue([]string{"a"}).SliceValue()))
}

func TestStaticNone(t *testing.T) {
	// Types
	assert.Equal(t, "null", None.String())
	assert.Equal(t, false, None.IsBool())
	assert.Equal(t, true, None.IsNone())
	assert.Equal(t, false, None.IsInt())
	assert.Equal(t, false, None.IsNumber())
	assert.Equal(t, false, None.IsString())
	assert.Equal(t, false, None.IsMap())
	assert.Equal(t, false, None.IsSlice())

	// Conversion
	assert.Equal(t, false, must(None.BoolValue()))
	assert.Equal(t, int64(0), must(None.IntValue()))
	assert.Equal(t, 0.0, must(None.FloatValue()))
	assert.Equal(t, "", must(None.StringValue()))
	assert.Equal(t, map[string]interface{}(nil), must(None.MapValue()))
	assert.Equal(t, []interface{}(nil), must(None.SliceValue()))
}
