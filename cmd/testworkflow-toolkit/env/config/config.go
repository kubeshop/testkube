package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	UseProxyValue = false
)

type envConfig struct {
	Ip  string `envconfig:"TK_IP"`
	Ref string `envconfig:"TK_REF"`

	EnableJUnitParser bool `envconfig:"TK_FF_JUNIT_REPORT" default:"false"`
}

var cfg testworkflowconfig.InternalConfig
var envCfg envConfig
var cfgLoaded = false

func loadConfig() {
	if !cfgLoaded {
		err := json.Unmarshal([]byte(os.Getenv("TK_CFG")), &cfg)
		ui.ExitOnError("loading internal configuration", err)
		err = envconfig.Process("", &envCfg)
		ui.ExitOnError("loading feature flags", err)
		cfgLoaded = true
	}
}

func EnvConfig() *envConfig {
	loadConfig()
	return &envCfg
}

func Config() *testworkflowconfig.InternalConfig {
	loadConfig()
	return &cfg
}

func Debug() bool {
	return Config().Execution.Debug || os.Getenv("DEBUG") == "1"
}

func UseProxy() bool {
	return UseProxyValue
}

func Ref() string {
	return EnvConfig().Ref
}

func Namespace() string {
	return Config().Worker.Namespace
}

func IP() string {
	return EnvConfig().Ip
}

func WorkflowName() string {
	return Config().Workflow.Name
}

func ExecutionId() string {
	return Config().Execution.Id
}

func ExecutionName() string {
	return Config().Execution.Name
}

func ExecutionNumber() int64 {
	return int64(Config().Execution.Number)
}

func ExecutionScheduledAt() time.Time {
	return Config().Execution.ScheduledAt
}

func ExecutionDisableWebhooks() bool {
	return Config().Execution.DisableWebhooks
}

func JUnitParserEnabled() bool {
	return EnvConfig().EnableJUnitParser
}

func ExecutionTags() map[string]string {
	return Config().Execution.Tags
}
