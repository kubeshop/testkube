package cloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/skratchdot/open-golang/open"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/cloudlogin"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	listenAddr      = "127.0.0.1:8090"
	docsUrl         = "https://docs.testkube.io/testkube-cloud/intro"
	tokenQueryParam = "token"
)

func NewConnectCmd() *cobra.Command {
	var opts = common.HelmOptions{}

	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"c"},
		Short:   "Testkube Cloud connect ",
		Run: func(cmd *cobra.Command, args []string) {

			// create new cloud uris
			opts.CloudUris = common.NewCloudUris(opts.CloudRootDomain)

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			info, err := client.GetServerInfo()
			firstInstall := err != nil && strings.Contains(err.Error(), "not found")
			if err != nil && !firstInstall {
				ui.Failf("Can't get testkube cluster information: %s", err.Error())
			}

			var apiContext string
			if actx, ok := contextDescription[info.Context]; ok {
				apiContext = actx
			}

			ui.H1("Connect your cloud environment:")
			ui.Paragraph("You can learn more about connecting your Testkube instance to the Cloud here:\n" + docsUrl)
			ui.H2("You can safely switch between connecting Cloud and disconnecting without losing your data.")

			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)

			var clusterContext string
			if cfg.ContextType == config.ContextTypeKubeconfig {
				clusterContext, err = common.GetCurrentKubernetesContext()
				ui.ExitOnError("getting current kubernetes context", err)
			}

			// TODO: implement context info
			ui.H1("Current status of your Testkube instance")
			ui.Properties([][]string{
				{"Context", apiContext},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
			})

			newStatus := [][]string{
				{"Testkube mode"},
				{"Context", contextDescription["cloud"]},
				{"Kubectl context", clusterContext},
				{"Namespace", cfg.Namespace},
				{ui.Separator, ""},
			}

			var (
				token        string
				refreshToken string
			)
			// if no agent is passed create new environment and get its token
			if opts.CloudAgentToken == "" && opts.CloudOrgId == "" && opts.CloudEnvId == "" {
				token, refreshToken, err = LoginUser(opts)
				ui.ExitOnError("login", err)

				orgId, orgName, err := uiGetOrganizationId(opts.CloudRootDomain, token)
				ui.ExitOnError("getting organization", err)

				envName, err := uiGetEnvName()
				ui.ExitOnError("getting environment name", err)

				envClient := cloudclient.NewEnvironmentsClient(opts.CloudRootDomain, token, orgId)
				env, err := envClient.Create(cloudclient.Environment{Name: envName, Owner: orgId})
				ui.ExitOnError("creating environment", err)

				opts.CloudOrgId = orgId
				opts.CloudEnvId = env.Id
				opts.CloudAgentToken = env.AgentToken

				newStatus = append(
					newStatus,
					[][]string{
						{"Testkube will be connected to cloud org/env"},
						{"Organization Id", opts.CloudOrgId},
						{"Organization name", orgName},
						{"Environment Id", opts.CloudEnvId},
						{"Environment name", env.Name},
						{ui.Separator, ""},
					}...)
			}

			// validate if user created env - or was passed from flags
			if opts.CloudEnvId == "" {
				ui.Failf("You need pass valid environment id to connect to cloud")
			}
			if opts.CloudOrgId == "" {
				ui.Failf("You need pass valid organization id to connect to cloud")
			}

			// update summary
			newStatus = append(newStatus, []string{"Testkube support services not needed anymore"})
			newStatus = append(newStatus, []string{"MinIO    ", "Stopped and scaled down, (not deleted)"})
			newStatus = append(newStatus, []string{"MongoDB  ", "Stopped and scaled down, (not deleted)"})
			newStatus = append(newStatus, []string{"Dashboard", "Stopped and scaled down, (not deleted)"})

			ui.NL(2)

			ui.H1("Summary of your setup after connecting to Testkube Cloud")
			ui.Properties(newStatus)

			ui.NL()
			ui.Warn("Remember: All your historical data and artifacts will be safe in case you want to rollback. OSS and cloud executions will be separated.")
			ui.NL()

			if ok := ui.Confirm("Proceed with connecting Testkube Cloud?"); !ok {
				return
			}

			spinner := ui.NewSpinner("Connecting Testkube Cloud")
			err = common.HelmUpgradeOrInstallTestkubeCloud(opts, cfg, true)
			ui.ExitOnError("Installing Testkube Cloud", err)
			spinner.Success()

			ui.NL()

			// let's scale down deployment of mongo
			if opts.MongoReplicas == 0 {
				spinner = ui.NewSpinner("Scaling down MongoDB")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-mongodb", opts.MongoReplicas)
				spinner.Success()
			}
			if opts.MinioReplicas == 0 {
				spinner = ui.NewSpinner("Scaling down MinIO")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-minio-testkube", opts.MinioReplicas)
				spinner.Success()
			}
			if opts.DashboardReplicas == 0 {
				spinner = ui.NewSpinner("Scaling down Dashbaord")
				common.KubectlScaleDeployment(opts.Namespace, "testkube-dashboard", opts.DashboardReplicas)
				spinner.Success()
			}

			ui.H2("Testkube Cloud is connected to your Testkube instance, saving local configuration")

			ui.H2("Saving testkube cli cloud context")
			if token == "" && !common.IsUserLoggedIn(cfg, opts) {
				token, refreshToken, err = LoginUser(opts)
				ui.ExitOnError("user login", err)
			}
			err = common.PopulateLoginDataToContext(opts.CloudOrgId, opts.CloudEnvId, token, refreshToken, opts, cfg)

			ui.ExitOnError("Setting cloud environment context", err)

			ui.NL(2)

			ui.ShellCommand("In case you want to roll back you can simply run the following command in your CLI:", "testkube cloud disconnect")

			ui.Success("You can now login to Testkube Cloud and validate your connection:")
			ui.NL()
			ui.Link("https://cloud." + opts.CloudRootDomain + "/organization/" + opts.CloudOrgId + "/environment/" + opts.CloudEnvId + "/dashboard/tests")

			ui.NL(2)
		},
	}

	cmd.Flags().StringVar(&opts.Chart, "chart", "kubeshop/testkube", "chart name (usually you don't need to change it)")
	cmd.Flags().StringVar(&opts.Name, "name", "testkube", "installation name (usually you don't need to change it)")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().StringVar(&opts.Values, "values", "", "path to Helm values file")

	cmd.Flags().StringVar(&opts.CloudAgentToken, "agent-token", "", "Testkube Cloud agent key [required for cloud mode]")
	cmd.Flags().StringVar(&opts.CloudOrgId, "org-id", "", "Testkube Cloud organization id [required for cloud mode]")
	cmd.Flags().StringVar(&opts.CloudEnvId, "env-id", "", "Testkube Cloud environment id [required for cloud mode]")

	cmd.Flags().StringVar(&opts.CloudRootDomain, "cloud-root-domain", "testkube.io", "defaults to testkube.io, usually don't need to be changed [required for cloud mode]")

	cmd.Flags().BoolVar(&opts.NoMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&opts.NoDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&opts.NoMongo, "no-mongo", false, "don't install MongoDB")

	cmd.Flags().IntVar(&opts.MinioReplicas, "minio-replicas", 0, "MinIO replicas")
	cmd.Flags().IntVar(&opts.MongoReplicas, "mongo-replicas", 0, "MongoDB replicas")
	cmd.Flags().IntVar(&opts.DashboardReplicas, "dashboard-replicas", 0, "Dashboard replicas")

	cmd.Flags().BoolVar(&opts.NoConfirm, "no-confirm", false, "don't ask for confirmation - unatended installation mode")

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "dry run mode - only print commands that would be executed")

	return cmd
}

