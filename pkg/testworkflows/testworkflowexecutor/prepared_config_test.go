package testworkflowexecutor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// TestApplyConfig_JSONConfigReadableByJSONFunction guards against a bug where
// ApplyConfig wrapped JSON-shaped values as `tojson(json("<escaped>"))`. The
// wrapped string flowed through castParameter's CompileTemplate as a literal
// (no `{{...}}` interpolation to trigger), so config.<key> ended up registered
// as the plain text "tojson(json(...))". A workflow expression like
// json(config.jobs) then received that text and failed to parse it as JSON.
func TestApplyConfig_JSONConfigReadableByJSONFunction(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"jobs": {Type: testworkflowsv1.ParameterTypeString},
				},
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: `{{ json(config.jobs).0.name }}`,
				},
			},
		},
	}

	ie := NewIntermediateExecution().SetWorkflow(workflow)
	err := ie.ApplyConfig(map[string]string{
		"jobs": `[{"name":"go"},{"name":"node"}]`,
	})

	assert.NoError(t, err)
	assert.Equal(t, "go", ie.cr.Spec.Pod.ServiceAccountName)
}

// TestApplyConfig_JSONConfigColonsSurviveJSONReparse verifies that colons and
// other special chars (`+`, `=`) inside a JSON config value survive ApplyConfig
// and can be pulled back out via json(config.<key>).<field>. Preventing colon
// splitting in JSON config values was the original motivation for PR #6952;
// this test covers that intent at the resolve-time level.
func TestApplyConfig_JSONConfigColonsSurviveJSONReparse(t *testing.T) {
	tests := []struct {
		name      string
		configVal string
		template  string
		expected  string
	}{
		{
			name:      "URL with port read back via json()",
			configVal: `{"url":"https://api.example.com:8080/v1"}`,
			template:  `{{ json(config.data).url }}`,
			expected:  "https://api.example.com:8080/v1",
		},
		{
			name:      "base64 secret with +/= read back via json()",
			configVal: `{"secret":"aGVsbG8rd29ybGQ+c2VjcmV0PQ=="}`,
			template:  `{{ json(config.data).secret }}`,
			expected:  "aGVsbG8rd29ybGQ+c2VjcmV0PQ==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := &testworkflowsv1.TestWorkflow{
				Spec: testworkflowsv1.TestWorkflowSpec{
					TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
						Config: map[string]testworkflowsv1.ParameterSchema{
							"data": {Type: testworkflowsv1.ParameterTypeString},
						},
						Pod: &testworkflowsv1.PodConfig{
							ServiceAccountName: tt.template,
						},
					},
				},
			}

			ie := NewIntermediateExecution().SetWorkflow(workflow)
			err := ie.ApplyConfig(map[string]string{"data": tt.configVal})

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, ie.cr.Spec.Pod.ServiceAccountName)
		})
	}
}

// TestApplyConfig_SimpleValuesUnchanged verifies that plain string values pass
// through ApplyConfig untouched into a `{{ config.<key> }}` substitution site.
func TestApplyConfig_SimpleValuesUnchanged(t *testing.T) {
	tests := []string{
		"production",
		"test-value",
		"key=value",
		"https://example.com",
		"123",
		"true",
		"https://api.example.com:8080",
		"key=value&another=value2",
		"opt1;opt2;opt3",
		"value1,value2,value3",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			workflow := &testworkflowsv1.TestWorkflow{
				Spec: testworkflowsv1.TestWorkflowSpec{
					TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
						Config: map[string]testworkflowsv1.ParameterSchema{
							"v": {Type: testworkflowsv1.ParameterTypeString},
						},
						Pod: &testworkflowsv1.PodConfig{
							ServiceAccountName: `{{ config.v }}`,
						},
					},
				},
			}

			ie := NewIntermediateExecution().SetWorkflow(workflow)
			err := ie.ApplyConfig(map[string]string{"v": input})

			assert.NoError(t, err)
			assert.Equal(t, input, ie.cr.Spec.Pod.ServiceAccountName)
		})
	}
}
