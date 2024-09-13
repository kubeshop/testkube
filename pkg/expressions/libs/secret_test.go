package libs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestSecret(t *testing.T) {
	mapEnvs := make(map[string]corev1.EnvVarSource)
	machine := NewSecretMachine(mapEnvs)
	assert.Equal(t, "{{"+expressions.InternalFnCall+"env.S_N_name_one_K_key_0_two_1_three}}", MustCall(machine, "secret", "name-one", "key-two-three"))
	assert.EqualValues(t, map[string]corev1.EnvVarSource{
		"S_N_name_one_K_key_0_two_1_three": {
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "name-one",
				},
				Key: "key-two-three",
			},
		},
	}, mapEnvs)
}
