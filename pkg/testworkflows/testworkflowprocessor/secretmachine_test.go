package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestSecret(t *testing.T) {
	mapEnvs := make(map[string]corev1.EnvVarSource)
	machine := createSecretMachine(mapEnvs)
	call, err := expressions.CompileAndResolve(`secret("name-one.two", "key-three.four")`, machine)
	assert.NoError(t, err)
	assert.Equal(t, "env.name_0_one_1_two_K_key_0_three_1_four", call.String())
	assert.EqualValues(t, map[string]corev1.EnvVarSource{
		"_02S_name_0_one_1_two_K_key_0_three_1_four": {
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "name-one.two",
				},
				Key: "key-three.four",
			},
		},
	}, mapEnvs)
}

func TestSecretComputed(t *testing.T) {
	mapEnvs := make(map[string]corev1.EnvVarSource)
	machine := createSecretMachine(mapEnvs)
	call, err := expressions.CompileAndResolve(`secret("name-one.two", "key-three.four", true)`, machine)
	assert.NoError(t, err)
	assert.Equal(t, "env.name_0_one_1_two_K_key_0_three_1_four", call.String())
	assert.EqualValues(t, map[string]corev1.EnvVarSource{
		"_02CS_name_0_one_1_two_K_key_0_three_1_four": {
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "name-one.two",
				},
				Key: "key-three.four",
			},
		},
	}, mapEnvs)
}
