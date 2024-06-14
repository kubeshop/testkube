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

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

var (
	refList = []testworkflowsv1.TemplateRef{
		{Name: "official/something"},
		{Name: "official--another"},
		{Name: "official/another"},
		{Name: "something"},
	}
	refListWant = map[string]struct{}{"official--something": {}, "official--another": {}, "something": {}}
	refList2    = []testworkflowsv1.TemplateRef{
		{Name: "official/something"},
		{Name: "another"},
	}
	refList2Want      = map[string]struct{}{"official--something": {}, "another": {}}
	refList1Plus2Want = map[string]struct{}{"official--something": {}, "official--another": {}, "something": {}, "another": {}}
)

func TestGetInternalTemplateName(t *testing.T) {
	assert.Equal(t, "keep-same-name", GetInternalTemplateName("keep-same-name"))
	assert.Equal(t, "some--namespace", GetInternalTemplateName("some--namespace"))
	assert.Equal(t, "some--namespace", GetInternalTemplateName("some/namespace"))
	assert.Equal(t, "some--namespace--multiple", GetInternalTemplateName("some--namespace--multiple"))
	assert.Equal(t, "some--namespace--multiple", GetInternalTemplateName("some/namespace--multiple"))
	assert.Equal(t, "some--namespace--multiple", GetInternalTemplateName("some/namespace/multiple"))
}

func TestGetDisplayTemplateName(t *testing.T) {
	assert.Equal(t, "keep-same-name", GetDisplayTemplateName("keep-same-name"))
	assert.Equal(t, "some/namespace", GetDisplayTemplateName("some--namespace"))
	assert.Equal(t, "some/namespace", GetDisplayTemplateName("some/namespace"))
	assert.Equal(t, "some/namespace/multiple", GetDisplayTemplateName("some--namespace--multiple"))
	assert.Equal(t, "some/namespace/multiple", GetDisplayTemplateName("some/namespace--multiple"))
	assert.Equal(t, "some/namespace/multiple", GetDisplayTemplateName("some/namespace/multiple"))
}

func TestListTemplates(t *testing.T) {
	assert.Equal(t, map[string]struct{}(nil), ListTemplates(nil))
	assert.Equal(t, map[string]struct{}{}, ListTemplates(&testworkflowsv1.TestWorkflow{}))
	assert.Equal(t, refListWant, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{Use: refList},
	}))
	assert.Equal(t, refListWant, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{Setup: []testworkflowsv1.Step{{Use: refList}}},
	}))
	assert.Equal(t, refListWant, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{Use: refList}}},
	}))
	assert.Equal(t, refListWant, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{After: []testworkflowsv1.Step{{Use: refList}}},
	}))
	assert.Equal(t, map[string]struct{}{"official--something": {}}, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{After: []testworkflowsv1.Step{{Template: &refList[0]}}},
	}))
	assert.Equal(t, refListWant, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{After: []testworkflowsv1.Step{
			{Steps: []testworkflowsv1.Step{{Use: refList}}}}},
	}))
	assert.Equal(t, refList1Plus2Want, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Setup: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Use: refList}}}},
			After: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Use: refList2}}}},
		}}))
	assert.Equal(t, refList2Want, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Setup: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Use: refList2}}}},
			After: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Template: &refList2[0]}}}},
		}}))
	assert.Equal(t, refList2Want, ListTemplates(&testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Parallel: &testworkflowsv1.StepParallel{
				TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{Use: refList2},
			}}}}},
			After: []testworkflowsv1.Step{{Steps: []testworkflowsv1.Step{{Template: &refList2[0]}}}},
		}}))
}
