package testkube

var AllEventTypes = []EventType{
	QUEUE_TESTWORKFLOW_EventType,
	START_TESTWORKFLOW_EventType,
	END_TESTWORKFLOW_SUCCESS_EventType,
	END_TESTWORKFLOW_FAILED_EventType,
	END_TESTWORKFLOW_ABORTED_EventType,
	END_TESTWORKFLOW_CANCELED_EventType,
	CREATED_EventType,
	DELETED_EventType,
	UPDATED_EventType,
}

func (t EventType) String() string {
	return string(t)
}

func EventTypePtr(t EventType) *EventType {
	return &t
}

var (
	EventQueueTestWorkflow        = EventTypePtr(QUEUE_TESTWORKFLOW_EventType)
	EventStartTestWorkflow        = EventTypePtr(START_TESTWORKFLOW_EventType)
	EventEndTestWorkflowSuccess   = EventTypePtr(END_TESTWORKFLOW_SUCCESS_EventType)
	EventEndTestWorkflowFailed    = EventTypePtr(END_TESTWORKFLOW_FAILED_EventType)
	EventEndTestWorkflowAborted   = EventTypePtr(END_TESTWORKFLOW_ABORTED_EventType)
	EventEndTestWorkflowCanceled  = EventTypePtr(END_TESTWORKFLOW_CANCELED_EventType)
	EventEndTestWorkflowNotPassed = EventTypePtr(END_TESTWORKFLOW_NOT_PASSED_EventType)
	EventCreated                  = EventTypePtr(CREATED_EventType)
	EventDeleted                  = EventTypePtr(DELETED_EventType)
	EventUpdated                  = EventTypePtr(UPDATED_EventType)
)

// causeEventStatusDetailsTypes maps each event for the cause of a failure to its status
// details type. Nothing emits these events. Each one matches the not-passed end event of an
// execution with that type. An execution that does not pass emits exactly one not-passed
// event, so each cause event gives at most one call per execution.
var causeEventStatusDetailsTypes = map[EventType]StatusDetailsType{
	END_TESTWORKFLOW_TEST_FAILURE_EventType:           StatusDetailsTypeStepFailure,
	END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType: StatusDetailsTypeExecutionFailure,
	END_TESTWORKFLOW_CONFIGURATION_ERROR_EventType:    StatusDetailsTypeInitFailure,
}

// causeStatusDetailsType returns the status details type of an event for the cause of a
// failure, and false for every other event type.
func (t EventType) causeStatusDetailsType() (StatusDetailsType, bool) {
	detailsType, ok := causeEventStatusDetailsTypes[t]
	return detailsType, ok
}

func (t EventType) IsBecome() bool {
	types := []EventType{
		BECOME_TESTWORKFLOW_UP_EventType,
		BECOME_TESTWORKFLOW_DOWN_EventType,
		BECOME_TESTWORKFLOW_FAILED_EventType,
		BECOME_TESTWORKFLOW_ABORTED_EventType,
		BECOME_TESTWORKFLOW_CANCELED_EventType,
	}

	for _, tp := range types {
		if tp == t {
			return true
		}
	}

	return false
}

func (t EventType) MapBecomeToRegular() []EventType {
	eventMap := map[EventType][]EventType{
		BECOME_TESTWORKFLOW_UP_EventType:       {END_TESTWORKFLOW_SUCCESS_EventType},
		BECOME_TESTWORKFLOW_DOWN_EventType:     {END_TESTWORKFLOW_FAILED_EventType, END_TESTWORKFLOW_ABORTED_EventType},
		BECOME_TESTWORKFLOW_FAILED_EventType:   {END_TESTWORKFLOW_FAILED_EventType},
		BECOME_TESTWORKFLOW_ABORTED_EventType:  {END_TESTWORKFLOW_ABORTED_EventType},
		BECOME_TESTWORKFLOW_CANCELED_EventType: {END_TESTWORKFLOW_CANCELED_EventType},
	}

	return eventMap[t]
}

func (t EventType) IsBecomeTestWorkflowExecutionStatus(previousStatus TestWorkflowStatus) bool {
	eventMap := map[EventType]map[TestWorkflowStatus]struct{}{
		BECOME_TESTWORKFLOW_UP_EventType: {
			FAILED_TestWorkflowStatus:  {},
			ABORTED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_DOWN_EventType: {
			PASSED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_FAILED_EventType: {
			PASSED_TestWorkflowStatus:  {},
			ABORTED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_ABORTED_EventType: {
			PASSED_TestWorkflowStatus: {},
			FAILED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_CANCELED_EventType: {
			PASSED_TestWorkflowStatus: {},
			FAILED_TestWorkflowStatus: {},
		},
	}

	if statusMap, ok := eventMap[t]; ok {
		if _, ok := statusMap[previousStatus]; ok {
			return true
		}
	}

	return false
}
