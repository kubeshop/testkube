package pro

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
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

	cmd := &cobra.Command{
		Use:     "login [apiUrl]",
		Aliases: []string{"l"},
		Short:   "Login to Testkube Pro",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if len(args) > 0 {
				// Get the URL
				if !strings.Contains(args[0], "://") {
					args[0] = fmt.Sprintf("https://%s", args[0])
				}
				u, err := url.Parse(args[0])
				ui.ExitOnError("invalid instance url", err)
				u.Path, err = url.JoinPath(u.Path, "public-info")
				ui.ExitOnError("invalid instance url", err)

				// Call the Control Plane
				req, err := http.Get(u.String())
				if err != nil && strings.Contains(err.Error(), "response to HTTPS client") {
					// Automatically handle http/https discovery
					u.Scheme = "http"
					req, err = http.Get(u.String())
				}
				ui.ExitOnError("requesting control plane info", err)

				v, err := io.ReadAll(req.Body)
				ui.ExitOnError("reading control plane info", err)
				var result CloudConfig
				err = json.Unmarshal(v, &result)

				// Try with "api." prefix if direct failed
				if err != nil {
					u.Host = fmt.Sprintf("api.%s", u.Host)
					req, err = http.Get(u.String())
					ui.ExitOnError("requesting control plane info", err)
					v, err = io.ReadAll(req.Body)
					ui.ExitOnError("reading control plane info", err)
					err = json.Unmarshal(v, &result)
				}
				ui.ExitOnError("reading control plane info", err)

				if req.StatusCode != http.StatusOK {
					ui.Fail(fmt.Errorf("unexpected error while getting control plane info: %d: %s", req.StatusCode, string(v)))
				}
				if result.APIURL == "" && result.RootDomain == "" {
					ui.Fail(errors.New("unexpected error while getting control plane info missing URLs"))
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

			token, refreshToken, err := common.LoginUser(opts.Master.URIs.Auth, opts.Master.CustomAuth, opts.Master.CallbackPort)
			ui.ExitOnError("getting token", err)

			orgID := opts.Master.OrgId
			envID := opts.Master.EnvId

			if orgID == "" {
				orgID, _, err = common.UiGetOrganizationId(opts.Master.URIs.Api, token)
				ui.ExitOnError("getting organization", err)
			}
			if envID == "" {
				envID, _, err = common.UiGetEnvironmentID(opts.Master.URIs.Api, token, orgID)
				ui.ExitOnError("getting environment", err)
			}

			err = common.PopulateLoginDataToContext(orgID, envID, token, refreshToken, "", opts, cfg)
			ui.ExitOnError("saving config file", err)

			ui.Success("Your config was updated with new values")
			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	common.PopulateMasterFlags(cmd, &opts, false)

	return cmd
}
