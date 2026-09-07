// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"fmt"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// SelectFromSelf is the only selection source available today: this step's own
// previous retry attempt.
//
// Step references - `step:<id>` - need the whole workflow to resolve, which an
// Operation cannot see: it is handed one step at a time. Resolving them belongs
// with ResolveAndValidateStepIds, the existing whole-workflow walk, and is not
// wired up yet. Until it is, a reference is rejected rather than silently
// meaning "run everything".
const SelectFromSelf = "self"

// ProcessTestCases validates a step's testCases policy.
//
// It registers where the open source preset registers StubTestCases, which is
// the entire cloud-only gate for the feature: reaching this function at all
// means the deployment is entitled to it. It adds no stage of its own - the
// policy travels on the container stage that ProcessRunCommand or
// ProcessShellCommand already created.
//
// Validating here rather than in the pod is the point of doing it at processing
// time: a workflow that can never behave as written should be refused before it
// consumes a runner.
func ProcessTestCases(_ testworkflowprocessor.InternalProcessor, _ testworkflowprocessor.Intermediate, _ stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	policy := step.TestCases
	if policy == nil {
		return nil, nil
	}

	// Everything the policy can express is a statement about a report, so
	// without one there is nothing it could act on.
	if policy.Report == nil || len(policy.Report.Paths) == 0 {
		return nil, fmt.Errorf("testCases: report.paths is required, as mute, tolerate and select all read a report")
	}
	if format := policy.Report.Format; format != "" && format != "junit" {
		return nil, fmt.Errorf("testCases: report.format %q is not understood, only junit is", format)
	}
	if err := validateEnum("report.onMissing", policy.Report.OnMissing, "fail", "warn", "ignore"); err != nil {
		return nil, err
	}
	if err := validateEnum("enforce", policy.Enforce, "onFailure", "always"); err != nil {
		return nil, err
	}

	if policy.Mute != nil {
		mute := testresults.Selector{Include: policy.Mute.Include, Exclude: policy.Mute.Exclude}
		if err := mute.Validate(); err != nil {
			return nil, fmt.Errorf("testCases: mute: %w", err)
		}
	}

	if err := validateTolerance(policy.Tolerate); err != nil {
		return nil, err
	}

	if err := validateSelection(policy.Select); err != nil {
		return nil, err
	}

	return nil, nil
}

// validateTolerance rejects a requirement that can never be read as intended.
func validateTolerance(tolerance *testworkflowsv1.TestCaseTolerance) error {
	if tolerance == nil {
		return nil
	}
	for name, value := range map[string]*int32{
		"maxFailed": tolerance.MaxFailed,
		"minPassed": tolerance.MinPassed,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("testCases: tolerate.%s cannot be negative", name)
		}
	}
	for name, value := range map[string]*int32{
		"maxFailedPercent": tolerance.MaxFailedPercent,
		"minPassedPercent": tolerance.MinPassedPercent,
	} {
		if value != nil && (*value < 0 || *value > 100) {
			return fmt.Errorf("testCases: tolerate.%s must be between 0 and 100", name)
		}
	}
	if tolerance.MaxFailed == nil && tolerance.MaxFailedPercent == nil &&
		tolerance.MinPassed == nil && tolerance.MinPassedPercent == nil {
		return fmt.Errorf("testCases: tolerate is empty, so it states no requirement; remove it or set a threshold")
	}
	return nil
}

// validateSelection rejects a selection that could not narrow anything.
func validateSelection(selection *testworkflowsv1.TestCaseSelection) error {
	if selection == nil {
		return nil
	}

	switch from := strings.TrimSpace(selection.From); from {
	case "", SelectFromSelf:
	default:
		if strings.HasPrefix(from, "step:") {
			return fmt.Errorf("testCases: select.from %q is not available yet; only %q is supported today", from, SelectFromSelf)
		}
		return fmt.Errorf("testCases: select.from %q is not a known source; use %q", from, SelectFromSelf)
	}

	if _, err := testresults.ParseStatuses(selection.Status); err != nil {
		return fmt.Errorf("testCases: select.status: %w", err)
	}
	if err := validateEnum("select.empty", selection.Empty, "all", "skip", "fail"); err != nil {
		return err
	}

	filter := testresults.Selector{Include: selection.Include, Exclude: selection.Exclude}
	if err := filter.Validate(); err != nil {
		return fmt.Errorf("testCases: select: %w", err)
	}

	if selection.Write != nil && strings.TrimSpace(selection.Write.Path) == "" {
		return fmt.Errorf("testCases: select.write.path is required when write is set")
	}

	return nil
}

// validateEnum accepts an empty value, which means the field's default.
func validateEnum(field, value string, allowed ...string) error {
	if value == "" {
		return nil
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("testCases: %s %q is not one of %s", field, value, strings.Join(allowed, ", "))
}
