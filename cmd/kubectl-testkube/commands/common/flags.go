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

func PopulateMasterFlags(cmd *cobra.Command, opts *HelmOptions, isDockerCmd bool) {
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
	cmd.Flags().BoolVar(&opts.Master.CustomAuth, "custom-auth", false, "usually don't need to be changed [required for custom cloud mode]")

	// allow to override default values of all URIs
	cmd.Flags().String("api-uri-override", "", "api uri override")
	cmd.Flags().String("ui-uri-override", "", "ui uri override")
	cmd.Flags().String("auth-uri-override", "", "auth uri override")
	cmd.Flags().String("agent-uri-override", "", "agent uri override")
	cmd.Flags().String("logs-uri-override", "", "logs service uri override")

	agentURI := ""
	if isDockerCmd {
		agentURI = "agent.testkube.io:443"
	}

	cmd.Flags().StringVar(&opts.Master.URIs.Agent, "agent-uri", agentURI, "Testkube Pro agent URI [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.URIs.Logs, "logs-uri", "", "Testkube Pro logs URI [required for centralized mode]")
	cmd.Flags().StringVar(&opts.Master.AgentToken, "agent-token", "", "Testkube Pro agent key [required for centralized mode]")
	neededForLogin := ""
	if isDockerCmd {
		neededForLogin = ". It can be skipped for no login mode"
	}

	cmd.Flags().StringVar(&opts.Master.OrgId, "org-id", "", "Testkube Pro organization id [required for centralized mode]"+neededForLogin)
	cmd.Flags().StringVar(&opts.Master.EnvId, "env-id", "", "Testkube Pro environment id [required for centralized mode]"+neededForLogin)

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

	if cmd.Flag("insecure") != nil && cmd.Flag("insecure").Value.String() == "true" {
		opts.Master.Insecure = true
	}

	if cmd.Flag("api-prefix") != nil && cmd.Flags().Changed("api-prefix") {
		opts.Master.ApiUrlPrefix = cmd.Flag("api-prefix").Value.String()
	}

	if cmd.Flag("ui-prefix") != nil && cmd.Flags().Changed("ui-prefix") {
		opts.Master.UiUrlPrefix = cmd.Flag("ui-prefix").Value.String()
	}

	if cmd.Flag("logs-prefix") != nil && cmd.Flags().Changed("logs-prefix") {
		opts.Master.LogsUrlPrefix = cmd.Flag("logs-prefix").Value.String()
	}

	if cmd.Flags().Changed("custom-auth") {
		opts.Master.CustomAuth = cmd.Flag("custom-auth").Value.String() == "true"
	}

	uris := NewMasterUris(opts.Master.ApiUrlPrefix,
		opts.Master.UiUrlPrefix,
		opts.Master.AgentUrlPrefix,
		opts.Master.LogsUrlPrefix,
		opts.Master.URIs.Agent,
		opts.Master.URIs.Logs,
		opts.Master.RootDomain,
		opts.Master.Insecure)

	// override whole URIs usually composed from prefix - host parts
	if cmd.Flag("agent-uri-override") != nil && cmd.Flags().Changed("agent-uri-override") {
		uris.WithAgentURI(cmd.Flag("agent-uri-override").Value.String())
	}

	if cmd.Flag("logs-uri-override") != nil && cmd.Flags().Changed("logs-uri-override") {
		uris.WithLogsURI(cmd.Flag("logs-uri-override").Value.String())
	}

	if cmd.Flag("api-uri-override") != nil && cmd.Flags().Changed("api-uri-override") {
		uris.WithApiURI(cmd.Flag("api-uri-override").Value.String())
	}

	if cmd.Flag("ui-uri-override") != nil && cmd.Flags().Changed("ui-uri-override") {
		uris.WithUiURI(cmd.Flag("ui-uri-override").Value.String())
	}

	if cmd.Flag("auth-uri-override") != nil && cmd.Flags().Changed("auth-uri-override") {
		uris.WithAuthURI(cmd.Flag("auth-uri-override").Value.String())
	}

	opts.Master.URIs = uris

}

// CommaList is a custom flag type for features
type CommaList []string

func (s CommaList) String() string {
	return strings.Join(s, ",")
}
func (s *CommaList) Type() string {
	return "[]string"
}

func (s *CommaList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

// Enabled returns true if the feature is enabled, defaults to all
func (s *CommaList) Enabled(value string) bool {
	if len(*s) == 0 {
		return true
	}
	for _, f := range *s {
		if f == value {
			return true
		}
	}
	return false
}
