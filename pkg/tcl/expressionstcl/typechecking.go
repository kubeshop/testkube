// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import "reflect"

type noneType struct{}

var noneValue noneType

func isInt(s interface{}) bool {
	switch s.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return s.(float32) == float32(int32(s.(float32)))
	case float64:
		return s.(float64) == float64(int64(s.(float64)))
	}
	return false
}

func isString(s interface{}) bool {
	_, ok := s.(string)
	return ok
}

func isBool(s interface{}) bool {
	_, ok := s.(bool)
	return ok
}

func isNone(s interface{}) bool {
	_, ok := s.(noneType)
	return ok
}

func isNumber(s interface{}) bool {
	switch s.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}

func isMap(s interface{}) bool {
	return reflect.ValueOf(s).Kind() == reflect.Map
}

func isSlice(s interface{}) bool {
	return reflect.ValueOf(s).Kind() == reflect.Slice
}
