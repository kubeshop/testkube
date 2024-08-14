package libs

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func NewSecretMachine(mapEnvs map[string]corev1.EnvVarSource) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction("secret", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			if len(values) != 2 {
				return nil, true, fmt.Errorf(`"secret" function expects 2 arguments, %d provided`, len(values))
			}

			secretName, _ := values[0].StringValue()
			keyName, _ := values[1].StringValue()
			envName := fmt.Sprintf("S_N_%s_K_%s", secretName, keyName)
			mapEnvs[envName] = corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: keyName,
				},
			}

			return expressions.NewValue(fmt.Sprintf("{{env.%s}}", envName)), true, nil
		})

}
