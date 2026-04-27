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

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func TestApplyTemplates_SecurityContext(t *testing.T) {
	tpls := map[string]*testworkflowsv1.TestWorkflowTemplate{
		"secure-template": {
			Spec: testworkflowsv1.TestWorkflowTemplateSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Container: &testworkflowsv1.ContainerConfig{
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             common.Ptr(true),
							ReadOnlyRootFilesystem:   common.Ptr(true),
							AllowPrivilegeEscalation: common.Ptr(false),
						},
					},
				},
			},
		},
	}

	t.Run("template securityContext applied to workflow", func(t *testing.T) {
		wf := &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				Use: []testworkflowsv1.TemplateRef{
					{Name: "secure-template"},
				},
				Steps: []testworkflowsv1.Step{
					{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
				},
			},
		}
		err := ApplyTemplates(wf, tpls, nil)

		assert.NoError(t, err)
		assert.NotNil(t, wf.Spec.Container)
		assert.NotNil(t, wf.Spec.Container.SecurityContext)
		assert.Equal(t, common.Ptr(true), wf.Spec.Container.SecurityContext.RunAsNonRoot)
		assert.Equal(t, common.Ptr(true), wf.Spec.Container.SecurityContext.ReadOnlyRootFilesystem)
		assert.Equal(t, common.Ptr(false), wf.Spec.Container.SecurityContext.AllowPrivilegeEscalation)
	})

	t.Run("workflow securityContext overrides template", func(t *testing.T) {
		wf := &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Container: &testworkflowsv1.ContainerConfig{
						SecurityContext: &corev1.SecurityContext{
							ReadOnlyRootFilesystem: common.Ptr(false),
							RunAsUser:              common.Ptr(int64(5000)),
						},
					},
				},
				Use: []testworkflowsv1.TemplateRef{
					{Name: "secure-template"},
				},
				Steps: []testworkflowsv1.Step{
					{StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"}},
				},
			},
		}
		err := ApplyTemplates(wf, tpls, nil)

		assert.NoError(t, err)
		assert.NotNil(t, wf.Spec.Container)
		assert.NotNil(t, wf.Spec.Container.SecurityContext)
		// Template value preserved when workflow doesn't override
		assert.Equal(t, common.Ptr(true), wf.Spec.Container.SecurityContext.RunAsNonRoot)
		assert.Equal(t, common.Ptr(false), wf.Spec.Container.SecurityContext.AllowPrivilegeEscalation)
		// Workflow value overrides template
		assert.Equal(t, common.Ptr(false), wf.Spec.Container.SecurityContext.ReadOnlyRootFilesystem)
		// Workflow adds new field
		assert.Equal(t, common.Ptr(int64(5000)), wf.Spec.Container.SecurityContext.RunAsUser)
	})

	t.Run("template securityContext applied to step", func(t *testing.T) {
		wf := &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				Steps: []testworkflowsv1.Step{
					{
						Use: []testworkflowsv1.TemplateRef{
							{Name: "secure-template"},
						},
						StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"},
					},
				},
			},
		}
		err := ApplyTemplates(wf, tpls, nil)

		assert.NoError(t, err)
		assert.NotNil(t, wf.Spec.Steps[0].Container)
		assert.NotNil(t, wf.Spec.Steps[0].Container.SecurityContext)
		assert.Equal(t, common.Ptr(true), wf.Spec.Steps[0].Container.SecurityContext.RunAsNonRoot)
		assert.Equal(t, common.Ptr(true), wf.Spec.Steps[0].Container.SecurityContext.ReadOnlyRootFilesystem)
		assert.Equal(t, common.Ptr(false), wf.Spec.Steps[0].Container.SecurityContext.AllowPrivilegeEscalation)
	})

	t.Run("step securityContext overrides template", func(t *testing.T) {
		wf := &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				Steps: []testworkflowsv1.Step{
					{
						Use: []testworkflowsv1.TemplateRef{
							{Name: "secure-template"},
						},
						StepDefaults: testworkflowsv1.StepDefaults{
							Container: &testworkflowsv1.ContainerConfig{
								SecurityContext: &corev1.SecurityContext{
									ReadOnlyRootFilesystem: common.Ptr(false),
								},
							},
						},
						StepOperations: testworkflowsv1.StepOperations{Shell: "exit 0"},
					},
				},
			},
		}
		err := ApplyTemplates(wf, tpls, nil)

		assert.NoError(t, err)
		assert.NotNil(t, wf.Spec.Steps[0].Container)
		assert.NotNil(t, wf.Spec.Steps[0].Container.SecurityContext)
		// Template values preserved
		assert.Equal(t, common.Ptr(true), wf.Spec.Steps[0].Container.SecurityContext.RunAsNonRoot)
		assert.Equal(t, common.Ptr(false), wf.Spec.Steps[0].Container.SecurityContext.AllowPrivilegeEscalation)
		// Step value overrides template
		assert.Equal(t, common.Ptr(false), wf.Spec.Steps[0].Container.SecurityContext.ReadOnlyRootFilesystem)
	})
}
