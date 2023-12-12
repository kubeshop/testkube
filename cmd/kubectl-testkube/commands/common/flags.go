package common

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func CreateVariables(cmd *cobra.Command, ignoreSecretVariable bool) (vars map[string]testkube.Variable, err error) {
	basicParams, err := cmd.Flags().GetStringToString("variable")
	if err != nil {
		return vars, err
	}

	vars = map[string]testkube.Variable{}

	for k, v := range basicParams {
		vars[k] = testkube.NewBasicVariable(k, v)
	}

	if !ignoreSecretVariable {
		secretParams, err := cmd.Flags().GetStringToString("secret-variable")
		if err != nil {
			return vars, err
		}
		for k, v := range secretParams {
			vars[k] = testkube.NewSecretVariable(k, v)
		}
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

func PopulateProUriFlags(cmd *cobra.Command, opts *HelmOptions) {

	cmd.Flags().BoolVar(&opts.CloudClientInsecure, "cloud-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().StringVar(&opts.CloudAgentUrlPrefix, "cloud-agent-prefix", "agent", "defaults to 'agent', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.CloudApiUrlPrefix, "cloud-api-prefix", "api", "defaults to 'api', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.CloudUiUrlPrefix, "cloud-ui-prefix", "ui", "defaults to 'ui', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.CloudRootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for custom cloud mode]")

	opts.CloudUris = NewCloudUris(opts.CloudApiUrlPrefix, opts.CloudUiUrlPrefix, opts.CloudAgentUrlPrefix, opts.CloudRootDomain, opts.CloudClientInsecure)
}
