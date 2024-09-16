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
	assert.Equal(t, "{{"+expressions.InternalFnCall+"env.S_N_name_0_one_1_two_K_key_0_three_1_four}}", MustCall(machine, "secret", "name-one.two", "key-three.four"))
	assert.EqualValues(t, map[string]corev1.EnvVarSource{
		"S_N_name_0_one_1_two_K_key_0_three_1_four": {
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "name-one.two",
				},
				Key: "key-three.four",
			},
		},
	}, mapEnvs)
}
