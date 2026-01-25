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

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/rand"
)

func buildTemplate(template *testworkflowsv1.TestWorkflowTemplate, cfg map[string]intstr.IntOrString,
	externalize func(key, value string) (expressions.Expression, error)) (*testworkflowsv1.TestWorkflowTemplate, error) {
	v, err := ApplyWorkflowTemplateConfig(template.DeepCopy(), cfg, externalize)
	if err != nil {
		return template, err
	}
	return v, err
}

func getTemplate(name string, templates map[string]*testworkflowsv1.TestWorkflowTemplate) (tpl *testworkflowsv1.TestWorkflowTemplate, err error) {
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

func getConfiguredTemplate(name string, cfg map[string]intstr.IntOrString, templates map[string]*testworkflowsv1.TestWorkflowTemplate,
	externalize func(key, value string) (expressions.Expression, error)) (tpl *testworkflowsv1.TestWorkflowTemplate, err error) {
	tpl, err = getTemplate(name, templates)
	if err != nil {
		return tpl, err
	}
	return buildTemplate(tpl, cfg, externalize)
}

func injectTemplateToSpec(spec *testworkflowsv1.TestWorkflowSpec, template *testworkflowsv1.TestWorkflowTemplate) error {
	if spec == nil {
		return nil
	}
	// Apply top-level configuration
	spec.Pod = MergePodConfig(template.Spec.Pod, spec.Pod)
	spec.Job = MergeJobConfig(template.Spec.Job, spec.Job)
	spec.Events = append(template.Spec.Events, spec.Events...)
	spec.Execution = MergeExecution(template.Spec.Execution, spec.Execution)
	spec.Concurrency = MergeConcurrency(template.Spec.Concurrency, spec.Concurrency)
	spec.Pvcs = MergeMap(template.Spec.Pvcs, spec.Pvcs)

	// Apply basic configuration
	spec.Content = MergeContent(template.Spec.Content, spec.Content)
	spec.Services = MergeMap(common.MapMap(template.Spec.Services, ConvertIndependentServiceToService), spec.Services)
	spec.Container = MergeContainerConfig(template.Spec.Container, spec.Container)
	spec.System = MergeSystem(template.Spec.System, spec.System)

	// Include the steps from the template
	setup := common.MapSlice(template.Spec.Setup, ConvertIndependentStepToStep)
	spec.Setup = append(setup, spec.Setup...)
	steps := common.MapSlice(template.Spec.Steps, ConvertIndependentStepToStep)
	spec.Steps = append(steps, spec.Steps...)
	after := common.MapSlice(template.Spec.After, ConvertIndependentStepToStep)
	spec.After = append(spec.After, after...)
	return nil
}

func InjectStepTemplate(step *testworkflowsv1.Step, template *testworkflowsv1.TestWorkflowTemplate) error {
	if step == nil {
		return nil
	}

	// Apply basic configuration
	step.Content = MergeContent(template.Spec.Content, step.Content)
	step.Services = MergeMap(common.MapMap(template.Spec.Services, ConvertIndependentServiceToService), step.Services)
	step.Container = MergeContainerConfig(template.Spec.Container, step.Container)

	// Define the step purity
	if step.Pure != nil && template.Spec.System != nil && template.Spec.System.PureByDefault != nil && *template.Spec.System.PureByDefault {
		step.Pure = common.Ptr(true)
	}

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

func InjectServiceTemplate(svc *testworkflowsv1.ServiceSpec, template *testworkflowsv1.TestWorkflowTemplate) error {
	if svc == nil {
		return nil
	}
	svc.Pod = MergePodConfig(template.Spec.Pod, svc.Pod)
	svc.Content = MergeContent(template.Spec.Content, svc.Content)
	svc.ContainerConfig = *MergeContainerConfig(template.Spec.Container, &svc.ContainerConfig)
	svc.Pvcs = MergeMap(template.Spec.Pvcs, svc.Pvcs)
	return nil
}

func applyTemplatesToStep(step testworkflowsv1.Step, templates map[string]*testworkflowsv1.TestWorkflowTemplate,
	externalize func(key, value string) (expressions.Expression, error)) (testworkflowsv1.Step, error) {
	// Apply regular templates
	for i := len(step.Use) - 1; i >= 0; i-- {
		ref := step.Use[i]
		tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates, externalize)
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
		tpl, err := getConfiguredTemplate(step.Template.Name, step.Template.Config, templates, externalize)
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

	// Apply templates to the services
	for name, svc := range step.Services {
		for i := len(svc.Use) - 1; i >= 0; i-- {
			ref := svc.Use[i]
			tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates, externalize)
			if err != nil {
				return step, errors.Wrap(err, fmt.Sprintf("services[%s].use[%d]: resolving template", name, i))
			}
			if len(tpl.Spec.Setup) > 0 || len(tpl.Spec.Steps) > 0 || len(tpl.Spec.After) > 0 {
				return step, fmt.Errorf("services[%s].use[%d]: steps in template used for the service are not supported", name, i)
			}
			if len(tpl.Spec.Services) > 0 {
				return step, fmt.Errorf("services[%s].use[%d]: additional services in template used for the service are not supported", name, i)
			}
			err = InjectServiceTemplate(&svc, tpl)
			if err != nil {
				return step, errors.Wrap(err, fmt.Sprintf("services[%s].use[%d]: injecting template", name, i))
			}
		}
		svc.Use = nil
		step.Services[name] = svc
	}

	// Apply templates in the parallel steps
	if step.Parallel != nil {
		// Move the template operation alias along with other operations,
		// so they can be properly resolved and isolated
		if step.Parallel.Template != nil {
			step.Parallel.Steps = append([]testworkflowsv1.Step{{
				StepControl:    step.Parallel.StepControl,
				StepOperations: step.Parallel.StepOperations,
				Template:       step.Parallel.Template,
			}}, step.Parallel.Steps...)
			step.Parallel.StepControl = testworkflowsv1.StepControl{}
			step.Parallel.StepOperations = testworkflowsv1.StepOperations{}
			step.Parallel.Template = nil
		}

		// Resolve the spec inside of parallel step
		testWorkflowSpec := step.Parallel.NewTestWorkflowSpec()
		err := applyTemplatesToSpec(testWorkflowSpec, templates, externalize)
		if err != nil {
			return step, errors.Wrap(err, ".parallel")
		}
		step.Parallel.Use = testWorkflowSpec.Use
		step.Parallel.Events = testWorkflowSpec.Events
		step.Parallel.System = testWorkflowSpec.System
		step.Parallel.Config = testWorkflowSpec.Config
		step.Parallel.Content = testWorkflowSpec.Content
		step.Parallel.Container = testWorkflowSpec.Container
		step.Parallel.Job = testWorkflowSpec.Job
		step.Parallel.Pod = testWorkflowSpec.Pod
		step.Parallel.Notifications = testWorkflowSpec.Notifications
		step.Parallel.Execution = testWorkflowSpec.Execution
		step.Parallel.Services = testWorkflowSpec.Services
		step.Parallel.Setup = testWorkflowSpec.Setup
		step.Parallel.Steps = testWorkflowSpec.Steps
		step.Parallel.After = testWorkflowSpec.After
		step.Parallel.Pvcs = testWorkflowSpec.Pvcs
	}

	// Resolve templates in the sub-steps
	var err error
	for i := range step.Setup {
		step.Setup[i], err = applyTemplatesToStep(step.Setup[i], templates, externalize)
		if err != nil {
			return step, errors.Wrap(err, fmt.Sprintf(".steps[%d]", i))
		}
	}
	for i := range step.Steps {
		step.Steps[i], err = applyTemplatesToStep(step.Steps[i], templates, externalize)
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

func applyTemplatesToSpec(spec *testworkflowsv1.TestWorkflowSpec, templates map[string]*testworkflowsv1.TestWorkflowTemplate,
	externalize func(key, value string) (expressions.Expression, error)) error {
	if spec == nil {
		return nil
	}

	// Encapsulate TestWorkflow configuration to not pass it into templates accidentally
	random := rand.String(10)
	err := expressions.Simplify(spec, expressions.ReplacePrefixMachine("config.", random+"."))
	if err != nil {
		return err
	}
	defer expressions.Simplify(spec, expressions.ReplacePrefixMachine(random+".", "config."))

	// Apply top-level templates
	for i := len(spec.Use) - 1; i >= 0; i-- {
		ref := spec.Use[i]
		tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates, externalize)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.use[%d]: resolving template", i))
		}
		err = injectTemplateToSpec(spec, tpl)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.use[%d]: injecting template", i))
		}
	}
	spec.Use = nil

	// Apply templates to the services
	for name, svc := range spec.Services {
		for i := len(svc.Use) - 1; i >= 0; i-- {
			ref := svc.Use[i]
			tpl, err := getConfiguredTemplate(ref.Name, ref.Config, templates, externalize)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("services[%s].use[%d]: resolving template", name, i))
			}
			if len(tpl.Spec.Setup) > 0 || len(tpl.Spec.Steps) > 0 || len(tpl.Spec.After) > 0 {
				return fmt.Errorf("services[%s].use[%d]: steps in template used for the service are not supported", name, i)
			}
			if len(tpl.Spec.Services) > 0 {
				return fmt.Errorf("services[%s].use[%d]: additional services in template used for the service are not supported", name, i)
			}
			err = InjectServiceTemplate(&svc, tpl)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("services[%s].use[%d]: injecting template", name, i))
			}
		}
		svc.Use = nil
		spec.Services[name] = svc
	}

	// Apply templates on the step level
	for i := range spec.Setup {
		spec.Setup[i], err = applyTemplatesToStep(spec.Setup[i], templates, externalize)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.setup[%d]", i))
		}
	}
	for i := range spec.Steps {
		spec.Steps[i], err = applyTemplatesToStep(spec.Steps[i], templates, externalize)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("spec.steps[%d]", i))
		}
	}
	for i := range spec.After {
		spec.After[i], err = applyTemplatesToStep(spec.After[i], templates, externalize)
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

func ApplyTemplates(workflow *testworkflowsv1.TestWorkflow, templates map[string]*testworkflowsv1.TestWorkflowTemplate,
	externalize func(key, value string) (expressions.Expression, error)) error {
	if workflow == nil {
		return nil
	}
	return applyTemplatesToSpec(&workflow.Spec, templates, externalize)
}

func addGlobalTemplateRefToStep(step *testworkflowsv1.Step, ref testworkflowsv1.TemplateRef) {
	if step.Parallel != nil {
		addGlobalTemplateRefToSpec(step.Parallel.NewTestWorkflowSpec(), ref)
	}
	for i := range step.Setup {
		addGlobalTemplateRefToStep(&step.Setup[i], ref)
	}
	for i := range step.Steps {
		addGlobalTemplateRefToStep(&step.Steps[i], ref)
	}
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
}

func AddGlobalTemplateRef(t *testworkflowsv1.TestWorkflow, ref testworkflowsv1.TemplateRef) {
	if t != nil {
		addGlobalTemplateRefToSpec(&t.Spec, ref)
	}
}
