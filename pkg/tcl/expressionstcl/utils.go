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

func EvalExpression(str string, machines ...Machine) (StaticValue, error) {
	expr, err := Compile(str)
	if err != nil {
		return nil, errors.Wrap(err, "compiling")
	}
	expr, err = expr.Resolve(machines...)
	if err != nil {
		return nil, errors.Wrap(err, "resolving")
	}
	if expr.Static() == nil {
		return nil, fmt.Errorf("expression should be static: %s", expr.String())
	}
	return expr.Static(), nil
}

func Escape(str string) string {
	return NewStringValue(str).Template()
}
