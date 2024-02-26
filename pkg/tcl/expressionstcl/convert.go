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
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func toString(s interface{}) (string, error) {
	// Fast track
	v, ok := s.(string)
	if ok {
		return v, nil
	}
	if isNone(s) {
		return "", nil
	}
	// Convert
	if isNumber(s) {
		return fmt.Sprintf("%v", s), nil
	}
	if isSlice(s) {
		var err error
		value := reflect.ValueOf(s)
		results := make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			results[i], err = toString(value.Index(i).Interface())
			if err != nil {
				err = fmt.Errorf("error while converting '%v' slice item: %v", value.Index(i), err)
				return "", err
			}
		}
		return strings.Join(results, ","), nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("error while converting '%v' map to JSON: %v", s, err)
	}
	r := string(b)
	if isMap(s) && r == "null" {
		return "{}", nil
	}
	return r, nil
}

func toFloat(s interface{}) (float64, error) {
	// Fast track
	if v, ok := s.(float64); ok {
		return v, nil
	}
	if isNone(s) {
		return 0, nil
	}
	// Convert
	str, err := toString(s)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("error while converting value to number: %v: %v", s, err)
	}
	return v, nil
}

func toInt(s interface{}) (int64, error) {
	// Fast track
	if v, ok := s.(int64); ok {
		return v, nil
	}
	if v, ok := s.(int); ok {
		return int64(v), nil
	}
	if isNone(s) {
		return 0, nil
	}
	// Convert
	v, err := toFloat(s)
	return int64(v), err
}

func toBool(s interface{}) (bool, error) {
	// Fast track
	if v, ok := s.(bool); ok {
		return v, nil
	}
	if isNone(s) {
		return false, nil
	}
	if isMap(s) || isSlice(s) {
		return reflect.ValueOf(s).Len() > 0, nil
	}
	// Convert
	value, err := toString(s)
	if err != nil {
		return false, fmt.Errorf("error while converting value to bool: %v: %v", value, err)
	}
	return !(value == "" || value == "false" || value == "0" || value == "off"), nil
}

func toMap(s interface{}) (map[string]interface{}, error) {
	// Fast track
	if v, ok := s.(map[string]interface{}); ok {
		return v, nil
	}
	if isNone(s) {
		return nil, nil
	}
	// Convert
	if isMap(s) {
		value := reflect.ValueOf(s)
		res := make(map[string]interface{}, value.Len())
		for _, k := range value.MapKeys() {
			kk, err := toString(k.Interface())
			if err != nil {
				return nil, fmt.Errorf("error while converting map key to string: %v: %v", k, err)
			}
			res[kk] = value.MapIndex(k).Interface()
		}
		return res, nil
	}
	if isSlice(s) {
		value := reflect.ValueOf(s)
		res := make(map[string]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			res[strconv.Itoa(i)] = value.Index(i).Interface()
		}
		return res, nil
	}
	return nil, fmt.Errorf("error while converting value to map: %v", s)
}

func toSlice(s interface{}) ([]interface{}, error) {
	// Fast track
	if v, ok := s.([]interface{}); ok {
		return v, nil
	}
	if isNone(s) {
		return nil, nil
	}
	// Convert
	if isSlice(s) {
		value := reflect.ValueOf(s)
		res := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			res[i] = value.Index(i).Interface()
		}
		return res, nil
	}
	if isMap(s) {
		return nil, fmt.Errorf("error while converting map to slice: %v", s)
	}
	return nil, fmt.Errorf("error while converting value to slice: %v", s)
}
