package common

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func CreateVariables(cmd *cobra.Command, ignoreSecretVariable bool) (vars map[string]testkube.Variable, err error) {
	basicParams, err := cmd.Flags().GetStringArray("variable")
	if err != nil {
		return vars, err
	}

	vars = map[string]testkube.Variable{}

	for _, v := range basicParams {
		values := strings.SplitN(v, "=", 2)
		if len(values) != 2 {
			return vars, errors.New("wrong number of variable params")
		}

		vars[values[0]] = testkube.NewBasicVariable(values[0], values[1])
	}

	if !ignoreSecretVariable {
		secretParams, err := cmd.Flags().GetStringArray("secret-variable")
		if err != nil {
			return vars, err
		}

		for _, v := range secretParams {
			values := strings.SplitN(v, "=", 2)
			if len(values) != 2 {
				return vars, errors.New("wrong number of secret variable params")
			}

			vars[values[0]] = testkube.NewSecretVariable(values[0], values[1])
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
		apiURIPrefix, uiURIPrefix, agentURIPrefix, cloudRootDomain, proRootDomain string
		insecure                                                                  bool
	)

	cmd.Flags().BoolVar(&insecure, "cloud-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().MarkDeprecated("cloud-insecure", "use --master-insecure instead")
	cmd.Flags().StringVar(&agentURIPrefix, "cloud-agent-prefix", defaultAgentPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-agent-prefix", "use --agent-prefix instead")
	cmd.Flags().StringVar(&apiURIPrefix, "cloud-api-prefix", defaultApiPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-api-prefix", "use --api-prefix instead")
	cmd.Flags().StringVar(&uiURIPrefix, "cloud-ui-prefix", defaultUiPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-ui-prefix", "use --ui-prefix instead")
	cmd.Flags().StringVar(&cloudRootDomain, "cloud-root-domain", defaultRootDomain, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().MarkDeprecated("cloud-root-domain", "use --root-domain instead")
	cmd.Flags().StringVar(&proRootDomain, "pro-root-domain", defaultRootDomain, "usually don't need to be changed [required for custom pro mode]")
	cmd.Flags().MarkDeprecated("pro-root-domain", "use --root-domain instead")

	cmd.Flags().BoolVar(&opts.Master.Insecure, "master-insecure", false, "should client connect in insecure mode (will use http instead of https)")
	cmd.Flags().StringVar(&opts.Master.AgentUrlPrefix, "agent-prefix", defaultAgentPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.ApiUrlPrefix, "api-prefix", defaultApiPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.LogsUrlPrefix, "logs-prefix", defaultLogsPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.UiUrlPrefix, "ui-prefix", defaultUiPrefix, "usually don't need to be changed [required for custom cloud mode]")
	cmd.Flags().StringVar(&opts.Master.RootDomain, "root-domain", defaultRootDomain, "usually don't need to be changed [required for custom cloud mode]")

	cmd.Flags().StringVar(&opts.Master.URIs.Agent, "agent-uri", "", "Testkube Pro agent URI [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.URIs.Logs, "logs-uri", "", "Testkube Pro logs URI [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.AgentToken, "agent-token", "", "Testkube Pro agent key [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.OrgId, "org-id", "", "Testkube Pro organization id [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.EnvId, "env-id", "", "Testkube Pro environment id [required for centralized mode]")

	cmd.Flags().BoolVar(&opts.Master.Features.LogsV2, "feature-logs-v2", false, "Logs v2 feature flag")
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
		switch {
		case cmd.Flags().Changed("pro-root-domain"):
			opts.Master.RootDomain = cmd.Flag("pro-root-domain").Value.String()
		case cmd.Flags().Changed("cloud-root-domain"):
			opts.Master.RootDomain = cmd.Flag("cloud-root-domain").Value.String()
		case configured && cfg.Master.RootDomain != "":
			opts.Master.RootDomain = cfg.Master.RootDomain
		}
	}

	opts.Master.URIs = NewMasterUris(opts.Master.ApiUrlPrefix,
		opts.Master.UiUrlPrefix,
		opts.Master.AgentUrlPrefix,
		opts.Master.LogsUrlPrefix,
		opts.Master.URIs.Agent,
		opts.Master.URIs.Logs,
		opts.Master.RootDomain,
		opts.Master.Insecure)
}
