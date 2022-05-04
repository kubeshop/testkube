package mapper

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

func NewVariablesFromCmd(cmd *cobra.Command) *[]testkube.Variable {
	vars := []testkube.Variable{}
	variables, err := cmd.Flags().GetStringToString("variable")
	if err == nil {
		for k, v := range variables {
			vars = append(vars, testkube.NewBasicVariable(k, v))
		}
	}
	secretVariables, err := cmd.Flags().GetStringToString("secret-variable")
	if err == nil {
		for k, v := range secretVariables {
			vars = append(vars, testkube.NewSecretVariable(k, v))
		}
	}

	return &vars
}
