package expressions

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const maxCallStack = 10_000

func deepResolve(expr Expression, machines ...Machine) (Expression, error) {
	i := 1
	expr, changed, err := expr.SafeResolve(machines...)
	for changed && err == nil && expr.Static() == nil {
		if i > maxCallStack {
			return expr, fmt.Errorf("maximum call stack exceeded while resolving expression: %s", expr.String())
		}
		expr, changed, err = expr.SafeResolve(machines...)
		i++
	}
	return expr, err
}

func EvalTemplate(tpl string, machines ...Machine) (string, error) {
	expr, err := CompileTemplate(tpl)
	if err != nil {
		return "", errors.Wrap(err, "compiling")
	}
	expr, err = expr.Resolve(machines...)
	if err != nil {
		return "", errors.Wrap(err, "resolving")
	}
	if expr.Static() == nil {
		return "", fmt.Errorf("template should be static: %s", expr.Template())
	}
	return expr.Static().StringValue()
}

func EvalExpressionPartial(str string, machines ...Machine) (Expression, error) {
	expr, err := Compile(str)
	if err != nil {
		return nil, errors.Wrap(err, "compiling")
	}
	expr, err = expr.Resolve(machines...)
	if err != nil {
		return nil, errors.Wrap(err, "resolving")
	}
	return expr, err
}

func EvalBoolean(str string, machines ...Machine) (bool, bool, error) {
	// Fast-track
	if str == "" {
		return false, true, nil
	}

	// Compute
	expr, err := EvalExpressionPartial(str, machines...)
	if err != nil || expr.Static() == nil {
		return false, false, err
	}
	v, err := expr.Static().BoolValue()
	if err != nil {
		return false, false, err
	}
	return v, true, nil
}

func EvalExpression(str string, machines ...Machine) (StaticValue, error) {
	expr, err := EvalExpressionPartial(str, machines...)
	if err != nil {
		return nil, err
	}
	if expr.Static() == nil {
		return nil, fmt.Errorf("expression should be static: %s", expr.String())
	}
	return expr.Static(), nil
}

func Escape(str string) string {
	return NewStringValue(str).Template()
}

func EscapeLabelKeyForVarName(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), "-", "_"), "/", "_")
}

func MustCall(m Machine, name string, args ...interface{}) interface{} {
	list := make([]StaticValue, len(args))
	for i, v := range args {
		if vv, ok := v.(StaticValue); ok {
			list[i] = vv
		} else {
			list[i] = NewValue(v)
		}
	}
	v, ok, err := m.Call(name, list...)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("not recognized")
	}
	return v.Static().Value()
}
