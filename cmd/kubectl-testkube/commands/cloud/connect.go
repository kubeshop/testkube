package cloud

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/skratchdot/open-golang/open"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var connectOpts = common.HelmUpgradeOrInstalTestkubeOptions{DryRun: true}

func NewConnectCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"c"},
		Short:   "Testkube Cloud connect ",
		Run:     cloudConnect,
	}

	common.PopulateUpgradeInstallFlags(cmd, &connectOpts)

	return cmd
}

const (
	listenAddr      = "127.0.0.1:8090"
	redirectUrl     = "http://" + listenAddr + "/callback"
	authUrl         = "https://cloud.testkube.io/?redirectUri="
	docsUrl         = "https://docs.testkube.io/testkube-cloud/intro"
	tokenQueryParam = "token"
)

var contextDescription = map[string]string{
	"":      "Unknown context, try updating your testkube cluster installation",
	"oss":   "Open Source Testkube",
	"cloud": "Testkube in Cloud mode",
}

func cloudConnect(cmd *cobra.Command, args []string) {
	client, _ := common.GetClient(cmd)

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

	authUrlWithRedirect := authUrl + url.QueryEscape(redirectUrl)

	summary := [][]string{
		{"Testkube mode"},
		{"Context", contextDescription["cloud"]},
		{"Kubectl context", clusterContext},
		{"Namespace", cfg.Namespace},
		{ui.Separator, ""},
	}

	// if no agent is passed create new environment and get its token
	if connectOpts.CloudAgentToken == "" {
		ui.H1("Login")
		ui.Paragraph("Your browser should open automatically. If not, please open this link in your browser:")
		ui.Paragraph(authUrlWithRedirect)
		ui.Paragraph("(just login and get back to your terminal)")
		ui.Paragraph("")

		if ok := ui.Confirm("Continue"); !ok {
			return
		}

		// open browser with login page and redirect to localhost:7899
		open.Run(authUrlWithRedirect)

		token, err := uiGetToken()
		ui.ExitOnError("getting token", err)

		orgId, orgName, err := uiGetOrganizationId(token)
		ui.ExitOnError("getting organization", err)

		envName, err := uiGetEnvName()
		ui.ExitOnError("getting environment name", err)

		envClient := cloudclient.NewEnvironmentsClient(token)
		env, err := envClient.Create(cloudclient.Environment{Name: envName, Owner: orgId})
		ui.ExitOnError("creating environment", err)

		connectOpts.CloudOrgId = orgId
		connectOpts.CloudEnvId = env.Id

		summary = append(summary, []string{"Testkube will be connected to cloud org/env"})
		summary = append(summary, []string{"Organization Id", connectOpts.CloudOrgId})
		summary = append(summary, []string{"Organization name", orgName})
		summary = append(summary, []string{"Environment Id", connectOpts.CloudEnvId})
		summary = append(summary, []string{"Environment name", env.Name})
		summary = append(summary, []string{ui.Separator, ""})

		connectOpts.CloudAgentToken = env.AgentToken
	}

	// validate if user created env - or was passed from flags
	if connectOpts.CloudEnvId == "" {
		ui.Failf("You need pass valid environment id to connect to cloud")
	}
	if connectOpts.CloudOrgId == "" {
		ui.Failf("You need pass valid organization id to connect to cloud")
	}

	// scale down not remove
	connectOpts.NoDashboard = false
	connectOpts.NoMinio = false
	connectOpts.NoMongo = false
	connectOpts.DashboardReplicas = 0
	connectOpts.MinioReplicas = 0
	connectOpts.MongoReplicas = 0

	// update summary
	summary = append(summary, []string{"Testkube support services not needed anymore"})
	summary = append(summary, []string{"MinIO    ", "Stopped and scaled down, (not deleted)"})
	summary = append(summary, []string{"MongoDB  ", "Stopped and scaled down, (not deleted)"})
	summary = append(summary, []string{"Dashboard", "Stopped and scaled down, (not deleted)"})

	ui.NL(2)

	ui.H1("Summary of your setup after connecting to Testkube Cloud")
	ui.Properties(summary)

	ui.NL()
	ui.Warn("Remember: All your historical data and artifacts will be safe in case you want to rollback")
	ui.NL()

	if ok := ui.Confirm("Proceed with connecting Testkube Cloud?"); !ok {
		return
	}

	spinner := ui.NewSpinner("Connecting Testkube Cloud")

	err = common.HelmUpgradeOrInstallTestkubeCloud(connectOpts, cfg)
	ui.ExitOnError("Installing Testkube Cloud", err)
	spinner.Success()

	ui.NL()

	// let's scale down deployment of mongo
	if connectOpts.MongoReplicas == 0 {
		spinner = ui.NewSpinner("Scaling down MongoDB")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-mongodb", connectOpts.MongoReplicas)
		spinner.Success()
	}
	if connectOpts.MinioReplicas == 0 {
		spinner = ui.NewSpinner("Scaling down MinIO")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-minio-testkube", connectOpts.MinioReplicas)
		spinner.Success()
	}
	if connectOpts.DashboardReplicas == 0 {
		spinner = ui.NewSpinner("Scaling down Dashbaord")
		common.KubectlScaleDeployment(connectOpts.Namespace, "testkube-dashboard", connectOpts.DashboardReplicas)
		spinner.Success()
	}

	ui.H2("Testkube Cloud is connected to your Testkube instance, saving local configuration")
	err = common.PopulateAgentDataToContext(connectOpts, cfg)
	ui.ExitOnError("Populating agent data to context", err)

	ui.NL(2)

	ui.ShellCommand("In case you want to roll back you can simply run the following command in your CLI:", "testkube cloud disconnect")

	ui.Success("You can now login to Testkube Cloud and validate your connection:")
	ui.Info("https://cloud.testkube.io/organization/%s/environment/%s", connectOpts.CloudOrgId, connectOpts.CloudEnvId)
}

