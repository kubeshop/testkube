package config

type ProContext struct {
	APIKey                                       string
	URL                                          string
	TLSInsecure                                  bool
	WorkerCount                                  int
	LogStreamWorkerCount                         int
	WorkflowNotificationsWorkerCount             int
	WorkflowServiceNotificationsWorkerCount      int
	WorkflowParallelStepNotificationsWorkerCount int
	SkipVerify                                   bool
	EnvID                                        string
	OrgID                                        string
	Migrate                                      string
	ConnectionTimeout                            int
	DashboardURI                                 string
	NewExecutions                                bool
	TestWorkflowStorage                          bool
}
