package testkube

import (
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	TestStartSubject = "agentevents.test.start"
	TestStopSubject  = "agentevents.test.stop"
)

// check if Event implements model generic event type
var _ Trigger = Event{}

// Trigger for generic events
type Trigger interface {
	GetResourceId() string
}

func NewEvent(t *EventType, resource *EventResource, id string) Event {
	return Event{
		Id:         uuid.NewString(),
		ResourceId: id,
		Resource:   resource,
		Type_:      t,
	}
}

func NewEventStartTest(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventStartTest,
		TestExecution: execution,
		StreamTopic:   TestStartSubject,
		ResourceId:    execution.Id,
	}
}

func NewEventEndTestSuccess(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestSuccess,
		TestExecution: execution,
		StreamTopic:   TestStopSubject,
		ResourceId:    execution.Id,
	}
}

func NewEventEndTestFailed(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestFailed,
		TestExecution: execution,
		StreamTopic:   TestStopSubject,
		ResourceId:    execution.Id,
	}
}

func NewEventEndTestAborted(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestAborted,
		TestExecution: execution,
		StreamTopic:   TestStopSubject,
		ResourceId:    execution.Id,
	}
}

func NewEventEndTestTimeout(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestTimeout,
		TestExecution: execution,
		StreamTopic:   TestStopSubject,
		ResourceId:    execution.Id,
	}
}

func NewEventStartTestSuite(execution *TestSuiteExecution) Event {
	return Event{
		Id:                 uuid.NewString(),
		Type_:              EventStartTestSuite,
		TestSuiteExecution: execution,
	}
}

func NewEventEndTestSuiteSuccess(execution *TestSuiteExecution) Event {
	return Event{
		Id:                 uuid.NewString(),
		Type_:              EventEndTestSuiteSuccess,
		TestSuiteExecution: execution,
	}
}

func NewEventEndTestSuiteFailed(execution *TestSuiteExecution) Event {
	return Event{
		Id:                 uuid.NewString(),
		Type_:              EventEndTestSuiteFailed,
		TestSuiteExecution: execution,
	}
}

func NewEventEndTestSuiteAborted(execution *TestSuiteExecution) Event {
	return Event{
		Id:                 uuid.NewString(),
		Type_:              EventEndTestSuiteAborted,
		TestSuiteExecution: execution,
	}
}

func NewEventEndTestSuiteTimeout(execution *TestSuiteExecution) Event {
	return Event{
		Id:                 uuid.NewString(),
		Type_:              EventEndTestSuiteTimeout,
		TestSuiteExecution: execution,
	}
}

func NewEventQueueTestWorkflow(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventQueueTestWorkflow,
		TestWorkflowExecution: execution,
	}
}

func NewEventStartTestWorkflow(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventStartTestWorkflow,
		TestWorkflowExecution: execution,
	}
}

func NewEventEndTestWorkflowSuccess(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventEndTestWorkflowSuccess,
		TestWorkflowExecution: execution,
	}
}

func NewEventEndTestWorkflowFailed(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventEndTestWorkflowFailed,
		TestWorkflowExecution: execution,
	}
}

func NewEventEndTestWorkflowAborted(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventEndTestWorkflowAborted,
		TestWorkflowExecution: execution,
	}
}

func NewEventEndTestWorkflowCanceled(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventEndTestWorkflowCanceled,
		TestWorkflowExecution: execution,
	}
}

func NewEventEndTestWorkflowNotPassed(execution *TestWorkflowExecution) Event {
	return Event{
		Id:                    uuid.NewString(),
		Type_:                 EventEndTestWorkflowNotPassed,
		TestWorkflowExecution: execution,
	}
}

func (e Event) Type() EventType {
	if e.Type_ != nil {
		return *e.Type_
	}
	return EventType("")
}

func (e Event) IsSuccess() bool {
	return strings.Contains(e.Type().String(), "success")
}

func (e Event) Log() []any {
	var id, name, eventType, labelsStr string
	var labels map[string]string

	if e.TestWorkflowExecution != nil {
		id = e.TestWorkflowExecution.Id
		name = e.TestWorkflowExecution.Name
		if e.TestWorkflowExecution.Workflow != nil {
			labels = e.TestWorkflowExecution.Workflow.Labels
		}
	} else if e.TestSuiteExecution != nil {
		id = e.TestSuiteExecution.Id
		name = e.TestSuiteExecution.Name
		labels = e.TestSuiteExecution.Labels
	} else if e.TestExecution != nil {
		id = e.TestExecution.Id
		name = e.TestExecution.Name
		labels = e.TestExecution.Labels
	}

	if e.Type_ != nil {
		eventType = e.Type_.String()
	}

	for k, v := range labels {
		labelsStr += k + "=" + v + " "
	}

	resource := ""
	if e.Resource != nil {
		resource = string(*e.Resource)
	}

	return []any{
		"id", e.Id,
		"type", eventType,
		"resource", resource,
		"resourceId", e.ResourceId,
		"executionName", name,
		"executionId", id,
		"labels", labelsStr,
		"topic", e.Topic(),
	}
}

func (e Event) Valid(selector string, types []EventType) (matchedTypes []EventType, valid bool) {
	var executionLabels map[string]string

	// load labels from event test execution or test-suite execution or test workflow execution
	if e.TestWorkflowExecution != nil {
		if e.TestWorkflowExecution.Workflow != nil {
			executionLabels = e.TestWorkflowExecution.Workflow.Labels
		}
	} else if e.TestSuiteExecution != nil {
		executionLabels = e.TestSuiteExecution.Labels
	} else if e.TestExecution != nil {
		executionLabels = e.TestExecution.Labels
	}

	typesMatch := false
	for _, t := range types {
		ts := []EventType{t}
		if t.IsBecome() {
			ts = t.MapBecomeToRegular()
		}

		for _, et := range ts {
			if et == e.Type() {
				typesMatch = true
				matchedTypes = append(matchedTypes, t)
				break
			}
		}
	}

	if !typesMatch {
		return nil, false
	}

	valid = selector == ""
	if !valid {
		selector, err := labels.Parse(selector)
		if err != nil {
			return nil, false
		}

		valid = selector.Matches(labels.Set(executionLabels))
	}

	return
}

// Topic returns topic for event based on resource and resource id
// or fallback to global "events" topic
func (e Event) Topic() string {
	if e.StreamTopic != "" {
		return e.StreamTopic
	}

	if e.Resource == nil {
		return "agentevents.all"
	}

	if e.ResourceId == "" {
		return "agentevents." + string(*e.Resource)
	}

	return "agentevents." + string(*e.Resource) + "." + e.ResourceId
}

// GetResourceId implmenents generic event trigger
func (e Event) GetResourceId() string {
	return e.ResourceId
}
