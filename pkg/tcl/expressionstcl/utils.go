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
)

const maxCallStack = 10_000

func deepResolve(expr Expression, machines ...MachineCore) (Expression, error) {
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
