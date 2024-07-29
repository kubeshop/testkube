package action

import "github.com/kubeshop/testkube/pkg/expressions"

func simplifyExpression(expr string, machines ...expressions.Machine) string {
	v, err := expressions.EvalExpressionPartial(expr, machines...)
	if err == nil {
		return v.String()
	}
	return expr
}
