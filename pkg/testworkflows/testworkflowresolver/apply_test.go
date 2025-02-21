// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

var (
	tplPod = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"v1": "v2",
					},
				},
			},
		},
	}
	tplPodConfig = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"department": {Type: testworkflowsv1.ParameterTypeString},
				},
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"department": "{{config.department}}",
					},
				},
			},
		},
	}
	tplEnv = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					Env: []corev1.EnvVar{
						{Name: "test", Value: "the"},
					},
				},
			},
		},
	}
	tplSteps = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			Setup: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "setup-tpl-test"}},
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "steps-tpl-test"}},
			},
			After: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "after-tpl-test"}},
			},
		},
	}
	tplStepsEnv = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					Env: []corev1.EnvVar{
						{Name: "test", Value: "the"},
					},
				},
			},
			Setup: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "setup-tpl-test"}},
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "steps-tpl-test"}},
			},
			After: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "after-tpl-test"}},
			},
		},
	}
	tplStepsConfig = testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"index": {Type: testworkflowsv1.ParameterTypeInteger},
				},
			},
			Setup: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "setup-tpl-test-{{ config.index }}"}},
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "steps-tpl-test-{{ config.index }}"}},
			},
			After: []testworkflowsv1.IndependentStep{
				{StepMeta: testworkflowsv1.StepMeta{Name: "after-tpl-test-{{ config.index }}"}},
			},
		},
	}
	templates = map[string]*testworkflowsv1.TestWorkflowTemplate{
		"pod":         &tplPod,
		"podConfig":   &tplPodConfig,
		"env":         &tplEnv,
		"steps":       &tplSteps,
		"stepsEnv":    &tplStepsEnv,
		"stepsConfig": &tplStepsConfig,
	}
	tplPodRef       = testworkflowsv1.TemplateRef{Name: "pod"}
	tplPodConfigRef = testworkflowsv1.TemplateRef{
		Name: "podConfig",
		Config: map[string]intstr.IntOrString{
			"department": {Type: intstr.String, StrVal: "test-department"},
		},
	}
	tplPodConfigRefEmpty = testworkflowsv1.TemplateRef{Name: "podConfig"}
	tplEnvRef            = testworkflowsv1.TemplateRef{Name: "env"}
	tplStepsRef          = testworkflowsv1.TemplateRef{Name: "steps"}
	tplStepsEnvRef       = testworkflowsv1.TemplateRef{Name: "stepsEnv"}
	tplStepsConfigRef    = testworkflowsv1.TemplateRef{Name: "stepsConfig", Config: map[string]intstr.IntOrString{
		"index": {Type: intstr.Int, IntVal: 20},
	}}
	tplStepsConfigRefStringInvalid = testworkflowsv1.TemplateRef{Name: "stepsConfig", Config: map[string]intstr.IntOrString{
		"index": {Type: intstr.String, StrVal: "text"},
	}}
	tplStepsConfigRefStringValid = testworkflowsv1.TemplateRef{Name: "stepsConfig", Config: map[string]intstr.IntOrString{
		"index": {Type: intstr.String, StrVal: "10"},
	}}
	workflowPod = testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"the": "value",
					},
				},
			},
		},
	}
	workflowPodConfig = testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"department": {Type: testworkflowsv1.ParameterTypeString},
				},
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"department": "{{config.department}}",
					},
				},
			},
		},
	}
	workflowSteps = testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Setup: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "setup-tpl"}},
			},
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "steps-tpl"}},
			},
			After: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "after-tpl"}},
			},
		},
	}
	basicStep = testworkflowsv1.Step{
		StepMeta: testworkflowsv1.StepMeta{
			Name: "basic",
		},
		StepDefaults: testworkflowsv1.StepDefaults{
			Container: &testworkflowsv1.ContainerConfig{
				Env: []corev1.EnvVar{
					{Name: "XYZ", Value: "some-value"},
				},
			},
		},
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "shell-command",
		},
	}
	advancedStep = testworkflowsv1.Step{
		StepMeta: testworkflowsv1.StepMeta{
			Name:      "basic",
			Condition: "always",
		},
		StepDefaults: testworkflowsv1.StepDefaults{
			Container: &testworkflowsv1.ContainerConfig{
				Env: []corev1.EnvVar{
					{Name: "XYZ", Value: "some-value"},
				},
			},
		},
		StepOperations: testworkflowsv1.StepOperations{
			Delay: "5s",
			Shell: "another-shell-command",
			Artifacts: &testworkflowsv1.StepArtifacts{
				Paths: []string{"a", "b", "c"},
			},
		},
		Steps: []testworkflowsv1.Step{
			basicStep,
		},
	}
)

