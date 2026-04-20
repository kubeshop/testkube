package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)
	if compiled.Static() == nil {
		return "", false
	}
	val, _ := compiled.Static().StringValue()
	return val, true
}

func TestStepMachine_Results(t *testing.T) {
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

func TestStepMachine_Outputs(t *testing.T) {
	tests := map[string]struct {
		outputs map[string]map[string]string // stepId -> key -> value
		expr    string
		wantVal string
		wantOk  bool
	}{
		"resolves stored output": {
			outputs: map[string]map[string]string{"auth": {"token": "abc123"}},
			expr:    "step.auth.outputs.token",
			wantVal: "abc123",
			wantOk:  true,
		},
		"returns nothing for missing output": {
			outputs: nil,
			expr:    "step.auth.outputs.token",
			wantOk:  false,
		},
		"isolated between steps": {
			outputs: map[string]map[string]string{"step_a": {"token": "aaa"}, "step_b": {"token": "bbb"}},
			expr:    "step.step_a.outputs.token",
			wantVal: "aaa",
			wantOk:  true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			setupTestState(map[string]*StepData{
				"ref1": {Id: "auth"},
				"ref2": {Id: "step_a"},
				"ref3": {Id: "step_b"},
			}, "ref1")
			for stepId, outputs := range tc.outputs {
				for k, v := range outputs {
					GetState().SetStepOutput(stepId, k, v)
				}
			}
			val, ok := resolveStepExpr(t, tc.expr)
			assert.Equal(t, tc.wantOk, ok)
			if tc.wantOk {
				assert.Equal(t, tc.wantVal, val)
			}
		})
	}
}

func TestStepMachine_TemplateExpression(t *testing.T) {
	setupTestState(map[string]*StepData{"ref1": {Id: "auth"}}, "ref1")
	GetState().SetStepOutput("auth", "token", "mytoken")

	compiled, err := expressions.CompileAndResolveTemplate(
		"curl -H \"Auth: {{ step.auth.outputs.token }}\" https://api",
		StepMachine,
	)
	assert.NoError(t, err)
	val, _ := compiled.Static().StringValue()
	assert.Equal(t, "curl -H \"Auth: mytoken\" https://api", val)
}

func TestScanStepOutputs(t *testing.T) {
	t.Run("scans files and trims whitespace", func(t *testing.T) {
		dir := t.TempDir()
		setupTestState(map[string]*StepData{"ref1": {Id: "auth"}}, "ref1")

		require.NoError(t, os.WriteFile(filepath.Join(dir, "token"), []byte("abc123\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "count"), []byte("42"), 0644))

		require.NoError(t, scanStepOutputsFrom(dir, "auth"))

		val, ok := resolveStepExpr(t, "step.auth.outputs.token")
		assert.True(t, ok)
		assert.Equal(t, "abc123", val)

		val, ok = resolveStepExpr(t, "step.auth.outputs.count")
		assert.True(t, ok)
		assert.Equal(t, "42", val)
	})

	t.Run("skips hidden files", func(t *testing.T) {
		dir := t.TempDir()
		setupTestState(map[string]*StepData{"ref1": {Id: "auth"}}, "ref1")

		require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "visible"), []byte("ok"), 0644))

		require.NoError(t, scanStepOutputsFrom(dir, "auth"))

		val, ok := resolveStepExpr(t, "step.auth.outputs.visible")
		assert.True(t, ok)
		assert.Equal(t, "ok", val)
	})

	t.Run("skips oversized files", func(t *testing.T) {
		dir := t.TempDir()
		setupTestState(map[string]*StepData{"ref1": {Id: "auth"}}, "ref1")

		require.NoError(t, os.WriteFile(filepath.Join(dir, "big"), []byte(strings.Repeat("x", MaxOutputSize+1)), 0644))

		require.NoError(t, scanStepOutputsFrom(dir, "auth"))

		_, ok := resolveStepExpr(t, "step.auth.outputs.big")
		assert.False(t, ok)
	})

	t.Run("noop for empty id or missing dir", func(t *testing.T) {
		assert.NoError(t, scanStepOutputsFrom("/nonexistent", ""))
		assert.NoError(t, scanStepOutputsFrom("/nonexistent", "build"))
	})
}

func TestStepResultsDir(t *testing.T) {
	assert.Equal(t, "/data/.steps/build", StepResultsDir("build"))
	assert.Equal(t, "/data/.steps/run_tests", StepResultsDir("run_tests"))
}
