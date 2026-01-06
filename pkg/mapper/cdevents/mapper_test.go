package cdevents

import (
	"testing"

	cdevents "github.com/cdevents/sdk-go/pkg/api"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapTestkubeEventQueuedTestWorkflowTestToCDEvent(t *testing.T) {

	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "test-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Test 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Template: &testkube.TestWorkflowTemplateRef{
								Name: "official--k6--v1",
							},
						},
					},
				},
			},

			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventQueuedTestWorkflowTestToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestCaseRunQueuedEvent)
	assert.Equal(t, true, ok)

	testID := cde.Subject.Content.TestCase.Id
	if testID != "Test 1" {
		t.Errorf("Unexpected test case id: %s", testID)
	}

	testType := cde.Subject.Content.TestCase.Type
	if testType != "performance" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/test-workflows/Test 1" {
		t.Errorf("Unexpected test case uri: %s", testURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != defaultNamespace {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}
}

func TestMapTestkubeEventQueuedTestWorkflowTestSuiteToCDEvent(t *testing.T) {

	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "suite-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Suite 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Execute: &testkube.TestWorkflowStepExecute{
								Workflows: []testkube.TestWorkflowStepExecuteTestWorkflowRef{
									{
										Name: "test-1",
									},
								},
							},
						},
					},
				},
			},
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventQueuedTestWorkflowTestSuiteToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestSuiteRunQueuedEvent)
	assert.Equal(t, true, ok)

	suiteID := cde.Subject.Content.TestSuite.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteURI := cde.Subject.Content.TestSuite.Url
	if suiteURI != "/test-workflows/Suite 1" {
		t.Errorf("Unexpected test case uri: %s", suiteURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != "default" {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}
}

func TestMapTestkubeEventStartTestWorkflowTestToCDEvent(t *testing.T) {

	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "test-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Test 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Template: &testkube.TestWorkflowTemplateRef{
								Name: "official--k6--v1",
							},
						},
					},
				},
			},
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventStartTestWorkflowTestToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestCaseRunStartedEvent)
	assert.Equal(t, true, ok)

	testID := cde.Subject.Content.TestCase.Id
	if testID != "Test 1" {
		t.Errorf("Unexpected test case id: %s", testID)
	}

	testType := cde.Subject.Content.TestCase.Type
	if testType != "performance" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/test-workflows/Test 1" {
		t.Errorf("Unexpected test case uri: %s", testURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != defaultNamespace {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}

}

func TestMapTestkubeEventStartTestWorkflowTestSuiteToCDEvent(t *testing.T) {

	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "suite-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Suite 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Execute: &testkube.TestWorkflowStepExecute{
								Workflows: []testkube.TestWorkflowStepExecuteTestWorkflowRef{
									{
										Name: "test-1",
									},
								},
							},
						},
					},
				},
			},
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventStartTestWorkflowTestSuiteToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestSuiteRunStartedEvent)
	assert.Equal(t, true, ok)

	suiteID := cde.Subject.Content.TestSuite.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteURI := cde.Subject.Content.TestSuite.Uri
	if suiteURI != "/test-workflows/Suite 1" {
		t.Errorf("Unexpected test case uri: %s", suiteURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != "default" {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}
}

func TestMapTestkubeEventFinishTestWorkflowTestToCDEvent(t *testing.T) {

	status := testkube.FAILED_TestWorkflowStatus
	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "test-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Test 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Template: &testkube.TestWorkflowTemplateRef{
								Name: "official--k6--v1",
							},
						},
					},
				},
			},
			Result: &testkube.TestWorkflowResult{
				Status: &status,
				Steps: map[string]testkube.TestWorkflowStepResult{
					"first": {
						ErrorMessage: "fake",
					},
				},
			},
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventFinishTestWorkflowTestToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestCaseRunFinishedEvent)
	assert.Equal(t, true, ok)

	testID := cde.Subject.Content.TestCase.Id
	if testID != "Test 1" {
		t.Errorf("Unexpected test case id: %s", testID)
	}

	testType := cde.Subject.Content.TestCase.Type
	if testType != "performance" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/test-workflows/Test 1" {
		t.Errorf("Unexpected test case uri: %s", testURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != defaultNamespace {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}

	outcome := cde.Subject.Content.Outcome
	if outcome != "fail" {
		t.Errorf("Unexpected outcome: %s", outcome)
	}

	reason := cde.Subject.Content.Reason
	if reason != "fake" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}

func TestMapTestkubeEventFinishTestWorkflowTestSuiteToCDEvent(t *testing.T) {

	status := testkube.FAILED_TestWorkflowStatus
	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:        "1",
			Name:      "suite-1",
			Namespace: "default",
			Workflow: &testkube.TestWorkflow{
				Name: "Suite 1",
			},
			ResolvedWorkflow: &testkube.TestWorkflow{
				Spec: &testkube.TestWorkflowSpec{
					Steps: []testkube.TestWorkflowStep{
						{
							Execute: &testkube.TestWorkflowStepExecute{
								Workflows: []testkube.TestWorkflowStepExecuteTestWorkflowRef{
									{
										Name: "test-1",
									},
								},
							},
						},
					},
				},
			},
			Result: &testkube.TestWorkflowResult{
				Status: &status,
				Steps: map[string]testkube.TestWorkflowStepResult{
					"first": {
						ErrorMessage: "fake",
					},
				},
			},
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
				},
			},
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventFinishTestWorkflowTestSuiteToCDEvent(event, clusterID, defaultNamespace, "")
	if err != nil {
		t.Errorf("Error mapping event: %v", err)
		return
	}

	subjectID := ev.GetSubjectId()
	if subjectID != "1" {
		t.Errorf("Unexpected subject ID: %s", subjectID)
	}

	subjectSource := ev.GetSubjectSource()
	if subjectSource != clusterID {
		t.Errorf("Unexpected subject source: %s", subjectSource)
	}

	source := ev.GetSource()
	if source != clusterID {
		t.Errorf("Unexpected source: %s", source)
	}

	cde, ok := ev.(*cdevents.TestSuiteRunFinishedEvent)
	assert.Equal(t, true, ok)

	suiteID := cde.Subject.Content.TestSuite.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteURI := cde.Subject.Content.TestSuite.Uri
	if suiteURI != "/test-workflows/Suite 1" {
		t.Errorf("Unexpected test case uri: %s", suiteURI)
	}

	envID := cde.Subject.Content.Environment.Id
	if envID != "default" {
		t.Errorf("Unexpected environment id: %s", envID)
	}

	envSource := cde.Subject.Content.Environment.Source
	if envSource != clusterID {
		t.Errorf("Unexpected environment source: %s", envSource)
	}

	outcome := cde.Subject.Content.Outcome
	if outcome != "fail" {
		t.Errorf("Unexpected outcome: %s", outcome)
	}

	reason := cde.Subject.Content.Reason
	if reason != "fake" {
		t.Errorf("Unexpected reason: %s", reason)
	}
}
