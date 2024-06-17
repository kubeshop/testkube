// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"context"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows"
	"github.com/kubeshop/testkube/pkg/version"
)

func (s *apiTCL) sendCreateWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: testworkflows.GetDataSource(workflow.Spec.Content),
			Host:       testworkflows.GetHostname(),
			ClusterID:  testworkflows.GetClusterID(ctx, s.configMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowTemplateUsed: len(workflow.Spec.Use) != 0,
			TestWorkflowImage:        testworkflows.GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: testworkflows.HasWorkflowStepLike(workflow.Spec, testworkflows.HasArtifacts),
			TestWorkflowKubeshopGitURI: testworkflows.IsKubeshopGitURI(workflow.Spec.Content) ||
				testworkflows.HasWorkflowStepLike(workflow.Spec, testworkflows.HasKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event", "output", out)
	}
}

func (s *apiTCL) sendCreateWorkflowTemplateTelemetry(ctx context.Context, template *testworkflowsv1.TestWorkflowTemplate) {
	if template == nil {
		log.DefaultLogger.Debug("empty template passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow_template", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: testworkflows.GetDataSource(template.Spec.Content),
			Host:       testworkflows.GetHostname(),
			ClusterID:  testworkflows.GetClusterID(ctx, s.configMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(template.Spec.Setup) + len(template.Spec.Steps) + len(template.Spec.After)),
			TestWorkflowImage:        testworkflows.GetImage(template.Spec.Container),
			TestWorkflowArtifactUsed: testworkflows.HasTemplateStepLike(template.Spec, testworkflows.HasTemplateArtifacts),
			TestWorkflowKubeshopGitURI: testworkflows.IsKubeshopGitURI(template.Spec.Content) ||
				testworkflows.HasTemplateStepLike(template.Spec, testworkflows.HasTemplateKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}
