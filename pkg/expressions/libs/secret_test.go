package libs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSecret(t *testing.T) {
	mapEnvs := make(map[string]corev1.EnvVarSource)
	machine := NewSecretMachine(mapEnvs)
	assert.Equal(t, "{{env.S_N_name_K_key}}", MustCall(machine, "secret", "name", "key"))
	assert.EqualValues(t, map[string]corev1.EnvVarSource{
		"S_N_name_K_key": {
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "name",
				},
				Key: "key",
			},
		},
	}, mapEnvs)
}
