package v1

import (
	"context"
	"os"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
)

func (s *apiTCL) sendCreateWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
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

	out, err := telemetry.SendCreateEvent("testkube_api_create_test_workflow", telemetry.CreateParams{
		AppVersion: version.Version,
		DataSource: dataSource,
		Host:       host,
		ClusterID:  clusterID,
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event", "output", out)
	}
}

func (s *apiTCL) sendCreateWorkflowTemplateTelemetry(ctx context.Context, template *testworkflowsv1.TestWorkflowTemplate) {
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
	if len(template.Spec.Content.Files) != 0 {
		dataSource = "files"
	} else if template.Spec.Content.Git != nil {
		dataSource = "git"
	}

	out, err := telemetry.SendCreateEvent("testkube_api_create_test_workflow_template", telemetry.CreateParams{
		AppVersion: version.Version,
		DataSource: dataSource,
		Host:       host,
		ClusterID:  clusterID,
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}

func (s *apiTCL) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
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

	out, err := telemetry.SendCreateEvent("testkube_api_run_test_workflow", telemetry.CreateParams{
		AppVersion: version.Version,
		DataSource: dataSource,
		Host:       host,
		ClusterID:  clusterID,
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}
