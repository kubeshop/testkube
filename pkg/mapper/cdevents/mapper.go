package cdevents

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	cdevents "github.com/cdevents/sdk-go/pkg/api"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/mapper"
)

// MapTestkubeEventToCDEvent maps OpenAPI spec Event to CDEvent CDEventReader
func MapTestkubeEventToCDEvent(tkEvent testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	if tkEvent.Type_ == nil {
		return nil, errors.New("empty event type")
	}

	switch *tkEvent.Type_ {
	case *testkube.EventQueueTestWorkflow:
		if tkEvent.TestWorkflowExecution.ContainsExecuteAction() {
			return MapTestkubeEventQueuedTestWorkflowTestSuiteToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
		}

		return MapTestkubeEventQueuedTestWorkflowTestToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
	case *testkube.EventStartTestWorkflow:
		if tkEvent.TestWorkflowExecution.ContainsExecuteAction() {
			return MapTestkubeEventStartTestWorkflowTestSuiteToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
		}

		return MapTestkubeEventStartTestWorkflowTestToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
	case *testkube.EventEndTestWorkflowAborted, *testkube.EventEndTestWorkflowFailed, *testkube.EventEndTestWorkflowSuccess:
		if tkEvent.TestWorkflowExecution.ContainsExecuteAction() {
			return MapTestkubeEventFinishTestWorkflowTestSuiteToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
		}

		return MapTestkubeEventFinishTestWorkflowTestToCDEvent(tkEvent, clusterID, defaultNamespace, dashboardURI)
	}

	return nil, fmt.Errorf("not supported event type %s", tkEvent.Type_)
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

// MapTestkubeEventQueuedTestWorkflowTestToCDEvent maps OpenAPI spec Queued Test Workflow Test Event to CDEvent CDEventReader
func MapTestkubeEventQueuedTestWorkflowTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunQueuedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestCase(&cdevents.TestCaseRunQueuedSubjectContentTestCase{
			Id:   workflowName,
			Type: MapTestkubeTestWorkflowTemplateToCDEventTestCaseType(event.TestWorkflowExecution.GetTemplateRefs()),
			Uri:  fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.RunningContext != nil && event.TestWorkflowExecution.RunningContext.Actor != nil &&
			event.TestWorkflowExecution.RunningContext.Actor.Type_ != nil {
			ev.SetSubjectTrigger(&cdevents.TestCaseRunQueuedSubjectContentTrigger{
				// Pro edition only (tcl protected code)
				Type: mapper.MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(*event.TestWorkflowExecution.RunningContext.Actor.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventQueuedTestWorkflowTestSuiteToCDEvent maps OpenAPI spec Queued Test Workflow Test Suite Event to CDEvent CDEventReader
func MapTestkubeEventQueuedTestWorkflowTestSuiteToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunQueuedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestSuite(&cdevents.TestSuiteRunQueuedSubjectContentTestSuite{
			Id:  workflowName,
			Url: fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.RunningContext != nil && event.TestWorkflowExecution.RunningContext.Actor != nil &&
			event.TestWorkflowExecution.RunningContext.Actor.Type_ != nil {
			ev.SetSubjectTrigger(&cdevents.TestSuiteRunQueuedSubjectContentTrigger{
				// Pro edition only (tcl protected code)
				Type: mapper.MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(*event.TestWorkflowExecution.RunningContext.Actor.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventStartTestWorkflowTestToCDEvent maps OpenAPI spec Start Test Workflow Test Event to CDEvent CDEventReader
func MapTestkubeEventStartTestWorkflowTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunStartedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestCase(&cdevents.TestCaseRunStartedSubjectContentTestCase{
			Id:   workflowName,
			Type: MapTestkubeTestWorkflowTemplateToCDEventTestCaseType(event.TestWorkflowExecution.GetTemplateRefs()),
			Uri:  fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.RunningContext != nil && event.TestWorkflowExecution.RunningContext.Actor != nil &&
			event.TestWorkflowExecution.RunningContext.Actor.Type_ != nil {
			ev.SetSubjectTrigger(&cdevents.TestCaseRunStartedSubjectContentTrigger{
				// Pro edition only (tcl protected code)
				Type: mapper.MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(*event.TestWorkflowExecution.RunningContext.Actor.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventStartTestWorkflowTestSuiteToCDEvent maps OpenAPI spec Start Test Workflow Test Suite Event to CDEvent CDEventReader
func MapTestkubeEventStartTestWorkflowTestSuiteToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunStartedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestSuite(&cdevents.TestSuiteRunStartedSubjectContentTestSuite{
			Id:  workflowName,
			Uri: fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.RunningContext != nil && event.TestWorkflowExecution.RunningContext.Actor != nil &&
			event.TestWorkflowExecution.RunningContext.Actor.Type_ != nil {
			ev.SetSubjectTrigger(&cdevents.TestSuiteRunStartedSubjectContentTrigger{
				// Pro edition only (tcl protected code)
				Type: mapper.MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(*event.TestWorkflowExecution.RunningContext.Actor.Type_),
			})
		}
	}

	return ev, nil
}

// MapTestkubeEventFinishTestWorkflowTestToCDEvent maps OpenAPI spec Failed, Aborted, Timeout, Success Test Workflow Test Event to CDEvent CDEventReader
func MapTestkubeEventFinishTestWorkflowTestToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestCaseRunFinishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestCase(&cdevents.TestCaseRunFinishedSubjectContentTestCase{
			Id:   workflowName,
			Type: MapTestkubeTestWorkflowTemplateToCDEventTestCaseType(event.TestWorkflowExecution.GetTemplateRefs()),
			Uri:  fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.Result != nil {
			var errs []string
			if event.TestWorkflowExecution.Result.Initialization != nil &&
				event.TestWorkflowExecution.Result.Initialization.ErrorMessage != "" {
				errs = append(errs, event.TestWorkflowExecution.Result.Initialization.ErrorMessage)
			}

			for _, step := range event.TestWorkflowExecution.Result.Steps {
				if step.ErrorMessage != "" {
					errs = append(errs, step.ErrorMessage)
				}
			}

			if event.TestWorkflowExecution.Result.IsAborted() {
				ev.SetSubjectOutcome("cancel")
				ev.SetSubjectReason(strings.Join(errs, ","))
			}

			if event.TestWorkflowExecution.Result.IsFailed() {
				ev.SetSubjectOutcome("fail")
				ev.SetSubjectReason(strings.Join(errs, ","))
			}

			if event.TestWorkflowExecution.Result.IsPassed() {
				ev.SetSubjectOutcome("pass")
			}
		}
	}

	return ev, nil
}

// MapTestkubeEventFinishTestWorkflowTestSuiteToCDEvent maps OpenAPI spec Failed, Aborted, Timeout, Success Test Workflow Test Event to CDEvent CDEventReader
func MapTestkubeEventFinishTestWorkflowTestSuiteToCDEvent(event testkube.Event, clusterID, defaultNamespace, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestSuiteRunFinishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id)
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectTestSuite(&cdevents.TestSuiteRunFinishedSubjectContentTestSuite{
			Id:  workflowName,
			Uri: fmt.Sprintf("%s/test-workflows/%s", dashboardURI, workflowName),
		})

		namespace := event.TestWorkflowExecution.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		ev.SetSubjectEnvironment(&cdevents.Reference{
			Id:     namespace,
			Source: clusterID,
		})

		if event.TestWorkflowExecution.Result != nil {
			var errs []string
			if event.TestWorkflowExecution.Result.Initialization != nil &&
				event.TestWorkflowExecution.Result.Initialization.ErrorMessage != "" {
				errs = append(errs, event.TestWorkflowExecution.Result.Initialization.ErrorMessage)
			}

			for _, step := range event.TestWorkflowExecution.Result.Steps {
				if step.ErrorMessage != "" {
					errs = append(errs, step.ErrorMessage)
				}
			}

			if event.TestWorkflowExecution.Result.IsAborted() {
				ev.SetSubjectOutcome("cancel")
				ev.SetSubjectReason(strings.Join(errs, ","))
			}

			if event.TestWorkflowExecution.Result.IsFailed() {
				ev.SetSubjectOutcome("fail")
				ev.SetSubjectReason(strings.Join(errs, ","))
			}

			if event.TestWorkflowExecution.Result.IsPassed() {
				ev.SetSubjectOutcome("pass")
			}
		}
	}

	return ev, nil
}

// MapTestkubeTestWorkflowLogToCDEvent maps OpenAPI spec Test WWorkflow log to CDEvent CDEventReader
func MapTestkubeTestWorkflowLogToCDEvent(event testkube.Event, clusterID, dashboardURI string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestOutputPublishedEvent()
	if err != nil {
		return nil, err
	}

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectId(event.TestWorkflowExecution.Id + "-log")
	}

	ev.SetSubjectSource(clusterID)
	ev.SetSource(clusterID)

	if event.TestWorkflowExecution != nil {
		ev.SetSubjectTestCaseRun(&cdevents.Reference{
			Id:     event.TestWorkflowExecution.Id,
			Source: clusterID,
		})
	}

	ev.SetSubjectFormat("text/x-uri")
	ev.SetSubjectOutputType("log")
	if event.TestWorkflowExecution != nil {
		workflowName := ""
		if event.TestWorkflowExecution.Workflow != nil {
			workflowName = event.TestWorkflowExecution.Workflow.Name
		}

		ev.SetSubjectUri(fmt.Sprintf("%s/test-workflows/%s/executions/%s", dashboardURI,
			workflowName, event.TestWorkflowExecution.Id))
	}

	return ev, nil
}

// MapTestkubeTestWorkflowTemplateToCDEventTestCaseType maps OpenAPI spec Test Workflow Template to CDEvent Test Case Type
func MapTestkubeTestWorkflowTemplateToCDEventTestCaseType(templateRefs []testkube.TestWorkflowTemplateRef) string {
	var types = map[string]string{
		"official--artillery":  "performance",
		"official--cypress":    "functional",
		"official--gradle":     "integration",
		"official--jmeter":     "performance",
		"official--k6":         "performance",
		"official--maven":      "integration",
		"official--playwright": "functional",
		"official--postman":    "functional",
	}

	templateNames := make(map[string]struct{})
	for _, templateRef := range templateRefs {
		if strings.Contains(templateRef.Name, "official--") {
			templateNames[templateRef.Name] = struct{}{}
		}
	}

	for key, value := range types {
		for templateName := range templateNames {
			if strings.Contains(templateName, key) {
				return value
			}
		}
	}

	return "other"
}

// CDEventsArtifactParameters contains cd events artifact parameters
type CDEventsArtifactParameters struct {
	Id           string
	Name         string
	WorkflowName string
	ClusterID    string
	DashboardURI string
}

// MapTestkubeGTestWorkflowArtifactToCDEvent maps OpenAPI spec Test Artifact to CDEvent CDEventReader
func MapTestkubeTestWorkflowArtifactToCDEvent(parameters CDEventsArtifactParameters, path, format string) (cdevents.CDEventReader, error) {
	// Create the base event
	ev, err := cdevents.NewTestOutputPublishedEvent()
	if err != nil {
		return nil, err
	}

	ev.SetSubjectId(filepath.Join(parameters.Name, path))
	ev.SetSubjectSource(parameters.ClusterID)
	ev.SetSource(parameters.ClusterID)
	ev.SetSubjectTestCaseRun(&cdevents.Reference{
		Id:     parameters.Id,
		Source: parameters.ClusterID,
	})

	ev.SetSubjectFormat(format)
	ev.SetSubjectOutputType(MapMimeTypeToCDEventOutputType(format))
	ev.SetSubjectUri(fmt.Sprintf("%s/test-workflows/%s/overview/%s/artifacts", parameters.DashboardURI, parameters.WorkflowName, parameters.Id))
	return ev, nil
}
