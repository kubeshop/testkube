package common

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
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

func PopulateMasterFlags(cmd *cobra.Command, opts *HelmOptions) {
	var (
		apiURIPrefix, uiURIPrefix, agentURIPrefix, rootDomain string
		insecure                                              bool
	)

	cmd.Flags().BoolVar(&insecure, "cloud-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().MarkDeprecated("cloud-insecure", "use --master-insecure instead")
	cmd.Flags().StringVar(&agentURIPrefix, "cloud-agent-prefix", "agent", "defaults to 'agent', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-agent-prefix", "use --agent-prefix instead")
	cmd.Flags().StringVar(&apiURIPrefix, "cloud-api-prefix", "api", "defaults to 'api', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-api-prefix", "use --api-prefix instead")
	cmd.Flags().StringVar(&uiURIPrefix, "cloud-ui-prefix", "ui", "defaults to 'ui', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-ui-prefix", "use --ui-prefix instead")
	cmd.Flags().StringVar(&rootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-root-domain", "use --root-domain instead")

	cmd.Flags().BoolVar(&opts.Master.Insecure, "master-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().StringVar(&opts.Master.AgentUrlPrefix, "agent-prefix", "agent", "defaults to 'agent', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.ApiUrlPrefix, "api-prefix", "api", "defaults to 'api', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.UiUrlPrefix, "ui-prefix", "ui", "defaults to 'ui', usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.RootDomain, "root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for custom cloud mode]")

	cmd.Flags().StringVar(&opts.Master.URIs.Agent, "agent-uri", "", "Testkube Cloud agent URI [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.AgentToken, "agent-token", "", "Testkube Cloud agent key [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.OrgId, "org-id", "", "Testkube Cloud organization id [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.EnvId, "env-id", "", "Testkube Cloud environment id [required for centralized mode]")

}

func ProcessMasterFlags(cmd *cobra.Command, opts *HelmOptions, cfg *config.Data) {
	configured := cfg != nil
	if !cmd.Flags().Changed("master-insecure") {
		if cmd.Flags().Changed("cloud-insecure") {
			opts.Master.Insecure = cmd.Flag("cloud-insecure").Value.String() == "true"
		} else if configured && cfg.Master.Insecure {
			opts.Master.Insecure = cfg.Master.Insecure
		}
	}

	if !cmd.Flags().Changed("agent-prefix") {
		if cmd.Flags().Changed("cloud-agent-prefix") {
			opts.Master.AgentUrlPrefix = cmd.Flag("cloud-agent-prefix").Value.String()
		} else if configured && cfg.Master.AgentUrlPrefix != "" {
			opts.Master.AgentUrlPrefix = cfg.Master.AgentUrlPrefix
		}
	}

	if !cmd.Flags().Changed("api-prefix") {
		if cmd.Flags().Changed("cloud-api-prefix") {
			opts.Master.ApiUrlPrefix = cmd.Flag("cloud-api-prefix").Value.String()
		} else if configured && cfg.Master.ApiUrlPrefix != "" {
			opts.Master.ApiUrlPrefix = cfg.Master.ApiUrlPrefix
		}
	}

	if !cmd.Flags().Changed("ui-prefix") {
		if cmd.Flags().Changed("cloud-ui-prefix") {
			opts.Master.UiUrlPrefix = cmd.Flag("cloud-ui-prefix").Value.String()
		} else if configured && cfg.Master.UiUrlPrefix != "" {
			opts.Master.UiUrlPrefix = cfg.Master.UiUrlPrefix
		}
	}

	if !cmd.Flags().Changed("root-domain") {
		if cmd.Flags().Changed("cloud-root-domain") {
			opts.Master.RootDomain = cmd.Flag("cloud-root-domain").Value.String()
		} else if configured && cfg.Master.RootDomain != "" {
			opts.Master.RootDomain = cfg.Master.RootDomain
		}
	}

	opts.Master.URIs = NewMasterUris(opts.Master.ApiUrlPrefix,
		opts.Master.UiUrlPrefix,
		opts.Master.AgentUrlPrefix,
		opts.Master.RootDomain,
		opts.Master.Insecure)
}
