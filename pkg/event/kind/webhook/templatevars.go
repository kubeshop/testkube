package webhook

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type TemplateVars struct {
	testkube.Event
	ExecutionURL     string
	ExecutionCommand string
	ArtifactURL      string
	ArtifactCommand  string
	LogsURL          string
	LogsCommand      string
	Config           map[string]string
}

func NewTemplateVars(event testkube.Event, dashboardURI, orgID, envID string, config map[string]string) TemplateVars {
	vars := TemplateVars{
		Event:  event,
		Config: config,
	}

	switch {
	case event.TestExecution != nil:
		vars.ExecutionCommand = fmt.Sprintf("kubectl testkube get execution %s", event.TestExecution.Id)
		vars.ArtifactCommand = fmt.Sprintf("kubectl testkube get artifacts %s", event.TestExecution.Id)
		vars.LogsCommand = fmt.Sprintf("kubectl testkube get execution %s --logs-only", event.TestExecution.Id)
	case event.TestSuiteExecution != nil:
		vars.ExecutionCommand = fmt.Sprintf("kubectl testkube get testsuiteexecution %s", event.TestSuiteExecution.Id)
	case event.TestWorkflowExecution != nil:
		vars.ExecutionCommand = fmt.Sprintf("kubectl testkube get testworkflowexecution %s", event.TestWorkflowExecution.Id)
		vars.ArtifactCommand = fmt.Sprintf("kubectl testkube get artifacts %s", event.TestWorkflowExecution.Id)
		vars.LogsCommand = fmt.Sprintf("kubectl testkube get testworkflowexecution %s", event.TestWorkflowExecution.Id)
	}

	if dashboardURI == "" || orgID == "" || envID == "" {
		return vars
	}

	switch {
	case event.TestExecution != nil:
		vars.ExecutionURL = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard/tests/%s/executions/%s", dashboardURI, orgID, envID, event.TestExecution.TestName, event.TestExecution.Id)
		vars.ArtifactURL = fmt.Sprintf("%s/artifacts", vars.ExecutionURL)
		vars.LogsURL = fmt.Sprintf("%s/log-output", vars.ExecutionURL)
	case event.TestSuiteExecution != nil:
		vars.ExecutionURL = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard/test-suites/%s/executions/%s", dashboardURI, orgID, envID, event.TestSuiteExecution.TestSuite.Name, event.TestSuiteExecution.Id)
	case event.TestWorkflowExecution != nil:
		vars.ExecutionURL = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard/test-workflows/%s/executions/%s", dashboardURI, orgID, envID, event.TestWorkflowExecution.Workflow.Name, event.TestWorkflowExecution.Id)
		vars.ArtifactURL = fmt.Sprintf("%s/artifacts", vars.ExecutionURL)
		vars.LogsURL = fmt.Sprintf("%s/log-output", vars.ExecutionURL)
	}

	return vars
}
