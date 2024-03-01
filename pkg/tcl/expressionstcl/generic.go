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

func resolve(v reflect.Value, t tagData, m []Machine, force bool, finalizer Machine) (changed bool, err error) {
	if t.value == "force" {
		force = true
	}
	if t.key == "" && t.value == "" && !force {
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
				return resolve(v.FieldByName("StrVal"), t, m, force, finalizer)
			}
		} else if t.value == "include" || force {
			tt := v.Type()
			for i := 0; i < tt.NumField(); i++ {
				f := tt.Field(i)
				tagStr := f.Tag.Get("expr")
				tag := parseTag(tagStr)
				if !f.IsExported() {
					if tagStr != "" && tagStr != "-" {
						return changed, errors.New(f.Name + ": private property marked with `expr` clause")
					}
					continue
				}
				value := v.FieldByName(f.Name)
				var ch bool
				ch, err = resolve(value, tag, m, force, finalizer)
				if ch {
					changed = true
				}
				if err != nil {
					return changed, errors.Wrap(err, f.Name)
				}
			}
		}
		return
	case reflect.Slice:
		if t.value == "" && !force {
			return changed, nil
		}
		for i := 0; i < v.Len(); i++ {
			ch, err := resolve(v.Index(i), t, m, force, finalizer)
			if ch {
				changed = true
			}
			if err != nil {
				return changed, errors.Wrap(err, fmt.Sprintf("%d", i))
			}
		}
		return
	case reflect.Map:
		if t.value == "" && t.key == "" && !force {
			return changed, nil
		}
		for _, k := range v.MapKeys() {
			if t.value != "" || force {
				// It's not possible to get a pointer to map element,
				// so we need to copy it and reassign
				item := clone(v.MapIndex(k))
				var ch bool
				ch, err = resolve(item, t, m, force, finalizer)
				if ch {
					changed = true
				}
				v.SetMapIndex(k, item)
				if err != nil {
					return changed, errors.Wrap(err, k.String())
				}
			}
			if t.key != "" || force {
				key := clone(k)
				var ch bool
				ch, err = resolve(key, tagData{value: t.key}, m, force, finalizer)
				if ch {
					changed = true
				}
				if !key.Equal(k) {
					item := clone(v.MapIndex(k))
					v.SetMapIndex(k, reflect.Value{})
					v.SetMapIndex(key, item)
				}
				if err != nil {
					return changed, errors.Wrap(err, "key("+k.String()+")")
				}
			}
		}
		return
	case reflect.String:
		if t.value == "expression" {
			var expr Expression
			str := v.String()
			expr, err = CompileAndResolve(str, m...)
			if err != nil {
				return changed, err
			}
			var vv string
			if finalizer != nil {
				expr2, err := expr.Resolve(finalizer)
				if err != nil {
					vv = expr.String()
				} else {
					vv, _ = expr2.Static().StringValue()
				}
			} else {
				vv = expr.String()
			}
			changed = vv != str
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else {
				ptr.Set(reflect.ValueOf(&vv))
			}
		} else if (t.value == "template" && !IsTemplateStringWithoutExpressions(v.String())) || force {
			var expr Expression
			str := v.String()
			expr, err = CompileAndResolveTemplate(str, m...)
			if err != nil {
				return changed, err
			}
			var vv string
			if finalizer != nil {
				expr2, err := expr.Resolve(finalizer)
				if err != nil {
					vv = expr.String()
				} else {
					vv, _ = expr2.Static().StringValue()
				}
			} else {
				vv = expr.Template()
			}
			changed = vv != str
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else {
				ptr.Set(reflect.ValueOf(&vv))
			}
		}
		return
	}

	// Ignore unrecognized values
	return
}

func simplify(t interface{}, tag tagData, finalizer Machine, m ...Machine) error {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Pointer {
		return errors.New("pointer needs to be passed to Simplify function")
	}
	changed, err := resolve(v, tag, m, false, finalizer)
	i := 1
	for changed && err == nil {
		if i > maxCallStack {
			return fmt.Errorf("maximum call stack exceeded while simplifying struct")
		}
		changed, err = resolve(v, tag, m, false, finalizer)
		i++
	}
	return err
}

func Simplify(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "include"}, nil, m...)
}

func SimplifyForce(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "force"}, nil, m...)
}

func Finalize(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "include"}, FinalizerNone, m...)
}

func FinalizeForce(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "force"}, FinalizerNone, m...)
}
