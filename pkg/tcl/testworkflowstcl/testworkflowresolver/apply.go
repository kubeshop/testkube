// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

func buildTemplate(template testworkflowsv1.TestWorkflowTemplate, cfg map[string]intstr.IntOrString) (testworkflowsv1.TestWorkflowTemplate, error) {
	v, err := ApplyWorkflowTemplateConfig(template.DeepCopy(), cfg)
	if err != nil {
		return template, err
	}
	return *v, err
}

func getTemplate(name string, templates map[string]testworkflowsv1.TestWorkflowTemplate) (tpl testworkflowsv1.TestWorkflowTemplate, err error) {
	key := GetInternalTemplateName(name)
	tpl, ok := templates[key]
	if ok {
		return tpl, nil
	}
	key = GetDisplayTemplateName(key)
	tpl, ok = templates[key]
	if ok {
		return tpl, nil
	}
	return tpl, fmt.Errorf(`template "%s" not found`, name)
}

func getConfiguredTemplate(name string, cfg map[string]intstr.IntOrString, templates map[string]testworkflowsv1.TestWorkflowTemplate) (tpl testworkflowsv1.TestWorkflowTemplate, err error) {
	tpl, err = getTemplate(name, templates)
	if err != nil {
		return tpl, err
	}
	return buildTemplate(tpl, cfg)
}

func injectTemplateToSpec(spec *testworkflowsv1.TestWorkflowSpec, template testworkflowsv1.TestWorkflowTemplate) error {
	if spec == nil {
		return nil
	}
	// Apply top-level configuration
	spec.Pod = MergePodConfig(template.Spec.Pod, spec.Pod)
	spec.Job = MergeJobConfig(template.Spec.Job, spec.Job)

	// Apply basic configuration
	spec.Content = MergeContent(template.Spec.Content, spec.Content)
	spec.Container = MergeContainerConfig(template.Spec.Container, spec.Container)

	// Include the steps from the template
	setup := common.MapSlice(template.Spec.Setup, ConvertIndependentStepToStep)
	spec.Setup = append(setup, spec.Setup...)
	steps := common.MapSlice(template.Spec.Steps, ConvertIndependentStepToStep)
	spec.Steps = append(steps, spec.Steps...)
	after := common.MapSlice(template.Spec.After, ConvertIndependentStepToStep)
	spec.After = append(spec.After, after...)
	return nil
}

func InjectTemplate(workflow *testworkflowsv1.TestWorkflow, template testworkflowsv1.TestWorkflowTemplate) error {
	if workflow == nil {
		return nil
	}
	return injectTemplateToSpec(&workflow.Spec, template)
}

func InjectStepTemplate(step *testworkflowsv1.Step, template testworkflowsv1.TestWorkflowTemplate) error {
	if step == nil {
		return nil
	}

	// Apply basic configuration
	step.Content = MergeContent(template.Spec.Content, step.Content)
	step.Container = MergeContainerConfig(template.Spec.Container, step.Container)

	// Fast-track when the template doesn't contain any steps to run
	if len(template.Spec.Setup) == 0 && len(template.Spec.Steps) == 0 && len(template.Spec.After) == 0 {
		return nil
	}

	// Decouple sub-steps from the template
	setup := common.MapSlice(template.Spec.Setup, ConvertIndependentStepToStep)
	steps := common.MapSlice(template.Spec.Steps, ConvertIndependentStepToStep)
	after := common.MapSlice(template.Spec.After, ConvertIndependentStepToStep)

	step.Setup = append(setup, step.Setup...)
	step.Steps = append(steps, append(step.Steps, after...)...)

	return nil
}

func applyTemplatesToStep(step testworkflowsv1.Step, templates map[string]testworkflowsv1.TestWorkflowTemplate) (testworkflowsv1.Step, error) {
	// Apply regular templates
	for i, ref := range step.Use {
		tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates)
		if err != nil {
			return step, errors.Wrap(err, fmt.Sprintf(".use[%d]: resolving template", i))
		}
		err = InjectStepTemplate(&step, tpl)
		if err != nil {
			return step, errors.Wrap(err, fmt.Sprintf(".use[%d]: injecting template", i))
		}
	}
	step.Use = nil

	// Apply alternative template syntax
	if step.Template != nil {
		tpl, err := getConfiguredTemplate(step.Template.Name, step.Template.Config, templates)
		if err != nil {
			return step, errors.Wrap(err, ".template: resolving template")
		}
		isolate := testworkflowsv1.Step{}
		err = InjectStepTemplate(&isolate, tpl)
		if err != nil {
			return step, errors.Wrap(err, ".template: injecting template")
		}

		if len(isolate.Setup) > 0 || len(isolate.Steps) > 0 {
			if isolate.Container == nil && isolate.Content == nil && isolate.WorkingDir == nil {
				step.Steps = append(append(isolate.Setup, isolate.Steps...), step.Steps...)
			} else {
				step.Steps = append([]testworkflowsv1.Step{isolate}, step.Steps...)
			}
		}

		step.Template = nil
	}

	// Apply templates in the parallel steps
	if step.Parallel != nil {
		err := applyTemplatesToSpec(&step.Parallel.TestWorkflowSpec, templates)
		if err != nil {
			return step, errors.Wrap(err, ".parallel")
		}
	}

	// Resolve templates in the sub-steps
	var err error
	for i := range step.Setup {
		step.Setup[i], err = applyTemplatesToStep(step.Setup[i], templates)
		if err != nil {
			return step, errors.Wrap(err, fmt.Sprintf(".steps[%d]", i))
		}
	}
	for i := range step.Steps {
		step.Steps[i], err = applyTemplatesToStep(step.Steps[i], templates)
		if err != nil {
			return step, errors.Wrap(err, fmt.Sprintf(".steps[%d]", i))
		}
	}

	return step, nil
}