// getToken returns chan to wait for token from http server
// client need to call / endpoint with token query param
func getToken() (chan string, error) {
	tokenChan := make(chan string)

	go func() {
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			tokenChan <- r.URL.Query().Get(tokenQueryParam)
		})

		log.Fatal(http.ListenAndServe(listenAddr, nil))
	}()

	return tokenChan, nil
}

type Organization struct {
	Id   string
	Name string
}

func getOrganizations(token string) ([]cloudclient.Organization, error) {
	c := cloudclient.NewOrganizationsClient(token)
	return c.List()
}

func getNames(orgs []cloudclient.Organization) []string {
	var names []string
	for _, org := range orgs {
		names = append(names, org.Name)
	}
	return names
}

func findId(orgs []cloudclient.Organization, name string) string {
	for _, org := range orgs {
		if org.Name == name {
			return org.Id
		}
	}
	return ""
}

func uiGetOrganizationId(token string) (string, string, error) {
	// Choose organization from orgs available
	orgs, err := getOrganizations(token)
	if err != nil {
		return "", "", fmt.Errorf("failed to get organizations: %s", err.Error())
	}

	orgNames := getNames(orgs)
	orgName := ui.Select("Choose organization", orgNames)
	orgId := findId(orgs, orgName)

	return orgId, orgName, nil
}

func uiGetToken() (string, error) {
	// wait for token received to browser
	s := ui.NewSpinner("waiting for auth token")
	tokenChan, err := getToken()
	if err != nil {
		s.Fail("Failed to get auth token", err.Error())
		return "", fmt.Errorf("failed to get auth token: %s", err.Error())
	}

	var token string
	select {
	case token = <-tokenChan:
		s.Success()
	case <-time.After(5 * time.Minute):
		s.Fail("Timeout waiting for auth token")
		return "", fmt.Errorf("timeout waiting for auth token")
	}
	ui.NL()

	return token, nil
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
