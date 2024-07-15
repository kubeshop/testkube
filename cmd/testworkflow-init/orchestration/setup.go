package orchestration

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
)

var (
	scopedRegex       = regexp.MustCompile(`^_(00|01|\d|[1-9]\d*)(C)?_`)
	Setup             = newSetup()
	defaultWorkingDir = getWorkingDir()
)

func getWorkingDir() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "/"
	}
	return wd
}

type setup struct {
	envBase           map[string]string
	envGroups         map[string]map[string]string
	envGroupsComputed map[string]map[string]struct{}
	envCurrentGroup   int
}

func newSetup() *setup {
	c := &setup{
		envBase:           map[string]string{},
		envGroups:         map[string]map[string]string{},
		envGroupsComputed: map[string]map[string]struct{}{},
		envCurrentGroup:   -1,
	}
	c.initialize()
	return c
}

func (c *setup) initialize() {
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
			c.envGroupsComputed[match[1]] = map[string]struct{}{}
		}
		c.envGroups[match[1]][key[len(match[0]):]] = value
		if match[2] == "C" {
			c.envGroupsComputed[match[1]][key[len(match[0]):]] = struct{}{}
		}
		os.Unsetenv(key)
	}
}

func (c *setup) UseBaseEnv() {
	os.Clearenv()
	for k, v := range c.envBase {
		_ = os.Setenv(k, v)
	}
}

func (c *setup) UseEnv(group string) {
	c.UseBaseEnv()

	envTemplates := map[string]string{}
	envResolutions := map[string]expressions.Expression{}
	for k, v := range c.envGroups[group] {
		if _, ok := c.envGroupsComputed[group][k]; ok {
			envTemplates[k] = v
		} else {
			_ = os.Setenv(k, v)
		}
	}

	// Configure PWD variable, to make it similar to shell environment variables
	cwd := getWorkingDir()
	if os.Getenv("PWD") == "" {
		_ = os.Setenv("PWD", cwd)
	}

	// Ensure the built-in binaries are available
	if os.Getenv("PATH") == "" {
		_ = os.Setenv("PATH", data.InternalBinPath)
	} else {
		_ = os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), data.InternalBinPath))
	}

	// Compute dynamic environment variables
	addonMachine := expressions.CombinedMachines(data.RefSuccessMachine, data.AliasMachine, data.StateMachine, libs.NewFsMachine(os.DirFS("/"), cwd))
	localEnvMachine := expressions.NewMachine().
		RegisterAccessorExt(func(accessorName string) (interface{}, bool, error) {
			if !strings.HasPrefix(accessorName, "env.") {
				return nil, false, nil
			}
			name := accessorName[4:]
			if v, ok := envResolutions[name]; ok {
				return v, true, nil
			} else if _, ok := envTemplates[name]; ok {
				result, err := expressions.CompileAndResolveTemplate(envTemplates[name], addonMachine)
				if err != nil {
					envResolutions[name] = result
				}
				return result, true, err
			}
			return os.Getenv(name), true, nil
		})
	for name, expr := range envTemplates {
		value, err := expressions.CompileAndResolveTemplate(expr, localEnvMachine, addonMachine, expressions.FinalizerFail)
		if err != nil {
			panic(fmt.Sprintf("failed to compute '%s' environment variable: %s", name, err.Error()))
		}
		str, _ := value.Static().StringValue()
		_ = os.Setenv(name, str)
	}
}

func (c *setup) UseCurrentEnv() {
	c.UseEnv(fmt.Sprintf("%d", c.envCurrentGroup))
}

func (c *setup) AdvanceEnv() {
	c.envCurrentGroup++
	c.UseCurrentEnv()
}

func (c *setup) SetConfig(config testworkflowsv1.ContainerConfig) {
	if config.WorkingDir == nil || *config.WorkingDir == "" {
		_ = os.Chdir(*config.WorkingDir)
	} else {
		_ = os.Chdir(defaultWorkingDir)
	}
}
