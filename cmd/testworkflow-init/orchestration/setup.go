package orchestration

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

var (
	scopedRegex              = regexp.MustCompile(`^_(00|01|02|03|04|05|\d|[1-9]\d*)(C)?(S?)_`)
	Setup                    *setup
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

// Initialize must be called before using Setup.
// In production, this is called early in main().
// In tests, this can be called after setting up test environment.
func Initialize() {
	if Setup == nil {
		Setup = newSetup()
		Setup.initialize()
	}
}

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
	envAdditionalSensitive map[string]struct{}
	envCurrentGroup        int
	envSelectedGroup       string
	minSensitiveWordLength int
}

func newSetup() *setup {
	return &setup{
		envBase:                map[string]string{},
		envGroups:              map[string]map[string]string{},
		envGroupsComputed:      map[string]map[string]struct{}{},
		envGroupsSensitive:     map[string]map[string]struct{}{},
		envAdditionalSensitive: map[string]struct{}{},
		envCurrentGroup:        -1,
		minSensitiveWordLength: 1,
	}
}

func (c *setup) initialize() {
	// Clear existing data
	c.envBase = map[string]string{}
	c.envGroups = map[string]map[string]string{}
	c.envGroupsComputed = map[string]map[string]struct{}{}
	c.envGroupsSensitive = map[string]map[string]struct{}{}
	c.envAdditionalSensitive = map[string]struct{}{}
	c.envCurrentGroup = -1

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

func (c *setup) AddSensitiveWords(words ...string) {
	for i := range words {
		c.envAdditionalSensitive[words[i]] = struct{}{}
	}
}

// GetSensitiveWords returns the values to mask in the log stream.
//
// Values shorter than the configured minimum are left out on purpose: masking two
// characters rewrites nearly every line they appear in, which destroys the logs while
// hiding almost nothing. This is a rendering decision - it must not be reused to decide
// what may leave the pod. See GetSensitiveValues.
func (c *setup) GetSensitiveWords() []string {
	return c.sensitiveValues(c.minSensitiveWordLength)
}

// GetSensitiveValues returns every value marked sensitive, whatever its length.
//
// This is the set for deciding what may leave the pod. A step output published into the
// execution record is readable by everyone who can read that execution, and readable
// from another workflow through execution().outputs, so a short secret has to count:
// missing it publishes the secret, while matching on it costs a withheld output, which
// is announced rather than silent.
func (c *setup) GetSensitiveValues() []string {
	return c.sensitiveValues(0)
}

// sensitiveValues collects the values marked sensitive, skipping the ones shorter than
// minLength. Words handed over at runtime - the values resolved credential() calls
// produced - are never skipped, whatever minLength says.
func (c *setup) sensitiveValues(minLength int) []string {
	words := make([]string, 0, len(c.envAdditionalSensitive))
	for value := range c.envAdditionalSensitive {
		words = append(words, value)
	}
	appendIfSensitive := func(value string) {
		if value == "" || len(value) < minLength {
			return
		}
		words = append(words, value)
	}
	for _, name := range commonSensitiveVariables {
		appendIfSensitive(os.Getenv(name))
	}
	for k := range c.envBase {
		if _, ok := c.envGroupsSensitive[c.envSelectedGroup][k]; ok {
			appendIfSensitive(os.Getenv(k))
		}
	}
	for k := range c.envGroups[c.envSelectedGroup] {
		if _, ok := c.envGroupsSensitive[c.envSelectedGroup][k]; ok {
			appendIfSensitive(os.Getenv(k))
		}
	}
	// TODO(TKC-2585): Avoid adding the secrets to all the groups without isolation
	for k := range c.envGroups[constants.EnvGroupSecrets] {
		if _, ok := c.envGroupsSensitive[constants.EnvGroupSecrets][k]; ok {
			appendIfSensitive(os.Getenv(k))
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

func (c *setup) GetInternalConfig() (config testworkflowconfig.InternalConfig) {
	serialized := c.envGroups[constants.EnvGroupInternal][constants.EnvInternalConfig]
	if serialized == "" {
		return
	}
	err := json.Unmarshal([]byte(serialized), &config)
	if err != nil {
		panic(fmt.Sprintf("failed to read the internal config from Pod: %s", err.Error()))
	}
	return config
}

func (c *setup) GetSignature() (config []testworkflowconfig.SignatureConfig) {
	serialized := c.envGroups[constants.EnvGroupInternal][constants.EnvSignature]
	if serialized == "" {
		return
	}
	err := json.Unmarshal([]byte(serialized), &config)
	if err != nil {
		panic(fmt.Sprintf("failed to read the signature from Pod: %s", err.Error()))
	}
	return config
}

func (c *setup) GetContainerResources() (config testworkflowconfig.ContainerResourceConfig) {
	config.Requests.CPU = c.envGroups[constants.EnvGroupResources][constants.EnvResourceRequestsCPU]
	config.Requests.Memory = c.envGroups[constants.EnvGroupResources][constants.EnvResourceRequestsMemory]
	config.Limits.CPU = c.envGroups[constants.EnvGroupResources][constants.EnvResourceLimitsCPU]
	config.Limits.Memory = c.envGroups[constants.EnvGroupResources][constants.EnvResourceLimitsMemory]
	return config
}

func (c *setup) UseEnv(group string) error {
	c.UseBaseEnv()
	c.envSelectedGroup = group

	envTemplates := map[string]string{}
	envResolutions := map[string]expressions.Expression{}
	for k, v := range c.envGroups[group] {
		if _, ok := c.envGroupsComputed[group][k]; ok {
			envTemplates[k] = v
		} else {
			if err := checkWithheldEnv(k, v); err != nil {
				return err
			}
			os.Setenv(k, v)
		}
	}

	// TODO(TKC-2585): Avoid adding the secrets to all the groups without isolation
	for k, v := range c.envGroups[constants.EnvGroupSecrets] {
		if _, ok := c.envGroupsComputed[constants.EnvGroupSecrets][k]; ok {
			envTemplates[k] = v
		} else {
			if err := checkWithheldEnv(k, v); err != nil {
				return err
			}
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
		os.Setenv("PATH", constants.InternalBinPath)
	} else {
		os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), constants.InternalBinPath))
	}

	// Compute dynamic environment variables
	addonMachine := expressions.CombinedMachines(data.RefSuccessMachine, data.AliasMachine, data.StateMachine, libs.NewFsMachine(os.DirFS("/"), cwd),
		data.ExecutionDataMachine(),
		credentials.NewCredentialMachine(data.Credentials(), func(_ string, value string) {
			c.AddSensitiveWords(value)
			output.Std.SetSensitiveWords(c.GetSensitiveWords())
		}))
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
			return errors.Wrapf(err, "failed to compute '%s' environment variable", name)
		}
		str, _ := value.Static().StringValue()
		if err := checkWithheldEnv(name, str); err != nil {
			return err
		}
		os.Setenv(name, str)
	}
	return nil
}

