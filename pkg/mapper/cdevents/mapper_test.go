package cdevents

import (
	"errors"
	"testing"

	cdevents "github.com/cdevents/sdk-go/pkg/api"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapTestkubeEventQueuedTestToCDEvent(t *testing.T) {
	t.Parallel()

	event := testkube.Event{
		TestExecution: &testkube.Execution{
			Id:            "1",
			Name:          "test-1",
			TestName:      "Test 1",
			TestType:      "ginkgo/test",
			TestNamespace: "default",
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
			TestSuiteName: "Suite 1",
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventQueuedTestToCDEvent(event, clusterID, defaultNamespace, "")
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
	if testType != "unit" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/tests/Test 1" {
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

	triggerType := cde.Subject.Content.Trigger.Type
	if triggerType != "schedule" {
		t.Errorf("Unexpected trigger type: %s", triggerType)
	}

	suiteID := cde.Subject.Content.TestSuiteRun.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteSource := cde.Subject.Content.TestSuiteRun.Source
	if suiteSource != clusterID {
		t.Errorf("Unexpected test suite source: %s", suiteSource)
	}
}

func TestMapTestkubeEventStatTestToCDEvent(t *testing.T) {
	t.Parallel()

	event := testkube.Event{
		TestExecution: &testkube.Execution{
			Id:            "1",
			Name:          "test-1",
			TestName:      "Test 1",
			TestType:      "ginkgo/test",
			TestNamespace: "default",
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
			TestSuiteName: "Suite 1",
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventStartTestToCDEvent(event, clusterID, defaultNamespace, "")
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
	if testType != "unit" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/tests/Test 1" {
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

	triggerType := cde.Subject.Content.Trigger.Type
	if triggerType != "schedule" {
		t.Errorf("Unexpected trigger type: %s", triggerType)
	}

	suiteID := cde.Subject.Content.TestSuiteRun.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteSource := cde.Subject.Content.TestSuiteRun.Source
	if suiteSource != clusterID {
		t.Errorf("Unexpected test suite source: %s", suiteSource)
	}
}

func TestMapTestkubeEventFinishTestToCDEvent(t *testing.T) {
	t.Parallel()

	result := testkube.NewErrorExecutionResult(errors.New("fake"))
	event := testkube.Event{
		TestExecution: &testkube.Execution{
			Id:            "1",
			Name:          "test-1",
			TestName:      "Test 1",
			TestType:      "ginkgo/test",
			TestNamespace: "default",
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
			TestSuiteName:   "Suite 1",
			ExecutionResult: &result,
		},
	}
	clusterID := "cluster-1"
	defaultNamespace := "default"

	ev, err := MapTestkubeEventFinishTestToCDEvent(event, clusterID, defaultNamespace, "")
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
	if testType != "unit" {
		t.Errorf("Unexpected test case type: %s", testType)
	}

	testURI := cde.Subject.Content.TestCase.Uri
	if testURI != "/tests/Test 1" {
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

	suiteID := cde.Subject.Content.TestSuiteRun.Id
	if suiteID != "Suite 1" {
		t.Errorf("Unexpected test suite id: %s", suiteID)
	}

	suiteSource := cde.Subject.Content.TestSuiteRun.Source
	if suiteSource != clusterID {
		t.Errorf("Unexpected test suite source: %s", suiteSource)
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

func TestMapTestkubeEventQueuedTestSuiteToCDEvent(t *testing.T) {
	t.Parallel()

	event := testkube.Event{
		TestSuiteExecution: &testkube.TestSuiteExecution{
			Id:   "1",
			Name: "suite-1",
			TestSuite: &testkube.ObjectRef{
				Namespace: "default",
				Name:      "Suite 1",
			},
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
		},
	}
	clusterID := "cluster-1"

	ev, err := MapTestkubeEventQueuedTestSuiteToCDEvent(event, clusterID, "")
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
	if suiteURI != "/test-suites/executions/Suite 1" {
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

	triggerType := cde.Subject.Content.Trigger.Type
	if triggerType != "schedule" {
		t.Errorf("Unexpected trigger type: %s", triggerType)
	}
}

func TestMapTestkubeEventStartTestSuiteToCDEvent(t *testing.T) {
	t.Parallel()

	event := testkube.Event{
		TestSuiteExecution: &testkube.TestSuiteExecution{
			Id:   "1",
			Name: "suite-1",
			TestSuite: &testkube.ObjectRef{
				Namespace: "default",
				Name:      "Suite 1",
			},
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
		},
	}
	clusterID := "cluster-1"

	ev, err := MapTestkubeEventStartTestSuiteToCDEvent(event, clusterID, "")
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
	if suiteURI != "/test-suites/Suite 1" {
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

	triggerType := cde.Subject.Content.Trigger.Type
	if triggerType != "schedule" {
		t.Errorf("Unexpected trigger type: %s", triggerType)
	}
}

func TestMapTestkubeEventFinishTestSuiteToCDEvent(t *testing.T) {
	t.Parallel()

	execution := testkube.NewFailedExecution(errors.New("fake"))
	event := testkube.Event{
		TestSuiteExecution: &testkube.TestSuiteExecution{
			Id:   "1",
			Name: "suite-1",
			TestSuite: &testkube.ObjectRef{
				Namespace: "default",
				Name:      "Suite 1",
			},
			RunningContext: &testkube.RunningContext{
				Type_: "scheduler",
			},
			Status: testkube.TestSuiteExecutionStatusFailed,
			ExecuteStepResults: []testkube.TestSuiteBatchStepExecutionResult{
				{
					Execute: []testkube.TestSuiteStepExecutionResult{
						{
							Execution: &execution,
						},
					},
				},
			},
		},
	}
	clusterID := "cluster-1"

	ev, err := MapTestkubeEventFinishTestSuiteToCDEvent(event, clusterID, "")
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
	if suiteURI != "/test-suites/Suite 1" {
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
