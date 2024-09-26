package config

type ProContext struct {
	APIKey                           string
	URL                              string
	TLSInsecure                      bool
	WorkerCount                      int
	LogStreamWorkerCount             int
	WorkflowNotificationsWorkerCount int
	SkipVerify                       bool
	EnvID                            string
	EnvName                          string
	EnvSlug                          string
	OrgID                            string
	OrgName                          string
	OrgSlug                          string
	Migrate                          string
	ConnectionTimeout                int
	DashboardURI                     string
	ClusterId                        string
	RunnerId                         string
}