// checkWithheldEnv refuses to install a variable holding a withheld marker.
//
// An output another workflow withheld resolves to a marker instead of the value it was
// meant to carry. Installing it would hand the marker to the tool as the value of the
// variable, where nothing looks at it again - the command guard only sees arguments,
// not what arrives through the environment. A plain value is checked as well as a
// computed one: a step that spawns workers resolves their specification itself, so a
// marker reaches the worker already baked in as a literal.
func checkWithheldEnv(name, value string) error {
	if !executiondata.IsWithheldMarker(value) {
		return nil
	}
	return executiondata.WithheldError(fmt.Sprintf("the %q environment variable", name), executiondata.WithheldMarkersIn(value))
}

func (c *setup) UseCurrentEnv() error {
	return c.UseEnv(fmt.Sprintf("%d", c.envCurrentGroup))
}

func (c *setup) AdvanceEnv() error {
	c.envCurrentGroup++
	return c.UseCurrentEnv()
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
		_ = os.MkdirAll(wd, 0755)
	}
	err = os.Chdir(wd)

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

func (c *setup) GetSecretVolumeData(mountPaths []string) []string {
	wordMap := make(map[string]struct{})
	for _, dir := range mountPaths {
		err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				output.Std.Direct().Warnf("warn: error reading %s as secret volume key: %s\n", path, err.Error())
				return nil
			}

			wordMap[string(data)] = struct{}{}
			return nil
		})

		if err != nil {
			output.Std.Direct().Warnf("warn: error using %s as secret volume path: %s\n", dir, err.Error())
		}
	}

	words := make([]string, len(wordMap))
	for word := range wordMap {
		words = append(words, word)
	}

	return words
}

func (c *setup) GetContainerName() string {
	return c.envGroups[constants.EnvGroupRuntime][constants.EnvContainerName]
}
