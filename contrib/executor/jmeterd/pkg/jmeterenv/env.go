package jmeterenv

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
