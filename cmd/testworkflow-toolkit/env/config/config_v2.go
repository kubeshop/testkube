package config

import (
	"encoding/json"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// ConfigV2 holds configuration loaded from TK_CFG
type ConfigV2 struct {
	internal testworkflowconfig.InternalConfig
	env      envConfig
}

// LoadConfigV2 loads configuration from TK_CFG environment variable
func LoadConfigV2() (*ConfigV2, error) {
	var cfg ConfigV2

	// Load internal config from TK_CFG
	tkCfgStr := os.Getenv("TK_CFG")
	if tkCfgStr == "" {
		return nil, errors.New("TK_CFG environment variable is not set")
	}

	if err := json.Unmarshal([]byte(tkCfgStr), &cfg.internal); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal TK_CFG")
	}

	// Load environment config
	if err := envconfig.Process("", &cfg.env); err != nil {
		return nil, errors.Wrap(err, "failed to load environment config")
	}

	return &cfg, nil
}

// Internal returns the internal configuration
func (c *ConfigV2) Internal() *testworkflowconfig.InternalConfig {
	return &c.internal
}

// Env returns the environment configuration
func (c *ConfigV2) Env() *envConfig {
	return &c.env
}

// Debug returns true if debug mode is enabled
func (c *ConfigV2) Debug() bool {
	return c.internal.Execution.Debug || os.Getenv("DEBUG") == "1"
}

// Ref returns the execution reference
func (c *ConfigV2) Ref() string {
	return c.env.Ref
}

// Namespace returns the worker namespace
func (c *ConfigV2) Namespace() string {
	return c.internal.Worker.Namespace
}

// IP returns the IP address
func (c *ConfigV2) IP() string {
	return c.env.Ip
}

// WorkflowName returns the workflow name
func (c *ConfigV2) WorkflowName() string {
	return c.internal.Workflow.Name
}

// ExecutionId returns the execution ID
func (c *ConfigV2) ExecutionId() string {
	return c.internal.Execution.Id
}

// ExecutionName returns the execution name
func (c *ConfigV2) ExecutionName() string {
	return c.internal.Execution.Name
}

// ExecutionNumber returns the execution number
func (c *ConfigV2) ExecutionNumber() int64 {
	return int64(c.internal.Execution.Number)
}

// JUnitParserEnabled returns true if JUnit parser is enabled
func (c *ConfigV2) JUnitParserEnabled() bool {
	return c.env.EnableJUnitParser
}

// ExecutionTags returns the execution tags
func (c *ConfigV2) ExecutionTags() map[string]string {
	return c.internal.Execution.Tags
}
