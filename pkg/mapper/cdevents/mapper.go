package cdevents

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	cdevents "github.com/cdevents/sdk-go/pkg/api"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestkubeEventToCDEvent maps OpenAPI spec Event to CDEvent CDEventReader
func MapTestkubeEventToCDEvent(tkEvent testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	if tkEvent.Type_ == nil {
		return nil, errors.New("empty event type")
	}

	switch *tkEvent.Type_ {
	case *testkube.EventStartTest:
		return MapTestkubeEventStartTestToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
	case *testkube.EventEndTestAborted, *testkube.EventEndTestFailed, *testkube.EventEndTestTimeout, *testkube.EventEndTestSuccess:
		return MapTestkubeEventFinishTestToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
	case *testkube.EventStartTestSuite:
		return MapTestkubeEventStartTestSuiteToCDEvent(tkEvent, clusterID, dashboardURI)
	case *testkube.EventEndTestSuiteAborted, *testkube.EventEndTestSuiteFailed, *testkube.EventEndTestSuiteTimeout, *testkube.EventEndTestSuiteSuccess:
		return MapTestkubeEventFinishTestSuiteToCDEvent(tkEvent, clusterID, dashboardURI)
	}

	return nil, fmt.Errorf("not supported event type %s", tkEvent.Type_)
}