var contextDescription = map[string]string{
	"":      "Unknown context, try updating your testkube cluster installation",
	"oss":   "Open Source Testkube",
	"cloud": "Testkube in Cloud mode",
}

func uiGetToken(tokenChan chan cloudlogin.Tokens) (string, string, error) {
	// wait for token received to browser
	s := ui.NewSpinner("waiting for auth token")

	var token cloudlogin.Tokens
	select {
	case token = <-tokenChan:
		s.Success()
	case <-time.After(5 * time.Minute):
		s.Fail("Timeout waiting for auth token")
		return "", "", fmt.Errorf("timeout waiting for auth token")
	}
	ui.NL()

	return token.IDToken, token.RefreshToken, nil
}

func uiGetEnvName() (string, error) {
	for i := 0; i < 3; i++ {
		if envName := ui.TextInput("Tell us the name of your environment"); envName != "" {
			return envName, nil
		}
		pterm.Error.Println("Environment name cannot be empty")
	}

	return "", fmt.Errorf("environment name cannot be empty")
}

func LoginUser(opts common.HelmOptions) (string, string, error) {
	authUrl, tokenChan, err := cloudlogin.CloudLogin(context.Background(), opts.CloudUris.Auth)
	if err != nil {
		return "", "", fmt.Errorf("cloud login: %w", err)
	}
	ui.H1("Login")
	ui.Paragraph("Your browser should open automatically. If not, please open this link in your browser:")
	ui.Link(authUrl)
	ui.Paragraph("(just login and get back to your terminal)")
	ui.Paragraph("")

	if ok := ui.Confirm("Continue"); !ok {
		return "", "", fmt.Errorf("login cancelled")
	}

	// open browser with login page and redirect to localhost
	open.Run(authUrl)

	idToken, refreshToken, err := uiGetToken(tokenChan)
	if err != nil {
		return "", "", fmt.Errorf("getting token")
	}
	return idToken, refreshToken, nil
}
