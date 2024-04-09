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

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
	}

	var dataSource string
	if len(workflow.Spec.Content.Files) != 0 {
		dataSource = "files"
	} else if workflow.Spec.Content.Git != nil {
		dataSource = "git"
	}

	hasArtifacts := false
	for _, step := range workflow.Spec.Steps {
		if step.Artifacts != nil {
			hasArtifacts = true
			break
		}
	}

	image := ""
	if workflow.Spec.Container != nil {
		image = workflow.Spec.Container.Image
	}

	isKubeshopGitURI := false
	if workflow.Spec.Content != nil && workflow.Spec.Content.Git != nil {
		if strings.Contains(workflow.Spec.Content.Git.Uri, "kubeshop") {
			isKubeshopGitURI = true
		}
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: dataSource,
			Host:       host,
			ClusterID:  clusterID,
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Steps)),
			TestWorkflowTemplateUsed:   len(workflow.Spec.Use) != 0,
			TestWorkflowImage:          image,
			TestWorkflowArtifactUsed:   hasArtifacts,
			TestWorkflowKubeshopGitURI: isKubeshopGitURI,
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

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
	}

	var dataSource string
	if template.Spec.Content != nil && len(template.Spec.Content.Files) != 0 {
		dataSource = "files"
	} else if template.Spec.Content.Git != nil {
		dataSource = "git"
	}

	hasArtifacts := false
	for _, step := range template.Spec.Steps {
		if step.Artifacts != nil {
			hasArtifacts = true
			break
		}
	}

	image := ""
	if template.Spec.Container != nil {
		image = template.Spec.Container.Image
	}

	isKubeshopGitURI := false
	if template.Spec.Content != nil && template.Spec.Content.Git != nil {
		if strings.Contains(template.Spec.Content.Git.Uri, "kubeshop") {
			isKubeshopGitURI = true
		}
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow_template", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: dataSource,
			Host:       host,
			ClusterID:  clusterID,
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(template.Spec.Steps)),
			TestWorkflowImage:          image,
			TestWorkflowArtifactUsed:   hasArtifacts,
			TestWorkflowKubeshopGitURI: isKubeshopGitURI,
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

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
	}

	var dataSource string
	if len(workflow.Spec.Content.Files) != 0 {
		dataSource = "files"
	} else if workflow.Spec.Content.Git != nil {
		dataSource = "git"
	}

	hasArtifacts := false
	for _, step := range workflow.Spec.Steps {
		if step.Artifacts != nil {
			hasArtifacts = true
			break
		}
	}

	image := ""
	if workflow.Spec.Container != nil {
		image = workflow.Spec.Container.Image
	}
	isKubeshopGitURI := false
	if workflow.Spec.Content != nil && workflow.Spec.Content.Git != nil {
		if strings.Contains(workflow.Spec.Content.Git.Uri, "kubeshop") {
			isKubeshopGitURI = true
		}
	}

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: dataSource,
			Host:       host,
			ClusterID:  clusterID,
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Steps)),
			TestWorkflowImage:          image,
			TestWorkflowArtifactUsed:   hasArtifacts,
			TestWorkflowKubeshopGitURI: isKubeshopGitURI,
		},
	})

	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}
