package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

var (
	scopedRegex              = regexp.MustCompile(`^_(00|01|\d|[1-9]\d*)(C)?(S?)_`)
	Setup                    = newSetup()
	defaultWorkingDir        = getWorkingDir()
	commonSensitiveVariables = []string{
		"TK_C_KEY",        // Cloud API key
		"TK_OS_ACCESSKEY", // Object Storage Access Key
		"TK_OS_SECRETKEY", // Object Storage Secret Key
		"TK_OS_TOKEN",     // Object Storage Token
		"TK_GIT_USERNAME", // Git Username
		"TK_GIT_TOKEN",    // Git Token
		"TK_SSH_KEY",      // Git SSH Key
	}
)

func getWorkingDir() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "/"
	}
	return wd
}

type setup struct {
	envBase                map[string]string
	envGroups              map[string]map[string]string
	envGroupsComputed      map[string]map[string]struct{}
	envGroupsSensitive     map[string]map[string]struct{}
	envCurrentGroup        int
	envSelectedGroup       string
	minSensitiveWordLength int
}

func newSetup() *setup {
	c := &setup{
		envBase:                map[string]string{},
		envGroups:              map[string]map[string]string{},
		envGroupsComputed:      map[string]map[string]struct{}{},
		envGroupsSensitive:     map[string]map[string]struct{}{},
		envCurrentGroup:        -1,
		minSensitiveWordLength: 1,
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
			c.envGroupsSensitive[match[1]] = map[string]struct{}{}
		}
		c.envGroups[match[1]][key[len(match[0]):]] = value
		if match[2] == "C" {
			c.envGroupsComputed[match[1]][key[len(match[0]):]] = struct{}{}
		}
		if match[3] == "S" {
			c.envGroupsSensitive[match[1]][key[len(match[0]):]] = struct{}{}
		}
		os.Unsetenv(key)
	}
}

func (c *setup) UseBaseEnv() {
	os.Clearenv()
	for k, v := range c.envBase {
		os.Setenv(k, v)
	}
}

func (c *setup) SetSensitiveWordMinimumLength(length int) {
	if length > 0 {
		c.minSensitiveWordLength = length
	} else {
		c.minSensitiveWordLength = 1
	}
}

func (c *setup) GetSensitiveWords() []string {
	words := make([]string, 0)
	for _, name := range commonSensitiveVariables {
		value := os.Getenv(name)
		if len(value) < c.minSensitiveWordLength {
			continue
		}
		words = append(words, value)
	}
	for k := range c.envBase {
		value := os.Getenv(k)
		if len(value) < c.minSensitiveWordLength {
			continue
		}
		if _, ok := c.envGroupsSensitive[c.envSelectedGroup][k]; ok {
			words = append(words, value)
		}
	}
	for k := range c.envGroups[c.envSelectedGroup] {
		value := os.Getenv(k)
		if len(value) < c.minSensitiveWordLength {
			continue
		}
		if _, ok := c.envGroupsSensitive[c.envSelectedGroup][k]; ok {
			words = append(words, value)
		}
	}
	return words
}

func (c *setup) GetActionGroups() (actions [][]lite.LiteAction) {
	serialized := c.envGroups[constants.EnvGroupActions][constants.EnvActions]
	if serialized == "" {
		return
	}
	err := json.Unmarshal([]byte(serialized), &actions)
	if err != nil {
		panic(fmt.Sprintf("failed to read the actions from Pod: %s", err.Error()))
	}
	return actions
}

func (c *setup) UseEnv(group string) {
	c.UseBaseEnv()
	c.envSelectedGroup = group

	envTemplates := map[string]string{}
	envResolutions := map[string]expressions.Expression{}
	for k, v := range c.envGroups[group] {
		if _, ok := c.envGroupsComputed[group][k]; ok {
			envTemplates[k] = v
		} else {
			os.Setenv(k, v)
		}
	}

	// Configure PWD variable, to make it similar to shell environment variables
	cwd := getWorkingDir()
	if os.Getenv("PWD") == "" {
		os.Setenv("PWD", cwd)
	}

	// Ensure the built-in binaries are available
	if os.Getenv("PATH") == "" {
		os.Setenv("PATH", data.InternalBinPath)
	} else {
		os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), data.InternalBinPath))
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
			output.ExitErrorf(data.CodeInputError, "failed to compute '%s' environment variable: %s", name, err.Error())
		}
		str, _ := value.Static().StringValue()
		os.Setenv(name, str)
	}
}

func (c *setup) UseCurrentEnv() {
	c.UseEnv(fmt.Sprintf("%d", c.envCurrentGroup))
}

func (c *setup) AdvanceEnv() {
	c.envCurrentGroup++
	c.UseCurrentEnv()
}

func (c *setup) SetWorkingDir(workingDir string) {
	_ = os.Chdir(defaultWorkingDir)
	if workingDir == "" {
		return
	}
	wd, err := filepath.Abs(workingDir)
	if err != nil {
		wd = workingDir
		_ = os.MkdirAll(wd, 0755)
	} else {
		err = os.MkdirAll(wd, 0755)
	}

	if err != nil {
		output.Std.Direct().Warnf("warn: error using %s as working directory: %s\n", workingDir, err.Error())
	}
}

func (c *setup) SetConfig(config lite.LiteContainerConfig) {
	if config.WorkingDir == nil || *config.WorkingDir == "" {
		c.SetWorkingDir("")
	} else {
		c.SetWorkingDir(*config.WorkingDir)
	}
}
