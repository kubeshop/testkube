package testkube

import (
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
)

func NewEventStartTest(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventStartTest,
		TestExecution: execution,
	}
}

func NewEventEndTestSuccess(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestSuccess,
		TestExecution: execution,
	}
}

func NewEventEndTestFailed(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestFailed,
		TestExecution: execution,
	}
}

func NewEventEndTestAborted(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestAborted,
		TestExecution: execution,
	}
}

func NewEventEndTestTimeout(execution *Execution) Event {
	return Event{
		Id:            uuid.NewString(),
		Type_:         EventEndTestTimeout,
		TestExecution: execution,
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

	if e.TestSuiteExecution != nil {
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

	return []any{
		"id", e.Id,
		"type", eventType,
		"executionId", id,
		"executionName", name,
		"labels", labelsStr,
	}
}

func (e Event) Valid(selector string, types []EventType) (valid bool) {
	var executionLabels map[string]string

	// load labels from event test execution or test-suite execution
	if e.TestSuiteExecution != nil {
		executionLabels = e.TestSuiteExecution.Labels
	} else if e.TestExecution != nil {
		executionLabels = e.TestExecution.Labels
	} else {
		return false
	}

	typesMatch := false
	for _, t := range types {
		if t == e.Type() {
			typesMatch = true
			break
		}
	}

	if !typesMatch {
		return false
	}

	valid = selector == ""
	if !valid {
		selector, err := labels.Parse(selector)
		if err != nil {
			return false
		}

		valid = selector.Matches(labels.Set(executionLabels))
	}

	return
}
