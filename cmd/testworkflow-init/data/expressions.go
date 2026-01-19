package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	OutputKey      = "output"
	OutputPrefix   = OutputKey + "."
	ServicesKey    = "services"
	ServicesPrefix = ServicesKey + "."
	EnvKey         = "env"
	EnvPrefix      = EnvKey + "."
	RefKey         = "_ref"
	StatusKey      = "status"
)

var aliases = map[string]string{
	"always": `true`,
	"never":  `false`,

	"error":   `failed`,
	"success": `passed`,

	"self.error":   `self.failed`,
	"self.success": `self.passed`,

	"passed": fmt.Sprintf(`%s == "passed"`, StatusKey),
	"failed": fmt.Sprintf(`%s != "passed" && %s != "skipped"`, StatusKey, StatusKey),

	"self.passed": `self.status == "passed"`,
	"self.failed": `self.status != "passed" && self.status != "skipped"`,
}

var LocalMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name == StatusKey {
			state := GetState()
			step := state.GetStep(state.CurrentRef)
			if step.Status == nil {
				return nil, false
			}
			return string(*step.Status), true
		}
		return nil, false
	})

var RefMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name == RefKey {
			return GetState().CurrentRef, true
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
		switch name {
		case "status":
			currentStatus := GetState().CurrentStatus
			expr, err := expressions.EvalExpression(currentStatus, RefNotFailedMachine, AliasMachine)
			if err != nil {
				output.ExitErrorf(constants.CodeInternal, "current status is invalid: %s: %v\n", currentStatus, err.Error())
			}
			if passed, _ := expr.BoolValue(); passed {
				return string(constants.StepStatusPassed), true
			}
			return string(constants.StepStatusFailed), true
		case "self.status":
			state := GetState()
			step := state.GetStep(state.CurrentRef)
			if step.Status == nil {
				return nil, false
			}
			return string(*step.Status), true
		}
		return nil, false
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, OutputPrefix) {
			return GetState().GetOutput(name[len(OutputPrefix):])
		}
		return nil, false, nil
	}).
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if strings.HasPrefix(name, ServicesPrefix) {
			// TODO TODO TODO TODO
			return GetState().GetOutput(name)
		}
		return nil, false, nil
	})

var EnvMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if strings.HasPrefix(name, EnvPrefix) {
			return os.Getenv(name[len(EnvPrefix):]), true
		}
		return nil, false
	}).
	RegisterAccessor(func(name string) (interface{}, bool) {
		if name != EnvKey {
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
		return *s.Status == constants.StepStatusPassed || *s.Status == constants.StepStatusSkipped, true
	})

var RefNotFailedMachine = expressions.NewMachine().
	RegisterAccessor(func(ref string) (interface{}, bool) {
		s := GetState().GetStep(ref)
		if s.Status == nil && s.Result != "" {
			exp, err := expressions.Compile(s.Result)
			if err == nil {
				return exp, true
			}
		}
		return s.Status == nil || *s.Status == constants.StepStatusPassed || *s.Status == constants.StepStatusSkipped, true
	})

func Expression(expr string, m ...expressions.Machine) (expressions.StaticValue, error) {
	m = append(m, AliasMachine, GetBaseTestWorkflowMachine(), ExecutionMachine())
	return expressions.EvalExpression(expr, m...)
}
