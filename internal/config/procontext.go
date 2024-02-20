package config

type ProContext struct {
	APIKey               string
	URL                  string
	LogsPath             string
	TLSInsecure          bool
	WorkerCount          int
	LogStreamWorkerCount int
	SkipVerify           bool
	EnvID                string
	OrgID                string
	Migrate              string
}
