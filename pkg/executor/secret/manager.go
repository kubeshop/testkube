package secret

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	corev1 "k8s.io/api/core/v1"
)

// Manager is responsible for exchanging secrets with executor pod
type Manager interface {
	// Prepare prepares secret env vars based on secret envs and variables
	Prepare(secretEnvs map[string]string, variables map[string]testkube.Variable) (secretEnvVars []corev1.EnvVar)
	// GetEnvs get secret envs
	GetEnvs() (secretEnvs []string)
	// GetVars gets secret vars
	GetVars(variables map[string]testkube.Variable)
}

// NewEnvManager returns an implementation of the Manager
func NewEnvManager() *EnvManager {

	return &EnvManager{}
}

// EnvManager manages secret exchange from job pods using env
type EnvManager struct {
}

// Prepare prepares secret env vars based on secret envs and variables
func (m EnvManager) Prepare(secretEnvs map[string]string, variables map[string]testkube.Variable) (secretEnvVars []corev1.EnvVar) {
	// preparet secret envs
	i := 1
	for secretName, secretVar := range secretEnvs {
		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: fmt.Sprintf("RUNNER_SECRET_ENV%d", i),
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

	// prepare secret vars
	for name, variable := range variables {
		if variable.Type_ != testkube.VariableTypeSecret && variable.SecretRef != nil {
			continue
		}

		secretEnvVars = append(secretEnvVars, corev1.EnvVar{
			Name: fmt.Sprintf("RUNNER_SECRET_VAR_%s", name),
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

// GetEnvs gets secret envs
func (m EnvManager) GetEnvs() (secretEnvs []string) {
	i := 1
	for {
		secretEnv, ok := os.LookupEnv(fmt.Sprintf("RUNNER_SECRET_ENV%d", i))
		if !ok {
			break
		}

		secretEnvs = append(secretEnvs, secretEnv)
		i++
	}

	return secretEnvs
}

// GetVars gets secret vars
func (m EnvManager) GetVars(variables map[string]testkube.Variable) {
	for name, variable := range variables {
		if variable.Type_ != testkube.VariableTypeSecret {
			continue
		}

		value, ok := os.LookupEnv(fmt.Sprintf("RUNNER_SECRET_VAR_%s", name))
		if !ok {
			continue
		}

		variable.Value = value
		variables[name] = variable
	}

	return
}
