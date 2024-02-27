package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func TestApplyConfigTestWorkflow(t *testing.T) {
	cfg := map[string]intstr.IntOrString{
		"foo":    {Type: intstr.Int, IntVal: 30},
		"bar":    {Type: intstr.String, StrVal: "some value"},
		"baz":    {Type: intstr.String, StrVal: "some {{ 30 }} value"},
		"foobar": {Type: intstr.String, StrVal: "some {{ unknown(300) }} value"},
	}
	want := &testworkflowsv1.TestWorkflow{
		Description: "{{some description here }}",
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: "abra 30",
					Labels: map[string]string{
						"some value-key": "some 30 value",
						"other":          "{{value}}",
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepBase: testworkflowsv1.StepBase{
						Container: &testworkflowsv1.ContainerConfig{
							WorkingDir: common.Ptr("some {{unknown(300)}} value {{another(500)}}"),
						},
					},
				},
			},
		},
	}
	got, err := ApplyWorkflowConfig(&testworkflowsv1.TestWorkflow{
		Description: "{{some description here }}",
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: "abra {{config.foo}}",
					Labels: map[string]string{
						"{{config.bar}}-key": "{{config.baz}}",
						"other":              "{{value}}",
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepBase: testworkflowsv1.StepBase{
						Container: &testworkflowsv1.ContainerConfig{
							WorkingDir: common.Ptr("{{config.foobar}} {{another(500)}}"),
						},
					},
				},
			},
		},
	}, cfg)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestApplyMissingConfig(t *testing.T) {
	cfg := map[string]intstr.IntOrString{
		"foo":    {Type: intstr.Int, IntVal: 30},
		"bar":    {Type: intstr.String, StrVal: "some value"},
		"foobar": {Type: intstr.String, StrVal: "some {{ unknown(300) }} value"},
	}
	_, err := ApplyWorkflowConfig(&testworkflowsv1.TestWorkflow{
		Description: "{{some description here }}",
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: "abra {{config.foo}}",
					Labels: map[string]string{
						"{{config.bar}}-key": "{{config.baz}}",
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{
					StepBase: testworkflowsv1.StepBase{
						Container: &testworkflowsv1.ContainerConfig{
							WorkingDir: common.Ptr("{{config.foobar}} {{another(500)}}"),
						},
					},
				},
			},
		},
	}, cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Spec: TestWorkflowSpecBase: Pod: Labels: {{config.bar}}-key")
	assert.Contains(t, err.Error(), "error while accessing config.baz: unknown variable")
}

func TestApplyConfigTestWorkflowTemplate(t *testing.T) {
	cfg := map[string]intstr.IntOrString{
		"foo":    {Type: intstr.Int, IntVal: 30},
		"bar":    {Type: intstr.String, StrVal: "some value"},
		"baz":    {Type: intstr.String, StrVal: "some {{ 30 }} value"},
		"foobar": {Type: intstr.String, StrVal: "some {{ unknown(300) }} value"},
	}
	want := &testworkflowsv1.TestWorkflowTemplate{
		Description: "{{some description here }}",
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: "abra 30",
					Labels: map[string]string{
						"some value-key": "some 30 value",
					},
				},
			},
			Steps: []testworkflowsv1.IndependentStep{
				{
					StepBase: testworkflowsv1.StepBase{
						Container: &testworkflowsv1.ContainerConfig{
							WorkingDir: common.Ptr("some {{unknown(300)}} value {{another(500)}}"),
						},
					},
				},
			},
		},
	}
	got, err := ApplyWorkflowTemplateConfig(&testworkflowsv1.TestWorkflowTemplate{
		Description: "{{some description here }}",
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					ServiceAccountName: "abra {{config.foo}}",
					Labels: map[string]string{
						"{{config.bar}}-key": "{{config.baz}}",
					},
				},
			},
			Steps: []testworkflowsv1.IndependentStep{
				{
					StepBase: testworkflowsv1.StepBase{
						Container: &testworkflowsv1.ContainerConfig{
							WorkingDir: common.Ptr("{{config.foobar}} {{another(500)}}"),
						},
					},
				},
			},
		},
	}, cfg)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
