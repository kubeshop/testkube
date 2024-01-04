package slaves

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	exenv "github.com/kubeshop/testkube/pkg/executor/env"
)

const (
	MasterOverrideJvmArgs      = "MASTER_OVERRIDE_JVM_ARGS"
	MasterAdditionalJvmArgs    = "MASTER_ADDITIONAL_JVM_ARGS"
	SlavesOverrideJvmArgs      = "SLAVES_OVERRIDE_JVM_ARGS"
	SlavesAdditionalJvmArgs    = "SLAVES_ADDITIONAL_JVM_ARGS"
	SlavesAdditionalJmeterArgs = "SLAVES_ADDITIONAL_JMETER_ARGS"
	SlavesCount                = "SLAVES_COUNT"
	MasterPrefix               = "MASTER_"
	SlavesPrefix               = "SLAVES_"
	RunnerPrefix               = "RUNNER_"
	HttpProxyPrefix            = "HTTP_PROXY="
	HttpsProxyPrefix           = "HTTPS_PROXY="
	DebugPrefix                = "DEBUG="
)

// ExtractSlaveEnvVariables removes slave environment variables from the given map and returns them separately.
func ExtractSlaveEnvVariables(variables map[string]testkube.Variable) map[string]testkube.Variable {
	slaveVariables := make(map[string]testkube.Variable)

	// Iterate through the variables to extract slave environment variables.
	for k, v := range variables {
		switch {
		case strings.HasPrefix(k, SlavesPrefix):
			slaveVariables[k] = v
			delete(variables, k) // Remove slave variable from the main variables map.
		case strings.HasPrefix(k, MasterPrefix):
			continue
		default:
			slaveVariables[k] = v
		}
	}
	return slaveVariables
}

// GetRunnerEnvVariables returns runner env variables
func GetRunnerEnvVariables() map[string]string {
	envVars := make(map[string]string)
	envs := os.Environ()
OuterLoop:
	for _, env := range envs {
		for _, prefix := range []string{exenv.SecretEnvVarPrefix, exenv.SecretVarPrefix,
			exenv.GitUsernameEnvVarName, exenv.GitTokenEnvVarName} {
			if strings.HasPrefix(env, prefix) {
				continue OuterLoop
			}
		}

		for _, prefix := range []string{RunnerPrefix, HttpProxyPrefix, HttpsProxyPrefix, DebugPrefix} {
			if strings.HasPrefix(env, prefix) {
				pair := strings.SplitN(env, "=", 2)
				if len(pair) != 2 {
					continue OuterLoop
				}

				envVars[pair[0]] += pair[1]
			}
		}
	}

	envVars["CI"] = "1"
	return envVars
}
