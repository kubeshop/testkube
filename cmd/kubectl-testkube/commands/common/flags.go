package common

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
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

	return
}
