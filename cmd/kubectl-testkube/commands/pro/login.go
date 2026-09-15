package pro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	tkhttp "github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/spf13/cobra"
)

type CloudConfig struct {
	AuthURL    string `json:"authUrl"`
	APIURL     string `json:"apiUrl"`
	UIURL      string `json:"uiUrl"`
	AgentURL   string `json:"agentUrl"`
	RootDomain string `json:"rootDomain"`
}

func NewLoginCmd() *cobra.Command {
	var opts common.HelmOptions
	var email string
	var emailLink string

	cmd := &cobra.Command{
		Use:     "login [apiUrl]",
		Aliases: []string{"l"},
		Short:   "Login to Testkube Pro",
		Args:    cobra.MaximumNArgs(1),
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
			skipTLS := common.SyncSkipTLSFromFlags(cmd, &cfg)
			discoveryClient := tkhttp.NewClient(skipTLS)

			if len(args) > 0 {
				// Get the URL
				if !strings.Contains(args[0], "://") {
					args[0] = fmt.Sprintf("https://%s", args[0])
				}
				// A bad address and an unreachable Control Plane need different
				// things from the user, so they are reported apart.
				exitOnInvalidURL := func(err error) {
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrInvalidRuntimeParameter,
							"Invalid Control Plane URL",
							"Pass the Control Plane address as a URL, for example `testkube pro login https://cp.example.com`",
							err,
						))
					}
				}
				u, err := url.Parse(args[0])
				exitOnInvalidURL(err)
				u.Path, err = url.JoinPath(u.Path, "public-info")
				exitOnInvalidURL(err)

				// u is rewritten as the http/https and "api." fallbacks are
				// tried, so the hint names whichever address failed last.
				exitOnDiscoveryError := func(err error) {
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrControlPlaneDiscoveryFailed,
							"Error reading the Control Plane information",
							"Check does "+u.String()+" point at a Testkube Control Plane and is it reachable from here",
							err,
						))
					}
				}

				// Call the Control Plane
				httpReq, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
				exitOnInvalidURL(reqErr)
				req, err := discoveryClient.Do(httpReq)
				if err != nil && strings.Contains(err.Error(), "response to HTTPS client") {
					// Automatically handle http/https discovery
					u.Scheme = "http"
					httpReq, reqErr = http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
					exitOnInvalidURL(reqErr)
					req, err = discoveryClient.Do(httpReq)
				}
				exitOnDiscoveryError(err)

				v, err := io.ReadAll(req.Body)
				exitOnDiscoveryError(err)
				_ = req.Body.Close()
				var result CloudConfig
				err = json.Unmarshal(v, &result)

				// Try with "api." prefix if direct failed
				if err != nil {
					u.Host = fmt.Sprintf("api.%s", u.Host)
					httpReq, reqErr = http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
					exitOnInvalidURL(reqErr)
					req, err = discoveryClient.Do(httpReq)
					exitOnDiscoveryError(err)
					v, err = io.ReadAll(req.Body)
					exitOnDiscoveryError(err)
					_ = req.Body.Close()
					err = json.Unmarshal(v, &result)
				}
				exitOnDiscoveryError(err)

				if req.StatusCode != http.StatusOK {
					exitOnDiscoveryError(fmt.Errorf("the Control Plane answered %d: %s", req.StatusCode, string(v)))
				}
				if result.APIURL == "" && result.RootDomain == "" {
					exitOnDiscoveryError(errors.New("the response carries neither an API URL nor a root domain"))
				}

				// Try to fill the data
				if result.RootDomain != "" {
					cmd.Flags().Set("root-domain", result.RootDomain)
					if result.RootDomain != "testkube.io" {
						cmd.Flags().Set("custom-auth", "true")
					}
				} else {
					if !cmd.Flags().Changed("auth-uri-override") && result.AuthURL != "" {
						cmd.Flags().Set("auth-uri-override", result.AuthURL)
					}
					if !cmd.Flags().Changed("api-uri-override") && result.APIURL != "" {
						cmd.Flags().Set("api-uri-override", result.APIURL)
					}
					if !cmd.Flags().Changed("ui-uri-override") && result.UIURL != "" {
						cmd.Flags().Set("ui-uri-override", result.UIURL)
					}
					if !cmd.Flags().Changed("agent-uri-override") && result.AgentURL != "" {
						if !strings.Contains(result.AgentURL, "://") {
							result.AgentURL = fmt.Sprintf("%s://%s", u.Scheme, result.AgentURL)
						}
						cmd.Flags().Set("agent-uri-override", result.AgentURL)
					}

					if !cmd.Flags().Changed("callback-port") {
						callbackPort, _ := cmd.Flags().GetInt("callback-port")
						reservedURLs := []string{
							fmt.Sprintf("http://localhost:%d", callbackPort),
							fmt.Sprintf("http://127.0.0.1:%d", callbackPort),
							fmt.Sprintf("https://localhost:%d", callbackPort),
							fmt.Sprintf("https://127.0.0.1:%d", callbackPort),
						}
						conflicting := false
						for _, url := range reservedURLs {
							if result.APIURL == url || strings.HasPrefix(result.APIURL, url+"/") {
								conflicting = true
								break
							}
						}
						if conflicting {
							cmd.Flags().Set("callback-port", fmt.Sprintf("%d", config.AlternativeCallbackPort))
						}
					}
					cmd.Flags().Set("custom-auth", "true")
				}
			}

			common.ProcessMasterFlags(cmd, &opts, &cfg)

			var token, refreshToken string
			tokenType := config.TokenTypeOIDC

			switch {
			case emailLink != "":
				// Email magic-link flow (via --email-link flag)
				token, refreshToken, err = common.LoginUserEmailLink(opts.Master.URIs.Api, emailLink, opts.Master.CallbackPort, skipTLS)
				tokenType = config.TokenTypeEmailLink
			case email != "":
				// SSO authentication flow
				token, refreshToken, err = common.LoginUserSSO(opts.Master.URIs.Api, opts.Master.URIs.Auth, email, opts.Master.CallbackPort, skipTLS)
			default:
				// Interactive selector: GitHub / GitLab / Google / Email magic-link
				tokenType, token, refreshToken, err = common.LoginUser(opts.Master.URIs.Auth, opts.Master.URIs.Api, opts.Master.CustomAuth, opts.Master.CallbackPort, skipTLS)
			}
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrLoginFailed,
					"Error logging in to Testkube Pro",
					"Check is the browser able to reach the Testkube Pro auth endpoint, and that the email or link you passed is still valid",
					err,
				))
			}

			// The organization has to resolve first: the environment lookup is
			// scoped to it.
			orgID, err := common.ResolveOrgOrPrompt(opts.Master.URIs.Api, token, opts.Master, skipTLS)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrOrgResolutionFailed,
					"Error resolving Testkube Pro organization",
					"Check does your account have access to the organization, or select it explicitly with the '--org-id' flag",
					err,
				))
			}

			envID, err := common.ResolveEnvOrPrompt(opts.Master.URIs.Api, token, orgID, opts.Master, skipTLS)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrEnvResolutionFailed,
					"Error resolving Testkube Pro environment",
					"Check does the environment exist in the selected organization, or select it explicitly with the '--env-id' flag",
					err,
				))
			}

			if err = common.PopulateLoginDataToContext(orgID, envID, tokenType, token, refreshToken, "", opts, cfg); err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrContextSaveFailed,
					"Error saving the Testkube Pro context",
					common.ConfigFileHint,
					err,
				))
			}

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	common.PopulateMasterFlags(cmd, &opts, false)
	cmd.Flags().StringVar(&email, "email", "", "email address for SSO authentication")
	cmd.Flags().StringVar(&emailLink, "email-link", "", "email address for magic-link authentication")
	cmd.MarkFlagsMutuallyExclusive("email", "email-link")

	return cmd
}
