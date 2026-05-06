package expressions

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

var ErrWalkStop = errors.New("end walking")

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

func hasUnexportedFields(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return false
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if !t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

func clone(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.String {
		s := v.String()
		return reflect.ValueOf(&s).Elem()
	} else if v.Kind() == reflect.Struct {
		r := reflect.New(v.Type()).Elem()
		t := v.Type()
		for i := 0; i < r.NumField(); i++ {
			if t.Field(i).IsExported() {
				r.Field(i).Set(v.Field(i))
			}
		}
		return r
	} else if v.Kind() == reflect.Slice {
		r := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			r.Index(i).Set(clone(v.Index(i)))
		}
		return r
	} else if v.Kind() == reflect.Interface {
		r := reflect.New(v.Type())
		r.Elem().Set(v)
		return r.Elem()
	}
	return v
}

// getElementString extracts a string value from a reflect.Value,
// unwrapping pointers and interfaces as needed.
func getElementString(v reflect.Value) (string, bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String(), true
	}
	return "", false
}

func resolve(v reflect.Value, t tagData, m []Machine, force bool, finalize bool) (changed bool, err error) {
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
		if v.CanAddr() {
			ptr = v
		}
		v = v.Elem()
	}

	if v.IsZero() || !v.IsValid() || (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		// TODO: Cache the tags for structs for better performance
		isIntOrStringType := IsIntOrStringType(v.Interface())
		if isIntOrStringType {
			return resolve(v.FieldByName("StrVal"), t, m, force, finalize)
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
				ch, err = resolve(value, tag, m, force, finalize)
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
		// Handle potential array expansion for template strings in slices.
		// When a slice element is a pure template expression like "{{ expr }}"
		// and the expression resolves to an array, expand the array elements
		// into the parent slice individually. When the template has surrounding
		// literal text (e.g., "prefix{{ expr }}suffix") or the expression resolves
		// to a non-array value, the existing stringification behavior is preserved.
		if t.value == "template" || force {
			newItems := make([]reflect.Value, 0, v.Len())
			anyExpanded := false
			for i := 0; i < v.Len(); i++ {
				elem := v.Index(i)
				str, isStr := getElementString(elem)
				if isStr && !IsTemplateStringWithoutExpressions(str) {
					if innerExpr, isPure := ExtractPureTemplateExpression(str); isPure {
						expr, compileErr := CompileAndResolve(innerExpr, m...)
						if compileErr == nil {
							if finalize && expr.Static() == nil {
								expr, compileErr = expr.Resolve(FinalizerFail)
							}
							if compileErr == nil && expr.Static() != nil {
								// Array result: expand into parent slice
								if items, sliceErr := expr.Static().SliceValue(); sliceErr == nil {
									for _, item := range items {
										sv := NewValue(item)
										s, _ := sv.StringValue()
										newItems = append(newItems, reflect.ValueOf(s).Convert(v.Type().Elem()))
									}
									anyExpanded = true
									continue
								}
								// Non-array result: reuse the already-resolved value
								// to avoid re-compiling the same expression as a template.
								s, _ := expr.Static().StringValue()
								changed = changed || s != str
								newItems = append(newItems, reflect.ValueOf(s).Convert(v.Type().Elem()))
								continue
							}
						}
					}
				}
				// Normal resolution for this element
				elemCopy := clone(elem)
				ch, err := resolve(elemCopy, t, m, force, finalize)
				if ch {
					changed = true
				}
				if err != nil {
					return changed, errors.Wrap(err, fmt.Sprintf("%d", i))
				}
				newItems = append(newItems, elemCopy)
			}
			if anyExpanded {
				changed = true
				newSlice := reflect.MakeSlice(v.Type(), len(newItems), len(newItems))
				for i, item := range newItems {
					newSlice.Index(i).Set(item)
				}
				if v.CanSet() {
					v.Set(newSlice)
				} else if ptr.Kind() == reflect.Interface {
					ptr.Set(newSlice)
				}
			} else if changed {
				newSlice := reflect.MakeSlice(v.Type(), len(newItems), len(newItems))
				for i, item := range newItems {
					newSlice.Index(i).Set(item)
				}
				if v.CanSet() {
					v.Set(newSlice)
				} else if ptr.Kind() == reflect.Interface {
					ptr.Set(newSlice)
				} else {
					// Fallback: write resolved elements in-place (always safe for slice elements)
					for i, item := range newItems {
						v.Index(i).Set(item)
					}
				}
			}
			return
		}
		for i := 0; i < v.Len(); i++ {
			ch, err := resolve(v.Index(i), t, m, force, finalize)
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
			if (t.value != "" || force) && !hasUnexportedFields(v.MapIndex(k)) {
				// It's not possible to get a pointer to map element,
				// so we need to copy it and reassign
				item := clone(v.MapIndex(k))
				var ch bool
				ch, err = resolve(item, t, m, force, finalize)
				if ch {
					changed = true
				}
				if err != nil {
					return changed, errors.Wrap(err, k.String())
				}
				v.SetMapIndex(k, item)
			}
			if (t.key != "" || force) && !hasUnexportedFields(k) && !hasUnexportedFields(v.MapIndex(k)) {
				key := clone(k)
				var ch bool
				ch, err = resolve(key, tagData{value: t.key}, m, force, finalize)
				if ch {
					changed = true
				}
				if err != nil {
					return changed, errors.Wrap(err, "key("+k.String()+")")
				}
				if !key.Equal(k) {
					item := clone(v.MapIndex(k))
					v.SetMapIndex(k, reflect.Value{})
					v.SetMapIndex(key.Convert(k.Type()), item)
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
			if finalize {
				expr2, err := expr.Resolve(FinalizerFail)
				if err != nil {
					return changed, errors.Wrap(err, "resolving the value")
				}
				vv, _ = expr2.Static().StringValue()
			} else {
				vv = expr.String()
			}
			changed = vv != str
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else if ptr.Kind() == reflect.Interface {
				ptr.Set(reflect.ValueOf(vv))
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
			if finalize {
				expr2, err := expr.Resolve(FinalizerFail)
				if err != nil {
					return changed, errors.Wrap(err, "resolving the value")
				}
				vv, _ = expr2.Static().StringValue()
			} else {
				vv = expr.Template()
			}
			changed = vv != str
			if ptr.Kind() == reflect.String {
				v.SetString(vv)
			} else if ptr.Kind() == reflect.Interface {
				ptr.Set(reflect.ValueOf(vv))
			} else {
				instance := reflect.New(v.Type())
				instance.Elem().SetString(vv)
				ptr.Set(instance)
			}
		}
		return
	}

	// Ignore unrecognized values
	return
}

func simplify(t interface{}, tag tagData, m ...Machine) error {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Pointer {
		return errors.New("pointer needs to be passed to Simplify function")
	}
	changed, err := resolve(v, tag, m, false, false)
	i := 1
	for changed && err == nil {
		if i > maxCallStack {
			return fmt.Errorf("maximum call stack exceeded while simplifying struct")
		}
		changed, err = resolve(v, tag, m, false, false)
		i++
	}
	return err
}

func finalize(t interface{}, tag tagData, m ...Machine) error {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Pointer {
		return errors.New("pointer needs to be passed to Finalize function")
	}
	_, err := resolve(v, tag, m, false, true)
	return err
}

func Simplify(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "include"}, m...)
}

func WalkVariables(t interface{}, variableFn func(name string) error) error {
	m := NewMachine().RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		err := variableFn(name)
		return nil, err != nil, err
	})
	err := Simplify(t, m)
	if errors.Is(err, ErrWalkStop) {
		return nil
	}
	return err
}

func SimplifyForce(t interface{}, m ...Machine) error {
	return simplify(t, tagData{value: "force"}, m...)
}

func Finalize(t interface{}, m ...Machine) error {
	return finalize(t, tagData{value: "include"}, m...)
}

func FinalizeForce(t interface{}, m ...Machine) error {
	return finalize(t, tagData{value: "force"}, m...)
}
