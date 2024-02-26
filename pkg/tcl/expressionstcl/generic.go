// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type tagData struct {
	key   string
	value string
}

func parseTag(tag string) tagData {
	s := strings.Split(tag, ",")
	if len(s) > 1 {
		return tagData{key: s[0], value: s[1]}
	}
	return tagData{value: s[0]}
}

var unrecognizedErr = errors.New("unsupported value passed for resolving expressions")

func clone(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.String {
		s := v.String()
		return reflect.ValueOf(&s).Elem()
	} else if v.Kind() == reflect.Struct {
		r := reflect.New(v.Type()).Elem()
		for i := 0; i < r.NumField(); i++ {
			r.Field(i).Set(v.Field(i))
		}
		return r
	}
	return v
}

func resolve(v reflect.Value, t tagData, m []MachineCore) (err error) {
	if t.key == "" && t.value == "" {
		return
	}

	ptr := v
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		ptr = v
		v = v.Elem()
	}

	if v.IsZero() || !v.IsValid() || (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		// TODO: Cache the tags for structs for better performance
		vv, ok := v.Interface().(intstr.IntOrString)
		if ok {
			if vv.Type == intstr.String {
				return resolve(v.FieldByName("StrVal"), t, m)
			}
		} else if t.value == "include" {
			tt := v.Type()
			for i := 0; i < tt.NumField(); i++ {
				f := tt.Field(i)
				tag := parseTag(f.Tag.Get("expr"))
				value := v.FieldByName(f.Name)
				err = resolve(value, tag, m)
				if err != nil {
					return errors.Wrap(err, f.Name)
				}
			}
		}
		return
	case reflect.Slice:
		if t.value == "" {
			return nil
		}
		for i := 0; i < v.Len(); i++ {
			err := resolve(v.Index(i), t, m)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("%d", i))
			}
		}
		return
	case reflect.Map:
		if t.value == "" && t.key == "" {
			return nil
		}
		for _, k := range v.MapKeys() {
			if t.value != "" {
				// It's not possible to get a pointer to map element,
				// so we need to copy it and reassign
				item := clone(v.MapIndex(k))
				err = resolve(item, t, m)
				v.SetMapIndex(k, item)
				if err != nil {
					return errors.Wrap(err, k.String())
				}
			}
			if t.key != "" {
				key := clone(k)
				err = resolve(key, tagData{value: t.key}, m)
				if !key.Equal(k) {
					item := clone(v.MapIndex(k))
					v.SetMapIndex(k, reflect.Value{})
					v.SetMapIndex(key, item)
				}
				if err != nil {
					return errors.Wrap(err, "key("+k.String()+")")
				}
			}
		}
		return
	case reflect.String:
		if t.value == "expression" {
			var expr Expression
			expr, err = CompileAndResolve(v.String(), m...)
			if err != nil {
				return err
			}
			vv := expr.String()
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else {
				ptr.Set(reflect.ValueOf(&vv))
			}
		} else if t.value == "template" && !IsTemplateStringWithoutExpressions(v.String()) {
			var expr Expression
			expr, err = CompileAndResolveTemplate(v.String(), m...)
			if err != nil {
				return err
			}
			vv := expr.Template()
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else {
				ptr.Set(reflect.ValueOf(&vv))
			}
		}
		return
	}

	// Fail for unrecognized values
	return unrecognizedErr
}

func SimplifyStruct(t interface{}, m ...MachineCore) error {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Pointer {
		return errors.New("pointer needs to be passed to Resolve function")
	}
	return resolve(v, tagData{value: "include"}, m)
}
