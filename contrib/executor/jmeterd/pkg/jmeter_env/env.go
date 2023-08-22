package jmeter_env

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	MASTER_OVERRIDE_JVM_ARGS      = "MASTER_OVERRIDE_JVM_ARGS"
	MASTER_ADDITIONAL_JVM_ARGS    = "MASTER_ADDITIONAL_JVM_ARGS"
	SLAVES_OVERRIDE_JVM_ARGS      = "SLAVES_OVERRIDE_JVM_ARGS"
	SLAVES_ADDITIONAL_JVM_ARGS    = "SLAVES_ADDITIONAL_JVM_ARGS"
	SLAVES_ADDITIONAL_JMETER_ARGS = "SLAVES_ADDITIONAL_JMETER_ARGS"
	SLAVES_COUNT                  = "SLAVES_COUNT"
	GLOBAL_JMETER_PROPERTIES      = "GLOBAL_JMETER_PROPERTIES"
	MASTER_PREFIX                 = "MASTER_"
	SLAVES_PREFIX                 = "SLAVES_"
)

// ExtractSlaveEnvVariables removes slave environment variables from the given map and returns them separately.
func ExtractSlaveEnvVariables(variables map[string]testkube.Variable) map[string]testkube.Variable {
	slaveVariables := make(map[string]testkube.Variable)

	// Iterate through the variables to extract slave environment variables.
	for varName, v := range variables {
		switch {
		case strings.HasPrefix(varName, SLAVES_PREFIX):
			slaveVariables[varName] = v
			delete(variables, varName) // Remove slave variable from the main variables map.
		case varName == GLOBAL_JMETER_PROPERTIES || strings.HasPrefix(varName, MASTER_PREFIX):
			continue
		default:
			slaveVariables[varName] = v
		}
	}
	return slaveVariables
}

// Split JemeterProperties provided using ',' seperator
// Append -G to each jmeter properties
func FormatJmeterProperties(jmeterProperties string) []string {
	finalJmeterPopertiesArgs := []string{}
	jmeterArgs := strings.Split(jmeterProperties, ",")
	for _, jmeterArg := range jmeterArgs {
		finalJmeterPopertiesArgs = append(finalJmeterPopertiesArgs, fmt.Sprintf("-G%s", strings.TrimSpace(jmeterArg)))
	}
	return finalJmeterPopertiesArgs
}
