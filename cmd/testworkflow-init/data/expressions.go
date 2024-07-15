package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/expressions"
)

var aliases = map[string]string{
	"always": `true`,
	"never":  `false`,

	"error":   `failed`,
	"success": `passed`,

	"self.error":   `self.failed`,
	"self.success": `self.passed`,

	"passed": `status == "passed"`,
	"failed": `status != "passed" && status != "skipped"`,

	"self.passed": `self.status == "passed"`,
	"self.failed": `self.status != "passed" && self.status != "skipped"`,
}

var LocalMachine = expressions.NewMachine().
	Register("status", expressions.MustCompile("self.status"))

var RefMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name == "_ref" {
			return Step.Ref, true
		}
		return nil, false
	})

var AliasMachine = expressions.NewMachine().
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		alias, ok := aliases[name]
		if !ok {
			return nil, false, nil
		}
		expr, err := expressions.Compile(alias)
		if err != nil {
			return expr, false, err
		}
		expr, err = expr.Resolve(RefMachine)
		return expr, true, err
	})

var StateMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name == "status" {
			currentStatus := GetState().CurrentStatus
			expr, err := expressions.EvalExpression(currentStatus, RefStatusMachine, AliasMachine)
			if err != nil {
				panic(fmt.Sprintf("current status is invalid: %s: %v", currentStatus, err.Error()))
			}
			if passed, _ := expr.BoolValue(); passed {
				return StepStatusPassed, true
			}
			return StepStatusFailed, true
		} else if name == "self.status" {
			step := GetState().GetStep(name)
			if step.Status == nil {
				return nil, false
			}
			return *step.Status, true
		}
		return nil, false
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, "output.") {
			// TODO TODO TODO TODO
			return GetState().GetOutput(name[7:])
		}
		return nil, false, nil
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, "services.") {
			// TODO TODO TODO TODO
			return GetState().GetOutput(name)
		}
		return nil, false, nil
	})

var EnvMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if strings.HasPrefix(name, "env.") {
			return os.Getenv(name[4:]), true
		}
		return nil, false
	}).
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name != "env" {
			return nil, false
		}
		env := make(map[string]string)
		for _, item := range os.Environ() {
			key, value, _ := strings.Cut(item, "=")
			env[key] = value
		}
		return env, true
	})

var RefSuccessMachine = expressions.NewMachine().
	RegisterAccessor(func(ref string) (interface{}, bool) {
		s := GetState().GetStep(ref)
		if s.Status == nil {
			return nil, false
		}
		return *s.Status == StepStatusPassed || *s.Status == StepStatusSkipped, true
	})

var RefStatusMachine = expressions.NewMachine().
	RegisterAccessor(func(ref string) (interface{}, bool) {
		status := GetState().GetStep(ref).Status
		if status == nil {
			return nil, false
		}
		return string(*status), true
	})

func Template(tpl string, m ...expressions.Machine) (string, error) {
	m = append(m, AliasMachine, GetBaseTestWorkflowMachine())
	return expressions.EvalTemplate(tpl, m...)
}

func Expression(expr string, m ...expressions.Machine) (expressions.StaticValue, error) {
	m = append(m, AliasMachine, GetBaseTestWorkflowMachine())
	return expressions.EvalExpression(expr, m...)
}
