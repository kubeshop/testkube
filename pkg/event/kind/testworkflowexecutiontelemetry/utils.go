package testworkflowexecutiontelemetry

import (
	"context"
	"os"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
)

type telemetryWorkflowDataSourceType string

const (
	telemetryWorkflowDataSourceGitType   telemetryWorkflowDataSourceType = "git"
	telemetryWorkflowDataSourceFilesType telemetryWorkflowDataSourceType = "files"
)

// GetImage returns the image of the container
func GetImage(container *testworkflowsv1.ContainerConfig) string {
	if container != nil {
		return container.Image
	}
	return ""
}

func HasWorkflowStepLike(spec testworkflowsv1.TestWorkflowSpec, fn func(step testworkflowsv1.Step) bool) bool {
	return HasStepLike(spec.Setup, fn) || HasStepLike(spec.Steps, fn) || HasStepLike(spec.After, fn)
}

func HasTemplateStepLike(spec testworkflowsv1.TestWorkflowTemplateSpec, fn func(step testworkflowsv1.IndependentStep) bool) bool {
	return HasIndependentStepLike(spec.Setup, fn) || HasIndependentStepLike(spec.Steps, fn) || HasIndependentStepLike(spec.After, fn)
}

func HasStepLike(steps []testworkflowsv1.Step, fn func(step testworkflowsv1.Step) bool) bool {
	for _, step := range steps {
		if fn(step) || HasStepLike(step.Setup, fn) || HasStepLike(step.Steps, fn) {
			return true
		}
	}
	return false
}

func HasIndependentStepLike(steps []testworkflowsv1.IndependentStep, fn func(step testworkflowsv1.IndependentStep) bool) bool {
	for _, step := range steps {
		if fn(step) || HasIndependentStepLike(step.Setup, fn) || HasIndependentStepLike(step.Steps, fn) {
			return true
		}
	}
	return false
}

// HasArtifacts checks if the test workflow steps have artifacts
func HasArtifacts(step testworkflowsv1.Step) bool {
	return step.Artifacts != nil
}

// HasTemplateArtifacts checks if the test workflow steps have artifacts
func HasTemplateArtifacts(step testworkflowsv1.IndependentStep) bool {
	return step.Artifacts != nil
}

// HasKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func HasKubeshopGitURI(step testworkflowsv1.Step) bool {
	return IsKubeshopGitURI(step.Content)
}

// HasTemplateKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func HasTemplateKubeshopGitURI(step testworkflowsv1.IndependentStep) bool {
	return IsKubeshopGitURI(step.Content)
}

// IsKubeshopGitURI checks if the content has a git URI that contains "kubeshop"
func IsKubeshopGitURI(content *testworkflowsv1.Content) bool {
	return content != nil && content.Git != nil && strings.Contains(content.Git.Uri, "kubeshop")
}

// GetDataSource returns the data source of the content
func GetDataSource(content *testworkflowsv1.Content) string {
	if content != nil {
		if content.Git != nil {
			return string(telemetryWorkflowDataSourceGitType)
		} else if len(content.Files) != 0 {
			return string(telemetryWorkflowDataSourceFilesType)
		}
	}
	return ""
}

// GetHostname returns the hostname
func GetHostname() string {
	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
		return ""
	}
	return host
}

// GetClusterID returns the cluster id
func GetClusterID(ctx context.Context, configMap configRepo.Repository) string {
	clusterID, err := configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
		return ""
	}
	return clusterID
}

// triggeredByBucket collapses the actor / interface enums from the execution's
// RunningContext into a coarse activation signal used for value dashboards.
// Returns "unknown" when the running context is missing or carries an
// unrecognised combination.
func triggeredByBucket(actorType, interfaceType string) string {
	switch testkube.TestWorkflowRunningContextActorType(actorType) {
	case testkube.CRON_TestWorkflowRunningContextActorType:
		return "scheduled"
	case testkube.TESTTRIGGER_TestWorkflowRunningContextActorType:
		return "event-triggered"
	case testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType,
		testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType:
		return "composite"
	case testkube.PROGRAM_TestWorkflowRunningContextActorType:
		return "automation"
	case testkube.USER_TestWorkflowRunningContextActorType:
		switch testkube.TestWorkflowRunningContextInterfaceType(interfaceType) {
		case testkube.CICD_TestWorkflowRunningContextInterfaceType:
			return "ci-cd"
		case testkube.UI_TestWorkflowRunningContextInterfaceType:
			return "human-ui"
		case testkube.CLI_TestWorkflowRunningContextInterfaceType:
			return "human-cli"
		case testkube.API_TestWorkflowRunningContextInterfaceType:
			return "api"
		}
	}
	return "unknown"
}
