package testworkflowresolver

import (
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

func HasEnvVar(envs []testworkflowsv1.EnvVar, name string) bool {
	for i := range envs {
		if envs[i].Name == name {
			return true
		}
	}
	return false
}

func DedupeEnvVars(envs []testworkflowsv1.EnvVar) []testworkflowsv1.EnvVar {
	exists := map[string]struct{}{}
	result := make([]testworkflowsv1.EnvVar, 0)
	for i := len(envs) - 1; i >= 0; i-- {
		if _, ok := exists[envs[i].Name]; !ok {
			exists[envs[i].Name] = struct{}{}
			result = append([]testworkflowsv1.EnvVar{envs[i]}, result...)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
