package libs

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	GlobalSecretGroup = "02"
)

// FIXME: it's copied from actiontypes to avoid import cycle
func EnvName(group string, computed bool, sensitive bool, name string) string {
	suffix := ""
	if computed {
		suffix = "C"
	}
	if sensitive {
		suffix += "S"
	}
	return fmt.Sprintf("_%s%s_%s", group, suffix, name)
}

// TODO: Probably it shouldn't be part of expressions lib, as it's actually workflow processing
func NewSecretMachine(mapEnvs map[string]corev1.EnvVarSource) expressions.Machine {
	return expressions.NewMachine().
		RegisterFunction("secret", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			computed := false
			if len(values) == 3 {
				if values[2].IsBool() {
					computed, _ = values[2].BoolValue()
				} else {
					return nil, true, fmt.Errorf(`"secret" function expects 3rd argument to be boolean, %s provided`, values[3].String())
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

			// TODO: Avoid adding the secrets to all the groups with virtual SN group
			envName := fmt.Sprintf("%s_K_%s", strs[0], strs[1])
			internalName := EnvName(GlobalSecretGroup, computed, true, envName)
			mapEnvs[internalName] = corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: keyName,
				},
			}

			return expressions.NewValue(fmt.Sprintf("{{%senv.%s}}", expressions.InternalFnCall, envName)), true, nil
		})

}
