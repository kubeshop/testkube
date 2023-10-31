package env

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	// SecretEnvVarPrefix is a prefix for secret env vars
	SecretEnvVarPrefix = "RUNNER_SECRET_ENV"
	// SecretVarPrefix is a prefix for secret vars
	SecretVarPrefix = "RUNNER_SECRET_VAR_"
	// ConfigMapVarPrefix is a prefix for config map vars
	ConfigMapVarPrefix = "RUNNER_CONFIGMAP_VAR_"
	// GitUsernameEnvVarName is git username environment var name
	GitUsernameEnvVarName = "RUNNER_GITUSERNAME"
	// GitTokenEnvVarName is git token environment var name
	GitTokenEnvVarName = "RUNNER_GITTOKEN"
)

// Interface is responsible for exchanging envs and vars with executor pod
type Interface interface {
	// PrepareSecrets prepares secret env vars based on secret envs and variables
	PrepareSecrets(secretEnvs map[string]string, variables map[string]testkube.Variable) (secretEnvVars []corev1.EnvVar)
	// PrepareEnvs prepares env vars based on envs and variables
	PrepareEnvs(envs map[string]string, variables map[string]testkube.Variable) []corev1.EnvVar
	// PrepareGitCredentials prepares git credentials
	PrepareGitCredentials(usernameSecret, tokenSecret *testkube.SecretRef) (envVars []corev1.EnvVar)
	// GetSecretEnvs get secret envs
	GetSecretEnvs() (secretEnvs map[string]string)
	// GetReferenceVars gets reference vars
	GetReferenceVars(variables map[string]testkube.Variable)
	// ObfuscateSecrets obfuscates secret values
	ObfuscateSecrets(p []byte) []byte
	// ObfuscateStringSlice obfuscates string slice values
	ObfuscateStringSlice(values []string) []string
}

// NewManager returns an implementation of the Manager
func NewManager() *Manager {
	return &Manager{}
}

// NewManagerWithVars returns an implementation of the Manager with variables
func NewManagerWithVars(variables map[string]testkube.Variable) *Manager {
	return &Manager{
		Variables: variables,
	}
}

// Manager manages secret and config map exchange from job pods using env
type Manager struct {
	Variables map[string]testkube.Variable
}

// PrepareSecrets prepares secret env vars based on secret envs and variables
func (m Manager) PrepareSecrets(secretEnvs map[string]string, variables map[string]testkube.Variable) (secretEnvVars []corev1.EnvVar) {
	// preparet secret envs
	i := 1
	// Deprecated: use Secret Variables instead
	for secretVar, secretName := range secretEnvs {
		// TODO: these are duplicated because Postman executor is expecting it as json string
		// and gets unmarshalled and the name and the value are taken from there, for other executors it will be like a normal env var.
		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: secretVar,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: secretVar,
				},
			},
		})

		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: fmt.Sprintf("%s%d", SecretEnvVarPrefix, i),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: secretVar,
				},
			},
		})
		i++
	}

	i = 1
	// prepare secret vars
	for name, variable := range variables {
		if !variable.IsSecret() || variable.SecretRef == nil {
			continue
		}

		// TODO: these are duplicated because Postman executor is expecting it as json string
		// and gets unmarshalled and the name and the value are taken from there, for other executors it will be like a normal env var.
		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: name,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: variable.SecretRef.Name,
					},
					Key: variable.SecretRef.Key,
				},
			},
		})

		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: fmt.Sprintf("%s%d", SecretEnvVarPrefix, i),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: variable.SecretRef.Name,
					},
					Key: variable.SecretRef.Key,
				},
			},
		})
		i++

		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: fmt.Sprintf("%s%s", SecretVarPrefix, name),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: variable.SecretRef.Name,
					},
					Key: variable.SecretRef.Key,
				},
			},
		})
	}

	return secretEnvVars
}

// PrepareEnvs prepares env vars based on envs and variables
func (m Manager) PrepareEnvs(envs map[string]string, variables map[string]testkube.Variable) []corev1.EnvVar {
	var env []corev1.EnvVar
	for k, v := range envs {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	// prepare vars
	for name, variable := range variables {
		if variable.IsSecret() {
			continue
		}

		if variable.ConfigMapRef == nil {
			env = append(env, corev1.EnvVar{
				Name:  name,
				Value: variable.Value,
			})
		} else {
			env = append(env, corev1.EnvVar{
				Name: name,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: variable.ConfigMapRef.Name,
						},
						Key: variable.ConfigMapRef.Key,
					},
				},
			})

			env = append(env, corev1.EnvVar{
				Name: fmt.Sprintf("%s%s", ConfigMapVarPrefix, name),
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: variable.ConfigMapRef.Name,
						},
						Key: variable.ConfigMapRef.Key,
					},
				},
			})
		}
	}

	return env
}

// PrepareGitCredentials prepares git credentials
func (m Manager) PrepareGitCredentials(usernameSecret, tokenSecret *testkube.SecretRef) (envVars []corev1.EnvVar) {
	var data = []struct {
		envVar    string
		secretRef *testkube.SecretRef
	}{
		{
			GitUsernameEnvVarName,
			usernameSecret,
		},
		{
			GitTokenEnvVarName,
			tokenSecret,
		},
	}

	for _, value := range data {
		if value.secretRef != nil {
			envVars = append(envVars, corev1.EnvVar{
				Name: value.envVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: value.secretRef.Name,
						},
						Key: value.secretRef.Key,
					},
				},
			})
		}
	}

	return envVars
}

// GetSecretEnvs gets secret envs
func (m Manager) GetSecretEnvs() (secretEnvs map[string]string) {
	secretEnvs = make(map[string]string, 0)
	i := 1
	for {
		envName := fmt.Sprintf("%s%d", SecretEnvVarPrefix, i)
		secretEnv, ok := os.LookupEnv(envName)
		if !ok {
			break
		}

		secretEnvs[envName] = secretEnv
		i++
	}

	return secretEnvs
}

// GetReferenceVars gets reference vars
func (m Manager) GetReferenceVars(variables map[string]testkube.Variable) {
	for name, variable := range variables {
		if variable.IsSecret() {
			value, ok := os.LookupEnv(fmt.Sprintf("%s%s", SecretVarPrefix, name))
			if !ok {
				continue
			}

			variable.Value = value
			variables[name] = variable
		} else {
			value, ok := os.LookupEnv(fmt.Sprintf("%s%s", ConfigMapVarPrefix, name))
			if !ok {
				continue
			}

			variable.Value = value
			variables[name] = variable
		}
	}

	return
}

// ObfuscateSecrets obfuscates secret values
func (m Manager) ObfuscateSecrets(p []byte) []byte {
	if m.Variables == nil {
		return p
	}

	for _, variable := range m.Variables {
		if !variable.IsSecret() {
			continue
		}

		p = bytes.ReplaceAll(p, []byte(variable.Value), []byte(strings.Repeat("*", len(variable.Value))))
	}

	return p
}

// ObfuscateStringSlice obfuscates string slice values
func (m Manager) ObfuscateStringSlice(values []string) []string {
	if m.Variables == nil {
		return values
	}

	var results []string
	for _, value := range values {
		for _, variable := range m.Variables {
			if !variable.IsSecret() {
				continue
			}

			value = strings.ReplaceAll(value, variable.Value, strings.Repeat("*", len(variable.Value)))
		}

		results = append(results, value)
	}

	return results
}
