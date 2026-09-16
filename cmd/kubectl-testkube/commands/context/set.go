package context

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

// orgEnvLookupHint turns a failed organization or environment lookup into
// advice. The status code is the only thing separating an id that does not
// exist from a credential that was refused - the response body is often empty -
// so without it the hint can do no better than list both causes.
func orgEnvLookupHint(err error) string {
	var statusErr *cloudclient.StatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "The Control Plane refused the stored credential: log in again with 'testkube login', " +
				"or pass a current key with the '--api-key' flag"
		case http.StatusNotFound:
			return "Check that '--org-id' and '--env-id' name an organization and an environment " +
				"that exist on this Control Plane"
		}
	}

	return "Check that '--org-id' and '--env-id' are set and correct for this Control Plane, " +
		"or log in again with 'testkube login' if the stored API key or login token stopped working"
}

func NewSetContextCmd() *cobra.Command {
	var (
		org, env, apiKey    string
		kubeconfig          bool
		namespace           string
		opts                common.HelmOptions
		dockerContainerName string
	)

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set context data for Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}
			common.SyncSkipTLSFromFlags(cmd, &cfg)
			common.ProcessMasterFlags(cmd, &opts, &cfg)

			if cmd.Flags().Changed("org") {
				opts.Master.OrgId = org
			}

			if cmd.Flags().Changed("env") {
				opts.Master.EnvId = env
			}

			if kubeconfig {
				cfg.ContextType = config.ContextTypeKubeconfig
			} else {
				cfg.ContextType = config.ContextTypeCloud
			}

			switch cfg.ContextType {
			case config.ContextTypeCloud:
				// --root-domain is registered with a default, so it is never empty
				// and testing the value let every flagless invocation through.
				rootDomainSet := cmd.Flags().Changed("root-domain") ||
					cmd.Flags().Changed("pro-root-domain") || cmd.Flags().Changed("cloud-root-domain")

				if opts.Master.OrgId == "" && opts.Master.EnvId == "" && opts.Master.OrgName == "" &&
					opts.Master.EnvName == "" && apiKey == "" && !rootDomainSet {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrInvalidRuntimeParameter,
						"No context value provided",
						"Provide at least one of the following flags: --org-id, --org-name, --env-id, --env-name, --api-key, --root-domain",
						errors.New("nothing to set on the Testkube Pro context"),
					))
				}

				// Everything below — the name lookups, the display names fetched
				// afterwards, and the URI persisted by PopulateCloudConfig — has
				// to point at the Control Plane the user is actually on. Without
				// an explicit flag ProcessMasterFlags composes the SaaS host from
				// prefixes, which for a custom Control Plane is both the wrong
				// place to look and the wrong thing to save.
				opts.Master.URIs.Api = common.ControlPlaneAPIURI(cmd, opts.Master.URIs.Api, &cfg)

				// Every other command refreshes an expired login token inside
				// GetClient. This one builds its cloud clients directly, so a
				// token that would have been renewed silently anywhere else used
				// to fail here and send the user off to log in again. A failure
				// is not fatal: the lookups below report it in context.
				if apiKey == "" {
					if err := common.RefreshContextToken(&cfg, cfg.SkipTLS || cfg.CloudContext.SkipTLS); err != nil {
						ui.Debug("could not refresh the stored login token", err.Error())
					}
				}

				// Names have to become ids before anything is written, and the
				// lookup needs a token: the one being set if there is one,
				// otherwise whatever the context already holds.
				if opts.Master.OrgName != "" || opts.Master.EnvName != "" {
					lookupToken := apiKey
					if lookupToken == "" {
						lookupToken = cfg.CloudContext.ApiKey
					}
					if lookupToken == "" {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrInvalidRuntimeParameter,
							"Missing credentials for the name lookup",
							"Pass an API key with the '--api-key' flag or run 'testkube pro login' first, or select the organization and environment by id with '--org-id' and '--env-id'",
							errors.New("resolving --org-name or --env-name requires an API key or a login token"),
						))
					}

					lookupSkipTLS := cfg.SkipTLS || cfg.CloudContext.SkipTLS

					// The organization has to resolve first: the environment
					// lookup is scoped to it.
					if err := common.ResolveNamedOrg(opts.Master.URIs.Api, lookupToken, &opts.Master, lookupSkipTLS); err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrOrgResolutionFailed,
							"Error resolving Testkube Pro organization",
							"Check does the organization name exist and is your API key or login token allowed to see it, or select it by id with the '--org-id' flag",
							err,
						))
					}

					if err := common.ResolveNamedEnv(opts.Master.URIs.Api, lookupToken, &opts.Master,
						cfg.CloudContext.OrganizationId, lookupSkipTLS); err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrEnvResolutionFailed,
							"Error resolving Testkube Pro environment",
							"Check does the environment name exist in the selected organization, or select it by id with the '--env-id' flag",
							err,
						))
					}
				}

				var dcName *string
				if cmd.Flags().Changed("docker-container") {
					dcName = &dockerContainerName
				}

				cfg = common.PopulateCloudConfig(cfg, apiKey, dcName, &opts)

				if cfg.CloudContext.ApiKey != "" {
					var err error
					cfg, err = common.PopulateOrgAndEnvNames(cfg, opts.Master.OrgId, opts.Master.EnvId, opts.Master.URIs.Api)
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrOrgEnvNamesFetchFailed,
							"Could not look up your organization or environment",
							orgEnvLookupHint(err),
							err,
						))
					}
				} else {
					ui.Warn("No API key provided, you need to login to Testkube Cloud")
				}

			case config.ContextTypeKubeconfig:
				// kubeconfig special use cases

			default:
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Unknown context type",
					"Use the '--kubeconfig' flag for a kubeconfig based context, or set the Testkube Pro context values",
					fmt.Errorf("unknown context type: %s", cfg.ContextType),
				))
			}

			if namespace != "" {
				cfg.Namespace = namespace
			}

			if cfg.ContextType == config.ContextTypeCloud {
				cfg.CloudContext.SkipTLS = cfg.SkipTLS
			}

			if err = config.Save(cfg); err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigSaveFailed,
					"Error saving testkube config file",
					common.ConfigFileHint,
					err,
				))
			}

			if err = validator.ValidateCloudContext(cfg); err != nil {
				common.UiCloudContextValidationError(err)
			}

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)

		},
	}

	cmd.Flags().BoolVarP(&kubeconfig, "kubeconfig", "", false, "reset context mode for CLI to default kubeconfig based")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Testkube Pro Organization ID")
	cmd.Flags().MarkDeprecated("org", "use --org-id instead")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Testkube Pro Environment ID")
	cmd.Flags().MarkDeprecated("env", "use --env-id instead")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Testkube namespace to use for CLI commands")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API Key for Testkube Pro")

	// allow to override default values of all URIs
	cmd.Flags().StringVar(&dockerContainerName, "docker-container", "testkube-agent", "Docker container name for Testkube Docker Agent")

	common.PopulateMasterFlags(cmd, &opts, false)

	// The deprecated aliases carry ids, so they conflict with the name flags in
	// the same way --org-id and --env-id do.
	cmd.MarkFlagsMutuallyExclusive("org", "org-name")
	cmd.MarkFlagsMutuallyExclusive("env", "env-name")

	return cmd
}