func FlattenStepList(steps []testworkflowsv1.Step) []testworkflowsv1.Step {
	changed := false
	result := make([]testworkflowsv1.Step, 0, len(steps))
	for _, step := range steps {
		setup := step.Setup
		sub := step.Steps
		step.Setup = nil
		step.Steps = nil
		if reflect.ValueOf(step).IsZero() {
			changed = true
			result = append(result, append(setup, sub...)...)
		} else {
			step.Setup = setup
			step.Steps = sub
			result = append(result, step)
		}
	}
	if !changed {
		return steps
	}
	return result
}

func applyTemplatesToSpec(spec *testworkflowsv1.TestWorkflowSpec, templates map[string]testworkflowsv1.TestWorkflowTemplate) error {
	if spec == nil {
		return nil
	}

	// Encapsulate TestWorkflow configuration to not pass it into templates accidentally
	random := rand.String(10)
	err := expressionstcl.Simplify(spec, expressionstcl.ReplacePrefixMachine("config.", random+"."))
	if err != nil {
		return err
	}
	defer expressionstcl.Simplify(spec, expressionstcl.ReplacePrefixMachine(random+".", "config."))

	// Apply top-level templates
	for i, ref := range spec.Use {
		tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.use[%d]: resolving template", i))
		}
		err = injectTemplateToSpec(spec, tpl)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.use[%d]: injecting template", i))
		}
	}
	spec.Use = nil

	// Apply templates on the step level
	for i := range spec.Setup {
		spec.Setup[i], err = applyTemplatesToStep(spec.Setup[i], templates)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.setup[%d]", i))
		}
	}
	for i := range spec.Steps {
		spec.Steps[i], err = applyTemplatesToStep(spec.Steps[i], templates)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.steps[%d]", i))
		}
	}
	for i := range spec.After {
		spec.After[i], err = applyTemplatesToStep(spec.After[i], templates)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.after[%d]", i))
		}
	}

	// Simplify the lists
	spec.Setup = FlattenStepList(spec.Setup)
	spec.Steps = FlattenStepList(spec.Steps)
	spec.After = FlattenStepList(spec.After)

	return nil
}

func ApplyTemplates(workflow *testworkflowsv1.TestWorkflow, templates map[string]testworkflowsv1.TestWorkflowTemplate) error {
	if workflow == nil {
		return nil
	}
	return applyTemplatesToSpec(&workflow.Spec, templates)
}

func addGlobalTemplateRefToStep(step *testworkflowsv1.Step, ref testworkflowsv1.TemplateRef) {
	if step.Parallel != nil {
		addGlobalTemplateRefToSpec(&step.Parallel.TestWorkflowSpec, ref)
	}
	for i := range step.Setup {
		addGlobalTemplateRefToStep(&step.Setup[i], ref)
	}
	for i := range step.Steps {
		addGlobalTemplateRefToStep(&step.Steps[i], ref)
	}
	return
}

func addGlobalTemplateRefToSpec(spec *testworkflowsv1.TestWorkflowSpec, ref testworkflowsv1.TemplateRef) {
	if spec == nil {
		return
	}
	spec.Use = append([]testworkflowsv1.TemplateRef{ref}, spec.Use...)
	for i := range spec.Setup {
		addGlobalTemplateRefToStep(&spec.Setup[i], ref)
	}
	for i := range spec.Steps {
		addGlobalTemplateRefToStep(&spec.Steps[i], ref)
	}
	for i := range spec.After {
		addGlobalTemplateRefToStep(&spec.After[i], ref)
	}
	return
}

func AddGlobalTemplateRef(t *testworkflowsv1.TestWorkflow, ref testworkflowsv1.TemplateRef) {
	if t != nil {
		addGlobalTemplateRefToSpec(&t.Spec, ref)
	}
}
