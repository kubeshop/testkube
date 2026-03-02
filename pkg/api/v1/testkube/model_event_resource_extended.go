package testkube

func EventResourcePtr(t EventResource) *EventResource {
	return &t
}

var (
	EventResourceTest                  = EventResourcePtr(TEST_EventResource)
	EventResourceTestsuite             = EventResourcePtr(TESTSUITE_EventResource)
	EventResourceExecutor              = EventResourcePtr(EXECUTOR_EventResource)
	EventResourceTrigger               = EventResourcePtr(TRIGGER_EventResource)
	EventResourceWebhook               = EventResourcePtr(WEBHOOK_EventResource)
	EventResourceWebhookTemplate       = EventResourcePtr(WEBHOOKTEMPLATE_EventResource)
	EventResourceTestexecution         = EventResourcePtr(TESTEXECUTION_EventResource)
	EventResourceTestsuiteexecution    = EventResourcePtr(TESTSUITEEXECUTION_EventResource)
	EventResourceTestsource            = EventResourcePtr(TESTSOURCE_EventResource)
	EventResourceTestWorkflow          = EventResourcePtr(TESTWORKFLOW_EventResource)
	EventResourceTestWorkflowExecution = EventResourcePtr(TESTWORKFLOWEXECUTION_EventResource)
)
