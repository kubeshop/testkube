package triggers

import (
	"fmt"
	"reflect"
	"strings"

	v1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// matchFieldSelector evaluates all field conditions against the event objects.
// Returns true if all conditions match (AND logic).
// Returns true if conditions is empty (no filtering).
func matchFieldSelector(conditions []v1.WorkflowTriggerFieldCondition, obj, oldObj any) bool {
	for _, cond := range conditions {
		if !evaluateFieldCondition(cond, obj, oldObj) {
			return false
		}
	}
	return true
}

func evaluateFieldCondition(cond v1.WorkflowTriggerFieldCondition, obj, oldObj any) bool {
	switch cond.Operator {
	case v1.FieldOperatorExists:
		return fieldPathExists(cond.Path, obj)
	case v1.FieldOperatorNotExists:
		if !fieldPathHasValidSyntax(cond.Path) {
			return false // invalid path syntax is not "field doesn't exist"
		}
		return !fieldPathExists(cond.Path, obj)
	case v1.FieldOperatorEquals:
		val, err := evaluateFieldPath(cond.Path, obj)
		return err == nil && val == cond.Value
	case v1.FieldOperatorNotEquals:
		val, err := evaluateFieldPath(cond.Path, obj)
		return err == nil && val != cond.Value
	case v1.FieldOperatorChanged:
		if oldObj == nil {
			return false
		}
		return fieldPathChanged(cond.Path, obj, oldObj)
	case v1.FieldOperatorChangedTo:
		if oldObj == nil {
			return false
		}
		newVal, newErr := evaluateFieldPath(cond.Path, obj)
		if newErr != nil {
			return false
		}
		oldVal, oldErr := evaluateFieldPath(cond.Path, oldObj)
		if oldErr != nil {
			// Field didn't exist before, now it does with the target value
			return newVal == cond.Value
		}
		return newVal != oldVal && newVal == cond.Value
	case v1.FieldOperatorChangedFrom:
		if oldObj == nil {
			return false
		}
		oldVal, oldErr := evaluateFieldPath(cond.Path, oldObj)
		if oldErr != nil || oldVal != cond.Value {
			// Old value wasn't the target — didn't change *from* it.
			return false
		}
		newVal, newErr := evaluateFieldPath(cond.Path, obj)
		if newErr != nil {
			// Field was removed on the new object. That *is* a change away from the
			// target value (including when cond.Value is "" and old held an explicit
			// empty string). Greptile flagged silent miss on this edge case.
			return true
		}
		return newVal != cond.Value
	}
	return false
}

// fieldPathChanged checks if a field value differs between old and new objects.
// Works for any type: scalars, arrays, maps. Uses deep comparison.
func fieldPathChanged(path string, newObj, oldObj any) bool {
	newResult, newErr := resolveFieldExpression(path, newObj)
	oldResult, oldErr := resolveFieldExpression(path, oldObj)

	// If both fail to resolve, no change
	if newErr != nil && oldErr != nil {
		return false
	}
	// If one resolves and the other doesn't, that's a change
	if newErr != nil || oldErr != nil {
		return true
	}

	newStatic := newResult.Static()
	oldStatic := oldResult.Static()

	// If either didn't fully resolve, can't compare
	if newStatic == nil || oldStatic == nil {
		return false
	}

	return !reflect.DeepEqual(newStatic.Value(), oldStatic.Value())
}

// fieldPathExists checks if a field path resolves on the object.
// Works for any type: scalars, arrays, maps.
// Returns false for both missing fields and invalid path syntax.
func fieldPathExists(path string, obj any) bool {
	result, err := resolveFieldExpression(path, obj)
	if err != nil {
		return false
	}
	return result.Static() != nil && !result.Static().IsNone()
}

// fieldPathHasValidSyntax checks if the path expression compiles successfully.
// Used to distinguish "field doesn't exist" from "path is invalid" for not_exists.
func fieldPathHasValidSyntax(path string) bool {
	expr := "resource" + path
	if !strings.HasPrefix(path, ".") {
		expr = "resource." + path
	}
	_, err := expressions.Compile(expr)
	return err == nil
}

// evaluateFieldPath resolves a dot-path to a scalar string value.
// Returns error if the field doesn't exist or is not a scalar (array/map).
func evaluateFieldPath(path string, obj any) (string, error) {
	result, err := resolveFieldExpression(path, obj)
	if err != nil {
		return "", err
	}

	s := result.Static()
	if s == nil {
		return "", fmt.Errorf("field path %q could not be fully resolved", path)
	}
	if s.IsNone() {
		return "", fmt.Errorf("field path %q does not exist", path)
	}
	if s.IsMap() || s.IsSlice() {
		return "", fmt.Errorf("field path %q resolves to a non-scalar type (array or map), use exists/not_exists for these fields", path)
	}

	val, err := s.StringValue()
	if err != nil {
		return "", fmt.Errorf("field path %q: %w", path, err)
	}
	return val, nil
}

// resolveFieldExpression compiles and resolves a dot-path against an object.
func resolveFieldExpression(path string, obj any) (expressions.Expression, error) {
	expr := "resource" + path
	if !strings.HasPrefix(path, ".") {
		expr = "resource." + path
	}

	compiled, err := expressions.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("compile field path %q: %w", path, err)
	}

	machine := expressions.NewMachine().Register("resource", derefPtr(obj))
	result, err := compiled.Resolve(machine)
	if err != nil {
		return nil, fmt.Errorf("resolve field path %q: %w", path, err)
	}
	return result, nil
}

// derefPtr dereferences a pointer to its underlying value.
// The expression engine's isStruct check requires a struct, not a pointer.
func derefPtr(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		return rv.Elem().Interface()
	}
	return v
}
