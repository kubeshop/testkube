// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func applyStepBaseConfig(target testworkflowsv1.StepBase, src testworkflowsv1.StepBase) testworkflowsv1.StepBase {
	target.Name = src.Name
	target.Condition = src.Condition
	target.Negative = src.Negative
	target.Optional = src.Optional
	target.VirtualGroup = src.VirtualGroup
	target.Retry = src.Retry
	target.Timeout = src.Timeout
	target.Delay = src.Delay
	target.Content = src.Content
	target.WorkingDir = src.WorkingDir
	target.Container = src.Container
	target.Artifacts = src.Artifacts
	return target
}

func getStepBaseConfig(s testworkflowsv1.StepBase) testworkflowsv1.StepBase {
	return applyStepBaseConfig(testworkflowsv1.StepBase{}, s)
}

func getSetupBaseSteps(s testworkflowsv1.StepBase) []testworkflowsv1.StepBase {
	steps := make([]testworkflowsv1.StepBase, 0)
	if s.Run != nil {
		steps = append(steps, testworkflowsv1.StepBase{Run: s.Run})
	}
	if s.Execute != nil {
		steps = append(steps, testworkflowsv1.StepBase{Execute: s.Execute})
	}
	if s.Shell != "" {
		steps = append(steps, testworkflowsv1.StepBase{Shell: s.Shell})
	}
	return steps
}

func buildStepFromStepBase(s testworkflowsv1.StepBase) testworkflowsv1.Step {
	return testworkflowsv1.Step{StepBase: s}
}

func buildIndependentStepFromStepBase(s testworkflowsv1.StepBase) testworkflowsv1.IndependentStep {
	return testworkflowsv1.IndependentStep{StepBase: s}
}

func getStepList(s testworkflowsv1.Step) []testworkflowsv1.Step {
	setup := common.MapSlice(getSetupBaseSteps(s.StepBase), buildStepFromStepBase)
	return append(setup, s.Steps...)
}

func getIndependentStepList(s testworkflowsv1.IndependentStep) []testworkflowsv1.IndependentStep {
	setup := common.MapSlice(getSetupBaseSteps(s.StepBase), buildIndependentStepFromStepBase)
	return append(setup, s.Steps...)
}

func DecoupleStep(s testworkflowsv1.Step) testworkflowsv1.Step {
	// Decouple internal steps
	for i := range s.Steps {
		s.Steps[i] = DecoupleStep(s.Steps[i])
	}

	// Read the step details
	base := getStepBaseConfig(s.StepBase)
	steps := getStepList(s)

	// Ignore when there are no steps, or it's a single - not nested - step
	if len(steps) == 0 || (len(steps) == 1 && len(s.Steps) == 0) {
		return s
	}

	// Simplify a singular step
	if len(steps) == 1 {
		applyStepBaseConfig(steps[0].StepBase, base)
		return steps[0]
	}

	// Ignore when all the steps inside are already grouped
	if len(steps) == len(s.Steps) {
		return s
	}

	// Pack multiple steps in a single virtual group
	base.VirtualGroup = true

	return testworkflowsv1.Step{
		StepBase: base,
		Use:      s.Use,
		Steps:    steps,
	}
}

func DecoupleIndependentStep(s testworkflowsv1.IndependentStep) testworkflowsv1.IndependentStep {
	// Decouple internal steps
	for i := range s.Steps {
		s.Steps[i] = DecoupleIndependentStep(s.Steps[i])
	}

	// Read the step details
	base := getStepBaseConfig(s.StepBase)
	steps := getIndependentStepList(s)

	// Ignore when there are no steps, or it's a single - not nested - step
	if len(steps) == 0 || (len(steps) == 1 && len(s.Steps) == 0) {
		return s
	}

	// Simplify a singular step
	if len(steps) == 1 {
		applyStepBaseConfig(steps[0].StepBase, base)
		return steps[0]
	}

	// Ignore when all the steps inside are already grouped
	if len(steps) == len(s.Steps) {
		return s
	}

	// Pack multiple steps in a single virtual group
	base.VirtualGroup = true

	return testworkflowsv1.IndependentStep{
		StepBase: base,
		Steps:    steps,
	}
}

func DecoupleTestWorkflowSteps(t *testworkflowsv1.TestWorkflow) *testworkflowsv1.TestWorkflow {
	if t == nil {
		return nil
	}
	t.Spec.Setup = common.MapSlice(t.Spec.Setup, DecoupleStep)
	t.Spec.Steps = common.MapSlice(t.Spec.Steps, DecoupleStep)
	t.Spec.After = common.MapSlice(t.Spec.After, DecoupleStep)
	return t
}

func DecoupleTestWorkflowTemplateSteps(t *testworkflowsv1.TestWorkflowTemplate) *testworkflowsv1.TestWorkflowTemplate {
	if t == nil {
		return nil
	}
	t.Spec.Setup = common.MapSlice(t.Spec.Setup, DecoupleIndependentStep)
	t.Spec.Steps = common.MapSlice(t.Spec.Steps, DecoupleIndependentStep)
	t.Spec.After = common.MapSlice(t.Spec.After, DecoupleIndependentStep)
	return t
}
