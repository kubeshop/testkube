package container

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

var (
	scopedRegex          = regexp.MustCompile(`^_(00|01|\d|[1-9]\d*)_`)
	Container            = newContainer()
	defaultWorkingDir, _ = os.Getwd()
)

type container struct {
	envBase         map[string]string
	envGroups       map[string]map[string]string
	envCurrentGroup int
}

func newContainer() *container {
	c := &container{
		envBase:         map[string]string{},
		envGroups:       map[string]map[string]string{},
		envCurrentGroup: -1,
	}
	c.initialize()
	return c
}

func (c *container) initialize() {
	// Iterate over the environment variables to group them
	for _, item := range os.Environ() {
		match := scopedRegex.FindStringSubmatch(item)
		key, value, _ := strings.Cut(item, "=")
		if match == nil {
			c.envBase[key] = value
			continue
		}

		if c.envGroups[match[1]] == nil {
			c.envGroups[match[1]] = map[string]string{}
		}
		c.envGroups[match[1]][key[len(match[0]):]] = value
		os.Unsetenv(key)
	}
}

func (c *container) UseBaseEnv() {
	os.Clearenv()
	for k, v := range c.envBase {
		_ = os.Setenv(k, v)
	}
}

func (c *container) UseEnv(group string) {
	c.UseBaseEnv()
	for k, v := range c.envGroups[group] {
		_ = os.Setenv(k, v)
	}

	// Configure PWD variable, to make it similar to shell environment variables
	if os.Getenv("PWD") == "" {
		cwd, err := os.Getwd()
		if err == nil {
			_ = os.Setenv("PWD", cwd)
		}
	}

	// Ensure the built-in binaries are available
	if os.Getenv("PATH") == "" {
		_ = os.Setenv("PATH", data.InternalBinPath)
	} else {
		_ = os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), data.InternalBinPath))
	}

	// TODO: Resolve computed environment variables
}

func (c *container) UseCurrentEnv() {
	c.UseEnv(fmt.Sprintf("%d", c.envCurrentGroup))
}

func (c *container) AdvanceEnv() {
	c.envCurrentGroup++
	c.UseCurrentEnv()
}

func (c *container) SetConfig(config testworkflowsv1.ContainerConfig) {
	if config.WorkingDir == nil || *config.WorkingDir == "" {
		_ = os.Chdir(*config.WorkingDir)
	} else {
		_ = os.Chdir(defaultWorkingDir)
	}
}
