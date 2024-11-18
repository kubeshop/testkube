package testworkflowprocessor

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

func createSecretMachine(mapEnvs map[string]corev1.EnvVarSource) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction("secret", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 3 {
				if values[2].IsBool() {
					computed, _ = values[2].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"secret" function expects 3rd argument to be boolean, %s provided`, values[2].String())
				}
			} else if len(values) != 2 {
				return nil, true, fmt.Errorf(`"secret" function expects 2-3 arguments, %d provided`, len(values))
			}

			secretName, _ := values[0].StringValue()
			keyName, _ := values[1].StringValue()
			strs := []string{secretName, keyName}
			for i := range strs {
				j := 0
				for _, char := range []string{"-", "."} {
					for ; strings.Contains(strs[i], char); j++ {
						strs[i] = strings.Replace(strs[i], char, fmt.Sprintf("_%d_", j), 1)
					}
				}
			}

			// TODO(TKC-2585): Avoid adding the secrets to all the groups with virtual 02 group
			envName := fmt.Sprintf("%s_K_%s", strs[0], strs[1])
			internalName := actiontypes.EnvName(constants.EnvGroupSecrets, computed, true, envName)
			mapEnvs[internalName] = corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: keyName,
				},
			}
			v, err := expressions.Compile("env." + envName)
			return v, true, err
		})

}
