package data

import (
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

	"passed": `!status`,
	"failed": `bool(status) && status != "skipped"`,

	"self.passed": `!self.status`,
	"self.failed": `bool(self.status) && self.status != "skipped"`,
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
		// TODO TODO TODO TODO
		if name == "status" {
			return State.GetStatus(), true
		} else if name == "self.status" {
			return State.GetSelfStatus(), true
		}
		return nil, false
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, "output.") {
			// TODO TODO TODO TODO
			return State.GetOutput(name[7:])
		}
		return nil, false, nil
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, "services.") {
			// TODO TODO TODO TODO
			return State.GetOutput(name)
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
		s := State.GetStep(ref)
		return s.Status == StepStatusPassed || s.Status == StepStatusSkipped, s.HasStatus
	})

var RefStatusMachine = expressions.NewMachine().
	RegisterAccessor(func(ref string) (interface{}, bool) {
		return string(State.GetStep(ref).Status), true
	})

func Template(tpl string, m ...expressions.Machine) (string, error) {
	m = append(m, AliasMachine, GetBaseTestWorkflowMachine())
	return expressions.EvalTemplate(tpl, m...)
}

func Expression(expr string, m ...expressions.Machine) (expressions.StaticValue, error) {
	m = append(m, AliasMachine, GetBaseTestWorkflowMachine())
	return expressions.EvalExpression(expr, m...)
}

func RefSuccessExpression(expr string) (expressions.StaticValue, error) {
	return expressions.EvalExpression(expr, RefSuccessMachine)
}

func RefStatusExpression(expr string) (expressions.StaticValue, error) {
	return expressions.EvalExpression(expr, RefStatusMachine)
}
