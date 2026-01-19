package testkube

import (
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/kubeshop/testkube/internal/common"
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

func NewEventEndTestWorkflowSuccess(execution *TestWorkflowExecution, groupId string) Event {
	return Event{
		Id:                    uuid.NewString(),
		GroupId:               groupId,
		Type_:                 EventEndTestWorkflowSuccess,
		TestWorkflowExecution: execution,
		Resource:              common.Ptr(TESTWORKFLOWEXECUTION_EventResource),
		ResourceId:            execution.Id,
	}
}

func NewEventEndTestWorkflowFailed(execution *TestWorkflowExecution, groupId string) Event {
	return Event{
		Id:                    uuid.NewString(),
		GroupId:               groupId,
		Type_:                 EventEndTestWorkflowFailed,
		TestWorkflowExecution: execution,
		Resource:              common.Ptr(TESTWORKFLOWEXECUTION_EventResource),
		ResourceId:            execution.Id,
	}
}

func NewEventEndTestWorkflowAborted(execution *TestWorkflowExecution, groupId string) Event {
	return Event{
		Id:                    uuid.NewString(),
		GroupId:               groupId,
		Type_:                 EventEndTestWorkflowAborted,
		TestWorkflowExecution: execution,
		Resource:              common.Ptr(TESTWORKFLOWEXECUTION_EventResource),
		ResourceId:            execution.Id,
	}
}

func NewEventEndTestWorkflowCanceled(execution *TestWorkflowExecution, groupId string) Event {
	return Event{
		Id:                    uuid.NewString(),
		GroupId:               groupId,
		Type_:                 EventEndTestWorkflowCanceled,
		TestWorkflowExecution: execution,
		Resource:              common.Ptr(TESTWORKFLOWEXECUTION_EventResource),
		ResourceId:            execution.Id,
	}
}

func NewEventEndTestWorkflowNotPassed(execution *TestWorkflowExecution, groupId string) Event {
	return Event{
		Id:                    uuid.NewString(),
		GroupId:               groupId,
		Type_:                 EventEndTestWorkflowNotPassed,
		TestWorkflowExecution: execution,
		Resource:              common.Ptr(TESTWORKFLOWEXECUTION_EventResource),
		ResourceId:            execution.Id,
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
	}
}

func (e Event) Valid(groupId, selector string, types []EventType) (matchedTypes []EventType, valid bool) {
	if groupId != "" && e.GroupId != groupId {
		return nil, false
	}

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

// GetResourceId implmenents generic event trigger
func (e Event) GetResourceId() string {
	return e.ResourceId
}
