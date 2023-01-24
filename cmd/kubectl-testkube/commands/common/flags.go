package common

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func CreateVariables(cmd *cobra.Command) (vars map[string]testkube.Variable, err error) {
	basicParams, err := cmd.Flags().GetStringToString("variable")
	if err != nil {
		return vars, err
	}

	vars = map[string]testkube.Variable{}

	for k, v := range basicParams {
		vars[k] = testkube.NewBasicVariable(k, v)
	}

	secretParams, err := cmd.Flags().GetStringToString("secret-variable")
	if err != nil {
		return vars, err
	}
	for k, v := range secretParams {
		vars[k] = testkube.NewSecretVariable(k, v)
	}

	secretParamReferences, err := cmd.Flags().GetStringToString("secret-variable-reference")
	if err != nil {
		return vars, err
	}
	for k, v := range secretParamReferences {
		values := strings.Split(v, "=")
		if len(values) != 2 {
			return vars, errors.New("wrong number of secret reference params")
		}

		vars[k] = testkube.NewSecretVariableReference(k, values[0], values[1])
	}

	return
}