// MapTestkubeEventQueuedTestToCDEvent maps OpenAPI spec Queued Test Event to CDEvent CDEventReader
func MapTestkubeEventQueuedTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunQueuedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestExecution != nil {
		ev.SetSubjectId(event.TestExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestExecution != nil {
		ev.SetSubjectTestCase(&cdevents.TestCaseRunQueuedSubjectContentTestCase{
			Id:   event.TestExecution.TestName,
			Type: MapTestkubeTestTypeToCDEventTestCaseType(event.TestExecution.TestType),
			Uri:  fmt.Sprintf("%s/tests/%s", dashboardURI, event.TestExecution.TestName),
		})

		namespace := event.TestExecution.TestNamespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestExecution.RunningContext != nil {
			ev.SetSubjectTrigger(&cdevents.TestCaseRunQueuedSubjectContentTrigger{
				Type: MapTestkubeRunningContextTypeToCDEventTiggerType(event.TestExecution.RunningContext.Type_),
			})
		}

		if event.TestExecution.TestSuiteName != "" {
			ev.SetSubjectTestSuiteRun(&cdevents.Reference{
				Id:     event.TestExecution.TestSuiteName,
				Source: clusterID,
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventStartTestToCDEvent maps OpenAPI spec Start Test Event to CDEvent CDEventReader
func MapTestkubeEventStartTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunStartedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestExecution != nil {
		ev.SetSubjectId(event.TestExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestExecution != nil {
		ev.SetSubjectTestCase(&cdevents.TestCaseRunStartedSubjectContentTestCase{
			Id:   event.TestExecution.TestName,
			Type: MapTestkubeTestTypeToCDEventTestCaseType(event.TestExecution.TestType),
			Uri:  fmt.Sprintf("%s/tests/%s", dashboardURI, event.TestExecution.TestName),
		})

		namespace := event.TestExecution.TestNamespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestExecution.RunningContext != nil {
			ev.SetSubjectTrigger(&cdevents.TestCaseRunStartedSubjectContentTrigger{
				Type: MapTestkubeRunningContextTypeToCDEventTiggerType(event.TestExecution.RunningContext.Type_),
			})
		}

		if event.TestExecution.TestSuiteName != "" {
			ev.SetSubjectTestSuiteRun(&cdevents.Reference{
				Id:     event.TestExecution.TestSuiteName,
				Source: clusterID,
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventFinishTestToCDEvent maps OpenAPI spec Failed, Aborted, Timeout, Success Test Event to CDEvent CDEventReader
func MapTestkubeEventFinishTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunFinishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestExecution != nil {
		ev.SetSubjectId(event.TestExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestExecution != nil {
		ev.SetSubjectTestCase(&cdevents.TestCaseRunFinishedSubjectContentTestCase{
			Id:   event.TestExecution.TestName,
			Type: MapTestkubeTestTypeToCDEventTestCaseType(event.TestExecution.TestType),
			Uri:  fmt.Sprintf("%s/tests/%s", dashboardURI, event.TestExecution.TestName),
		})

		namespace := event.TestExecution.TestNamespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestExecution.IsAborted() || event.TestExecution.IsTimeout() {
			ev.SetSubjectOutcome("cancel")
			if event.TestExecution.ExecutionResult != nil {
				ev.SetSubjectReason(event.TestExecution.ExecutionResult.ErrorMessage)
			}
		}

		if event.TestExecution.IsFailed() {
			ev.SetSubjectOutcome("fail")
			if event.TestExecution.ExecutionResult != nil {
				ev.SetSubjectReason(event.TestExecution.ExecutionResult.ErrorMessage)
			}
		}

		if event.TestExecution.IsPassed() {
			ev.SetSubjectOutcome("pass")
		}

		if event.TestExecution.TestSuiteName != "" {
			ev.SetSubjectTestSuiteRun(&cdevents.Reference{
				Id:     event.TestExecution.TestSuiteName,
				Source: clusterID,
			})
		}
	}

	return ev, nil
}

// MapTestkubeArtifactToCDEvent maps OpenAPI spec Artifact to CDEvent CDEventReader
func MapTestkubeArtifactToCDEvent(execution *testkube.Execution, clusterID, path, format, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestOutputPublishedEvent()
	if err != nil {
		return nil, err
	}

	ev.SetSubjectId(filepath.Join(execution.Name, path))
	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	ev.SetSubjectTestCaseRun(&cdevents.Reference{
		Id:     execution.Id,
		Source: clusterID,
	})

	ev.SetSubjectFormat(format)
	ev.SetSubjectOutputType(MapMimeTypeToCDEventOutputType(format))
	ev.SetSubjectUri(fmt.Sprintf("%s/tests/executions/%s/execution/%s", dashboardURI, execution.TestName, execution.Id))

	return ev, nil
}

// MapTestkubeLogToCDEvent maps OpenAPI spec log to CDEvent CDEventReader
func MapTestkubeLogToCDEvent(event testkube.Event, clusterID, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestOutputPublishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestExecution != nil {
		ev.SetSubjectId(event.TestExecution.Id + "-log")
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)

	if event.TestExecution != nil {
		ev.SetSubjectTestCaseRun(&cdevents.Reference{
			Id:     event.TestExecution.Id,
			Source: clusterID,
		})
	}

	ev.SetSubjectFormat("text/x-uri")
	ev.SetSubjectOutputType("log")
	if event.TestExecution != nil {
		ev.SetSubjectUri(fmt.Sprintf("%s/tests/%s/executions/%s/log-output", dashboardURI,
			event.TestExecution.TestName, event.TestExecution.Id))
	}

	return ev, nil
}

// MapTestkubeEventQueuedTestSuiteToCDEvent maps OpenAPI spec Queued Test Suite Event to CDEvent CDEventReader
func MapTestkubeEventQueuedTestSuiteToCDEvent(event testkube.Event, clusterID, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunQueuedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestSuiteExecution != nil {
		ev.SetSubjectId(event.TestSuiteExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestSuiteExecution != nil {
		if event.TestSuiteExecution.TestSuite != nil {
			ev.SetSubjectTestSuite(&cdevents.TestSuiteRunQueuedSubjectContentTestSuite{
				Id:  event.TestSuiteExecution.TestSuite.Name,
				Url: fmt.Sprintf("%s/test-suites/executions/%s", dashboardURI, event.TestSuiteExecution.TestSuite.Name),
			})

			ev.SetSubjectEnvironment(&cdevents.Reference{
				Id:     event.TestSuiteExecution.TestSuite.Namespace,
				Source: clusterID,
			})
		}

		if event.TestSuiteExecution.RunningContext != nil {
			ev.SetSubjectTrigger(&cdevents.TestSuiteRunQueuedSubjectContentTrigger{
				Type: MapTestkubeRunningContextTypeToCDEventTiggerType(event.TestSuiteExecution.RunningContext.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventStartTestSuiteToCDEvent maps OpenAPI spec Start Test Suite Event to CDEvent CDEventReader
func MapTestkubeEventStartTestSuiteToCDEvent(event testkube.Event, clusterID, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunStartedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestSuiteExecution != nil {
		ev.SetSubjectId(event.TestSuiteExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestSuiteExecution != nil {
		if event.TestSuiteExecution.TestSuite != nil {
			ev.SetSubjectTestSuite(&cdevents.TestSuiteRunStartedSubjectContentTestSuite{
				Id:  event.TestSuiteExecution.TestSuite.Name,
				Uri: fmt.Sprintf("%s/test-suites/%s", dashboardURI, event.TestSuiteExecution.TestSuite.Name),
			})

			ev.SetSubjectEnvironment(&cdevents.Reference{
				Id:     event.TestSuiteExecution.TestSuite.Namespace,
				Source: clusterID,
			})
		}

		if event.TestSuiteExecution.RunningContext != nil {
			ev.SetSubjectTrigger(&cdevents.TestSuiteRunStartedSubjectContentTrigger{
				Type: MapTestkubeRunningContextTypeToCDEventTiggerType(event.TestSuiteExecution.RunningContext.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventFinishTestSuiteToCDEvent maps OpenAPI spec Failed, Aborted, Timeout, Success Test Event to CDEvent CDEventReader
func MapTestkubeEventFinishTestSuiteToCDEvent(event testkube.Event, clusterID, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunFinishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestSuiteExecution != nil {
		ev.SetSubjectId(event.TestSuiteExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestSuiteExecution != nil {
		if event.TestSuiteExecution.TestSuite != nil {
			ev.SetSubjectTestSuite(&cdevents.TestSuiteRunFinishedSubjectContentTestSuite{
				Id:  event.TestSuiteExecution.TestSuite.Name,
				Uri: fmt.Sprintf("%s/test-suites/%s", dashboardURI, event.TestSuiteExecution.TestSuite.Name),
			})

			ev.SetSubjectEnvironment(&cdevents.Reference{
				Id:     event.TestSuiteExecution.TestSuite.Namespace,
				Source: clusterID,
			})
		}

		if event.TestSuiteExecution.IsAborted() || event.TestSuiteExecution.IsTimeout() {
			ev.SetSubjectOutcome("cancel")
			var errs []string
			for _, result := range event.TestSuiteExecution.StepResults {
				if result.Execution != nil && result.Execution.ExecutionResult != nil {
					errs = append(errs, result.Execution.ExecutionResult.ErrorMessage)
				}
			}

			for _, batch := range event.TestSuiteExecution.ExecuteStepResults {
				for _, result := range batch.Execute {
					if result.Execution != nil && result.Execution.ExecutionResult != nil {
						errs = append(errs, result.Execution.ExecutionResult.ErrorMessage)
					}
				}
			}

			ev.SetSubjectReason(strings.Join(errs, ","))
		}

		if event.TestSuiteExecution.IsFailed() {
			ev.SetSubjectOutcome("fail")
			var errs []string
			for _, result := range event.TestSuiteExecution.StepResults {
				if result.Execution != nil && result.Execution.ExecutionResult != nil {
					errs = append(errs, result.Execution.ExecutionResult.ErrorMessage)
				}
			}

			for _, batch := range event.TestSuiteExecution.ExecuteStepResults {
				for _, result := range batch.Execute {
					if result.Execution != nil && result.Execution.ExecutionResult != nil {
						errs = append(errs, result.Execution.ExecutionResult.ErrorMessage)
					}
				}
			}

			ev.SetSubjectReason(strings.Join(errs, ","))
		}

		if event.TestSuiteExecution.IsPassed() {
			ev.SetSubjectOutcome("pass")
		}
	}

	return ev, nil
}

// MapTestkubeRunningContextTypeToCDEventTiggerType maps OpenAPI spec Running Context Type to CDEvent Trigger Type
func MapTestkubeRunningContextTypeToCDEventTiggerType(contextType string) string {
	switch testkube.RunningContextType(contextType) {
	case testkube.RunningContextTypeUserCLI, testkube.RunningContextTypeUserUI:
		return "manual"
	case testkube.RunningContextTypeTestTrigger, testkube.RunningContextTypeTestSuite:
		return "event"
	case testkube.RunningContextTypeScheduler:
		return "schedule"
	}

	return "other"
}

// MapTestkubeTestTypeToCDEventTestCaseType maps OpenAPI spec Test Type to CDEvent Test Case Type
func MapTestkubeTestTypeToCDEventTestCaseType(testType string) string {
	var types = map[string]string{
		"artillery/":  "performance",
		"curl/":       "functional",
		"cypress/":    "functional",
		"ginkgo/":     "unit",
		"gradle/":     "integration",
		"jmeter/":     "performance",
		"k6/":         "performance",
		"kubepug/":    "compliance",
		"maven/":      "integration",
		"playwright/": "functional",
		"postman/":    "functional",
		"soapui/":     "functional",
		"zap/":        "security",
	}

	for key, value := range types {
		if strings.Contains(testType, key) {
			return value
		}
	}

	return "other"
}

// MapMimeTypeToCDEventOutputType maps mime type to CDEvent Output Type
func MapMimeTypeToCDEventOutputType(mimeType string) string {
	if strings.Contains(mimeType, "video/") || strings.Contains(mimeType, "audio/") {
		return "video"
	}

	if strings.Contains(mimeType, "image/") {
		return "image"
	}

	if strings.Contains(mimeType, "text/") {
		return "report"
	}

	var types = map[string]string{
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "report",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "report",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "report",
		"application/vnd.oasis.opendocument.text":                                   "report",
		"application/vnd.oasis.opendocument.spreadsheet":                            "report",
		"application/vnd.oasis.opendocument.presentation":                           "report",
		"application/pdf":               "report",
		"application/vnd.ms-excel":      "report",
		"application/vnd.ms-powerpoint": "report",
		"application/msword":            "report",
		"application/json":              "log",
	}

	for key, value := range types {
		if mimeType == key {
			return value
		}
	}

	return "other"
}
