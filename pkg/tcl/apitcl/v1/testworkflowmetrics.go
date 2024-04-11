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
	"os"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/log"
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
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowTemplateUsed:   len(workflow.Spec.Use) != 0,
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasWorkflowStepLike(workflow.Spec, hasArtifacts),
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(workflow.Spec.Content) || hasWorkflowStepLike(workflow.Spec, hasKubeshopGitURI),
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
			DataSource: getDataSource(template.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(template.Spec.Setup) + len(template.Spec.Steps) + len(template.Spec.After)),
			TestWorkflowImage:          getImage(template.Spec.Container),
			TestWorkflowArtifactUsed:   hasTemplateStepLike(template.Spec, hasTemplateArtifacts),
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(template.Spec.Content) || hasTemplateStepLike(template.Spec, hasTemplateKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}

func (s *apiTCL) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
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

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasWorkflowStepLike(workflow.Spec, hasArtifacts),
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(workflow.Spec.Content) || hasWorkflowStepLike(workflow.Spec, hasKubeshopGitURI),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}

// getClusterID returns the cluster id
func (s *apiTCL) getClusterID(ctx context.Context) string {
	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
		return ""
	}
	return clusterID
}

// getImage returns the image of the container
func getImage(container *testworkflowsv1.ContainerConfig) string {
	if container != nil {
		return container.Image
	}
	return ""
}

func hasWorkflowStepLike(spec testworkflowsv1.TestWorkflowSpec, fn func(step testworkflowsv1.Step) bool) bool {
	return hasStepLike(spec.Setup, fn) || hasStepLike(spec.Steps, fn) || hasStepLike(spec.After, fn)
}

func hasTemplateStepLike(spec testworkflowsv1.TestWorkflowTemplateSpec, fn func(step testworkflowsv1.IndependentStep) bool) bool {
	return hasIndependentStepLike(spec.Setup, fn) || hasIndependentStepLike(spec.Steps, fn) || hasIndependentStepLike(spec.After, fn)
}

func hasStepLike(steps []testworkflowsv1.Step, fn func(step testworkflowsv1.Step) bool) bool {
	for _, step := range steps {
		if fn(step) || hasStepLike(step.Setup, fn) || hasStepLike(step.Steps, fn) {
			return true
		}
	}
	return false
}

func hasIndependentStepLike(steps []testworkflowsv1.IndependentStep, fn func(step testworkflowsv1.IndependentStep) bool) bool {
	for _, step := range steps {
		if fn(step) || hasIndependentStepLike(step.Setup, fn) || hasIndependentStepLike(step.Steps, fn) {
			return true
		}
	}
	return false
}

// hasArtifacts checks if the test workflow steps have artifacts
func hasArtifacts(step testworkflowsv1.Step) bool {
	return step.Artifacts != nil
}

// hasTemplateArtifacts checks if the test workflow steps have artifacts
func hasTemplateArtifacts(step testworkflowsv1.IndependentStep) bool {
	return step.Artifacts != nil
}

// hasKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func hasKubeshopGitURI(step testworkflowsv1.Step) bool {
	return isKubeshopGitURI(step.Content)
}

// hasTemplateKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func hasTemplateKubeshopGitURI(step testworkflowsv1.IndependentStep) bool {
	return isKubeshopGitURI(step.Content)
}

// isKubeshopGitURI checks if the content has a git URI that contains "kubeshop"
func isKubeshopGitURI(content *testworkflowsv1.Content) bool {
	return content != nil && content.Git != nil && strings.Contains(content.Git.Uri, "kubeshop")
}

// getDataSource returns the data source of the content
func getDataSource(content *testworkflowsv1.Content) string {
	if content != nil {
		if content.Git != nil {
			return "git"
		} else if len(content.Files) != 0 {
			return "files"
		}
	}
	return ""
}

// getHostname returns the hostname
func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
		return ""
	}
	return host
}
