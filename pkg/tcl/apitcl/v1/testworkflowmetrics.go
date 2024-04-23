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
	"github.com/kubeshop/testkube/pkg/tcl/commontcl"
	"github.com/kubeshop/testkube/pkg/telemetry"
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
			DataSource: commontcl.GetDataSource(workflow.Spec.Content),
			Host:       commontcl.GetHostname(),
			ClusterID:  commontcl.GetClusterID(ctx, s.configMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowTemplateUsed: len(workflow.Spec.Use) != 0,
			TestWorkflowImage:        commontcl.GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: commontcl.HasWorkflowStepLike(workflow.Spec, commontcl.HasArtifacts),
			TestWorkflowKubeshopGitURI: commontcl.IsKubeshopGitURI(workflow.Spec.Content) ||
				commontcl.HasWorkflowStepLike(workflow.Spec, commontcl.HasKubeshopGitURI),
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
			DataSource: commontcl.GetDataSource(template.Spec.Content),
			Host:       commontcl.GetHostname(),
			ClusterID:  commontcl.GetClusterID(ctx, s.configMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(template.Spec.Setup) + len(template.Spec.Steps) + len(template.Spec.After)),
			TestWorkflowImage:        commontcl.GetImage(template.Spec.Container),
			TestWorkflowArtifactUsed: commontcl.HasTemplateStepLike(template.Spec, commontcl.HasTemplateArtifacts),
			TestWorkflowKubeshopGitURI: commontcl.IsKubeshopGitURI(template.Spec.Content) ||
				commontcl.HasTemplateStepLike(template.Spec, commontcl.HasTemplateKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}
