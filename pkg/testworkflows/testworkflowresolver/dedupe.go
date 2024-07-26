package testworkflowresolver

import (
	corev1 "k8s.io/api/core/v1"
)

func HasEnvVar(envs []corev1.EnvVar, name string) bool {
	for i := range envs {
		if envs[i].Name == name {
			return true
		}
	}
	return false
}

func DedupeEnvVars(envs []corev1.EnvVar) []corev1.EnvVar {
	exists := map[string]struct{}{}
	result := make([]corev1.EnvVar, 0)
	for i := len(envs) - 1; i >= 0; i-- {
		if _, ok := exists[envs[i].Name]; !ok {
			exists[envs[i].Name] = struct{}{}
			result = append([]corev1.EnvVar{envs[i]}, result...)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
