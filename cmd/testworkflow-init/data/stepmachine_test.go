package data

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func setupTestState(steps map[string]*StepData, currentRef string) {
	ClearState()
	state := GetState()
	state.CurrentRef = currentRef
	for ref, step := range steps {
		step.Ref = ref
		state.Steps[ref] = step
	}
}

func resolveStepExpr(t *testing.T, expr string) (string, bool) {
	t.Helper()
	compiled, err := expressions.CompileAndResolve(expr, StepMachine)
	if err != nil {
		return "", false
	}
	if compiled.Static() == nil {
		return "", false
	}
	val, _ := compiled.Static().StringValue()
	return val, true
}

func TestStepMachine(t *testing.T) {
	tests := map[string]struct {
		steps      map[string]*StepData
		currentRef string
		expr       string
		wantVal    string
		wantOk     bool
	}{
		"step.results resolves to current step dir": {
			steps:      map[string]*StepData{"ref1": {Id: "build"}},
			currentRef: "ref1",
			expr:       "step.results",
			wantVal:    "/data/.steps/build",
			wantOk:     true,
		},
		"step.results returns nothing without id": {
			steps:      map[string]*StepData{"ref1": {}},
			currentRef: "ref1",
			expr:       "step.results",
			wantOk:     false,
		},
		"step.<id>.results resolves to named step dir": {
			steps:      map[string]*StepData{"ref1": {Id: "build"}, "ref2": {Id: "test"}},
			currentRef: "ref2",
			expr:       "step.build.results",
			wantVal:    "/data/.steps/build",
			wantOk:     true,
		},
		"step.<id>.results returns nothing for unknown id": {
			steps:      map[string]*StepData{"ref1": {Id: "build"}},
			currentRef: "ref1",
			expr:       "step.unknown.results",
			wantOk:     false,
		},
		"unrelated expression not handled": {
			steps:      map[string]*StepData{"ref1": {Id: "build"}},
			currentRef: "ref1",
			expr:       "other.thing",
			wantOk:     false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			setupTestState(tc.steps, tc.currentRef)
			val, ok := resolveStepExpr(t, tc.expr)
			assert.Equal(t, tc.wantOk, ok)
			if tc.wantOk {
				assert.Equal(t, tc.wantVal, val)
			}
		})
	}

}

func TestStepMachine_TemplateExpression(t *testing.T) {
	setupTestState(map[string]*StepData{"ref1": {Id: "build"}}, "ref1")
	compiled, err := expressions.CompileAndResolveTemplate("go build -o {{ step.results }}/binary", StepMachine)
	assert.NoError(t, err)
	val, _ := compiled.Static().StringValue()
	assert.Equal(t, "go build -o /data/.steps/build/binary", val)
}

func TestStepResultsDir(t *testing.T) {
	assert.Equal(t, "/data/.steps/build", StepResultsDir("build"))
	assert.Equal(t, "/data/.steps/run_tests", StepResultsDir("run_tests"))
}

func TestStepData_SetId(t *testing.T) {
	step := &StepData{Ref: "ref1"}
	step.SetId("build")
	assert.Equal(t, "build", step.Id)

	status := constants.StepStatusPassed
	step.Status = &status
	assert.True(t, step.IsFinished())
}