func TestApplyTemplatesMissingTemplate(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{{Name: "unknown"}}
	err := ApplyTemplates(wf, templates, nil)

	assert.Error(t, err)
	assert.Equal(t, err.Error(), `spec.use[0]: resolving template: template "unknown" not found`)
}

func TestApplyTemplatesMissingConfig(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplPodConfigRefEmpty}
	err := ApplyTemplates(wf, templates, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), `spec.use[0]: resolving template:`)
	assert.Contains(t, err.Error(), `config.department: unknown variable`)
}

func TestApplyTemplatesInvalidConfig(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplStepsConfigRefStringInvalid}
	err := ApplyTemplates(wf, templates, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), `spec.use[0]: resolving template: config.index`)
	assert.Contains(t, err.Error(), `error while converting value to number`)
}

func TestApplyTemplatesConfig(t *testing.T) {
	wf := workflowPod.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplPodConfigRef}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowPod.DeepCopy()
	want.Spec.Pod.Labels["department"] = "test-department"

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplatesNoConfigMismatchNoOverride(t *testing.T) {
	wf := workflowPodConfig.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplPodConfigRef}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowPodConfig.DeepCopy()
	want.Spec.Pod.Labels["department"] = "{{config.department}}"

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplatesMergeTopLevelSteps(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplStepsRef}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowSteps.DeepCopy()
	want.Spec.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		want.Spec.Setup[0],
	}
	want.Spec.Steps = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		want.Spec.Steps[0],
	}
	want.Spec.After = []testworkflowsv1.Step{
		want.Spec.After[0],
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplatesMergeMultipleTopLevelSteps(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplStepsRef, tplStepsConfigRef}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowSteps.DeepCopy()
	want.Spec.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Setup[0]),
		want.Spec.Setup[0],
	}
	want.Spec.Setup[1].Name = "setup-tpl-test-20"
	want.Spec.Steps = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Steps[0]),
		want.Spec.Steps[0],
	}
	want.Spec.Steps[1].Name = "steps-tpl-test-20"
	want.Spec.After = []testworkflowsv1.Step{
		want.Spec.After[0],
		ConvertIndependentStepToStep(tplStepsConfig.Spec.After[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}
	want.Spec.After[1].Name = "after-tpl-test-20"

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplatesMergeMultipleConfigurable(t *testing.T) {
	wf := workflowSteps.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{tplStepsConfigRefStringValid, tplStepsConfigRef}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowSteps.DeepCopy()
	want.Spec.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Setup[0]),
		want.Spec.Setup[0],
	}
	want.Spec.Setup[0].Name = "setup-tpl-test-10"
	want.Spec.Setup[1].Name = "setup-tpl-test-20"
	want.Spec.Steps = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Steps[0]),
		want.Spec.Steps[0],
	}
	want.Spec.Steps[0].Name = "steps-tpl-test-10"
	want.Spec.Steps[1].Name = "steps-tpl-test-20"
	want.Spec.After = []testworkflowsv1.Step{
		want.Spec.After[0],
		ConvertIndependentStepToStep(tplStepsConfig.Spec.After[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.After[0]),
	}
	want.Spec.After[1].Name = "after-tpl-test-20"
	want.Spec.After[2].Name = "after-tpl-test-10"

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplatesStepBasic(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplEnvRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()
	want.Container.Env = append(tplEnv.Spec.Container.Env, want.Container.Env...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepIgnorePod(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplPodRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepBasicIsolatedIgnore(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Template = &tplEnvRef
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepBasicIsolated(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Template = &tplStepsRef
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}, want.Steps...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepBasicIsolatedWrapped(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Template = &tplStepsEnvRef
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()
	want.Steps = append([]testworkflowsv1.Step{{
		StepDefaults: testworkflowsv1.StepDefaults{
			Container: tplStepsEnv.Spec.Container,
		},
		Setup: []testworkflowsv1.Step{
			ConvertIndependentStepToStep(tplStepsEnv.Spec.Setup[0]),
		},
		Steps: []testworkflowsv1.Step{
			ConvertIndependentStepToStep(tplStepsEnv.Spec.Steps[0]),
			ConvertIndependentStepToStep(tplStepsEnv.Spec.After[0]),
		},
	}}, want.Steps...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepBasicSteps(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplStepsRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()
	want.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
	}
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
	}, append(want.Steps, []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}...)...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepBasicMultipleSteps(t *testing.T) {
	s := *basicStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplStepsRef, tplStepsConfigRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *basicStep.DeepCopy()
	want.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Setup[0]),
	}
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Steps[0]),
	}, append(want.Steps, []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplStepsConfig.Spec.After[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}...)...)
	want.Setup[1].Name = "setup-tpl-test-20"
	want.Steps[1].Name = "steps-tpl-test-20"
	want.Steps[2].Name = "after-tpl-test-20"

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepAdvancedIsolated(t *testing.T) {
	s := *advancedStep.DeepCopy()
	s.Template = &tplStepsRef
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *advancedStep.DeepCopy()
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}, want.Steps...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepAdvancedIsolatedWrapped(t *testing.T) {
	s := *advancedStep.DeepCopy()
	s.Template = &tplStepsEnvRef
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *advancedStep.DeepCopy()
	want.Steps = append([]testworkflowsv1.Step{{
		StepDefaults: testworkflowsv1.StepDefaults{
			Container: tplStepsEnv.Spec.Container,
		},
		Setup: []testworkflowsv1.Step{
			ConvertIndependentStepToStep(tplStepsEnv.Spec.Setup[0]),
		},
		Steps: []testworkflowsv1.Step{
			ConvertIndependentStepToStep(tplStepsEnv.Spec.Steps[0]),
			ConvertIndependentStepToStep(tplStepsEnv.Spec.After[0]),
		},
	}}, want.Steps...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesParallel(t *testing.T) {
	s := *advancedStep.DeepCopy()
	s.Parallel = &testworkflowsv1.StepParallel{
		TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
			Use:   []testworkflowsv1.TemplateRef{tplStepsEnvRef},
			Steps: []testworkflowsv1.Step{basicStep},
		},
	}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *advancedStep.DeepCopy()
	want.Parallel = &testworkflowsv1.StepParallel{
		TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					Env: []corev1.EnvVar{
						{Name: "test", Value: "the"},
					},
				},
			},
			Setup: []testworkflowsv1.Step{
				ConvertIndependentStepToStep(tplStepsEnv.Spec.Setup[0]),
			},
			Steps: []testworkflowsv1.Step{
				ConvertIndependentStepToStep(tplStepsEnv.Spec.Steps[0]),
				basicStep,
			},
			After: []testworkflowsv1.Step{
				ConvertIndependentStepToStep(tplStepsEnv.Spec.After[0]),
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepAdvancedSteps(t *testing.T) {
	s := *advancedStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplStepsRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *advancedStep.DeepCopy()
	want.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
	}
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
	}, append(want.Steps, []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}...)...)

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesStepAdvancedMultipleSteps(t *testing.T) {
	s := *advancedStep.DeepCopy()
	s.Use = []testworkflowsv1.TemplateRef{tplStepsRef, tplStepsConfigRef}
	s, err := applyTemplatesToStep(s, templates, nil)

	want := *advancedStep.DeepCopy()
	want.Setup = []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Setup[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Setup[0]),
	}
	want.Steps = append([]testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplSteps.Spec.Steps[0]),
		ConvertIndependentStepToStep(tplStepsConfig.Spec.Steps[0]),
	}, append(want.Steps, []testworkflowsv1.Step{
		ConvertIndependentStepToStep(tplStepsConfig.Spec.After[0]),
		ConvertIndependentStepToStep(tplSteps.Spec.After[0]),
	}...)...)
	want.Setup[1].Name = "setup-tpl-test-20"
	want.Steps[1].Name = "steps-tpl-test-20"
	want.Steps[3].Name = "after-tpl-test-20"

	assert.NoError(t, err)
	assert.Equal(t, want, s)
}

func TestApplyTemplatesConfigOverflow(t *testing.T) {
	wf := workflowPod.DeepCopy()
	wf.Spec.Use = []testworkflowsv1.TemplateRef{{
		Name: "podConfig",
		Config: map[string]intstr.IntOrString{
			"department": {Type: intstr.String, StrVal: "{{config.value}}"},
		},
	}}
	err := ApplyTemplates(wf, templates, nil)

	want := workflowPod.DeepCopy()
	want.Spec.Pod.Labels["department"] = "{{config.value}}"

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplates_ConditionAlways(t *testing.T) {
	tpls := map[string]*testworkflowsv1.TestWorkflowTemplate{
		"example": {
			Spec: testworkflowsv1.TestWorkflowTemplateSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Config: map[string]testworkflowsv1.ParameterSchema{
						"result": {
							Type:    testworkflowsv1.ParameterTypeString,
							Default: &intstr.IntOrString{Type: intstr.String, StrVal: ""},
						},
					},
				},
				Steps: []testworkflowsv1.IndependentStep{
					{
						StepMeta: testworkflowsv1.StepMeta{Condition: "always"},
						StepOperations: testworkflowsv1.StepOperations{
							Run: &testworkflowsv1.StepRun{
								ContainerConfig: testworkflowsv1.ContainerConfig{
									Env: []corev1.EnvVar{
										{Name: "result", Value: "{{ config.result }}"},
									},
								},
								Shell: common.Ptr("echo $result"),
							},
						},
					},
				},
			},
		},
	}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
				{Template: &testworkflowsv1.TemplateRef{
					Name: "example",
					Config: map[string]intstr.IntOrString{
						"result": {Type: intstr.String, StrVal: "{{ passed }}"},
					},
				}},
			},
		},
	}
	err := ApplyTemplates(wf, tpls, nil)

	want := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
				{
					StepMeta: testworkflowsv1.StepMeta{Condition: "always"},
					StepOperations: testworkflowsv1.StepOperations{
						Run: &testworkflowsv1.StepRun{
							ContainerConfig: testworkflowsv1.ContainerConfig{
								Env: []corev1.EnvVar{
									{Name: "result", Value: "{{passed}}"},
								},
							},
							Shell: common.Ptr("echo $result"),
						},
					},
				},
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}

func TestApplyTemplates_MergePodValues(t *testing.T) {
	tpls := map[string]*testworkflowsv1.TestWorkflowTemplate{
		"top": {
			Spec: testworkflowsv1.TestWorkflowTemplateSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Pod: &testworkflowsv1.PodConfig{
						Labels: map[string]string{
							"label1": "topvalue",
							"label2": "topvalue",
							"label3": "topvalue",
						},
					},
				},
			},
		},
		"middle": {
			Spec: testworkflowsv1.TestWorkflowTemplateSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Pod: &testworkflowsv1.PodConfig{
						Labels: map[string]string{
							"label1": "middlevalue",
							"label2": "middlevalue",
						},
					},
				},
			},
		},
	}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"label1": "workflowvalue",
					},
				},
			},
			Use: []testworkflowsv1.TemplateRef{
				{Name: "top"},
				{Name: "middle"},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
			},
		},
	}
	err := ApplyTemplates(wf, tpls, nil)

	want := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Pod: &testworkflowsv1.PodConfig{
					Labels: map[string]string{
						"label1": "workflowvalue",
						"label2": "middlevalue",
						"label3": "topvalue",
					},
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, wf)
}
