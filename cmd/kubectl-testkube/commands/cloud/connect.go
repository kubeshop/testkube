package cloud

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	ui.ExitOnError("getting server info", err)

	var apiContext string
	if actx, ok := contextDescription[info.Context]; ok {
		apiContext = actx
	}

	h1 := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightMagenta)).WithMargin(0)
	h2 := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.Bold)).WithTextStyle(pterm.NewStyle(pterm.FgLightGreen)).WithMargin(0)
	// text := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgDarkGray)).
	// 	WithTextStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgGray)).WithMargin(0)

	text := pterm.DefaultParagraph.WithMaxWidth(100)

	h1.Println("Connect your cloud environment:")
	text.Println("You can learn more about connecting your Testkube instance to the Cloud here:\n" + docsUrl)
	h2.Println("You can safely switch between connecting Cloud and disconnecting without losing your data.")

	cfg, err := config.Load()
	if err != nil {
		pterm.Error.Printfln("Failed to load config file: %s", err.Error())
		return
	}

	var clusterContext string
	if cfg.ContextType == config.ContextTypeKubeconfig {
		clusterContext, err = common.GetCurrentKubernetesContext()
		if err != nil {
			pterm.Error.Printfln("Failed to get current kubernetes context: %s", err.Error())
			return
		}
	}

	// TODO: implement context info
	h1.Println("Current status of your Testkube instance")
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
	if connectOpts.AgentToken == "" {
		h1.Println("Login")
		text.Println("Your browser should open automatically. If not, please open this link in your browser:")
		text.Println(authUrlWithRedirect)
		text.Println("(just login and get back to your terminal)")
		text.Println("")

		if ok := ui.Confirm("Continue"); !ok {
			return
		}

		// open browser with login page and redirect to localhost:7899
		open.Run(authUrlWithRedirect)

		token, err := uiGetToken()
		ui.ExitOnError("getting token", err)

		orgId, orgName, err := uiGetOrganizationId(token)
		ui.ExitOnError("getting token", err)

		envName, err := uiGetEnvName()
		ui.ExitOnError("getting environment name", err)

		envClient := cloudclient.NewEnvironmentsClient(token)
		env, err := envClient.Create(cloudclient.Environment{Name: envName, Owner: orgId})
		if err != nil {
			pterm.Error.Println("Failed to create environment", err.Error())
			return
		}

		summary = append(summary, []string{"Testkube will be connected to cloud org/env"})
		summary = append(summary, []string{"Organization Id", orgId})
		summary = append(summary, []string{"Organization name", orgName})
		summary = append(summary, []string{"Environment Id", env.Id})
		summary = append(summary, []string{"Environment name", env.Name})
		summary = append(summary, []string{ui.Separator, ""})

		connectOpts.AgentToken = env.AgentToken
	}

	// scale down not remove
	connectOpts.NoDashboard = false
	connectOpts.NoMinio = false
	connectOpts.NoMongo = false
	connectOpts.DashboardReplicas = 0
	connectOpts.MinioReplicas = 0
	connectOpts.MongoReplicas = 0

	summary = append(summary, []string{"Testkube support services not needed anymore"})
	summary = append(summary, []string{"MinIO    ", "Stopped and scaled down, (not deleted)"})
	summary = append(summary, []string{"MongoDB  ", "Stopped and scaled down, (not deleted)"})
	summary = append(summary, []string{"Dashboard", "Stopped and scaled down, (not deleted)"})

	ui.NL(2)

	h1.Println("Summary of your setup after connecting to Testkube Cloud")
	ui.Properties(summary)
	// TODO expected statsus

	//         Connect your cloud environment:        You can learn more about connecting your Testkube instance to the Cloud here:         https://docs.testkube.io/etc...

	//         You can safely switch between connecting Cloud and disconnecting without losing your data.STATUS  Current status of your Testkube instance

	//         Context:   On premise (local OSS context)
	//         Cluster:   Cluster name        Namespace: Testkube
	// LOGIN   Login        Please open the following link in your browser and log in:         https://cloud.testkube.io/login?redirect_uri=....
	// ORG     Organisation        Select the organisation you want to connect this instance to        ● My-Org-1        ○ Another Org
	// ENV     Environment
	//         Create an environment on Cloud for this instance        Tell us the name of your new environment        > my-env-1        ⏳ Creating environment         ✅ Environment created        ❌ Environment creation failed – your plan does only support 2 environments
	//         SANITY  Summary of your setup after connecting:         Context:   Cloud
	//         Cluster:   Cluster name        Namespace: Testkube        Org. name: My-Org-1        Env. name: my-env-1        Minio:     stopped and scaled down (not deleted)        MongoDB:   stopped and scaled down (not deleted)        Dashboard: stopped and scaled down (not deleted)        Shall we start connecting?

	// 		Remember: All your historical data and artifacts will be safe in case you want to rollback

	// 		● Yes        ○ No        CONNECT        ✅ Updating context to Cloud         ✅ Stopping Minio        ✅ Stopping Dashboard UI        ⏳ Stopping MongoDB

	//         ✅ Connection finished successfully        🎉 Happy testing!

	//         You can now login to Testkube Cloud and validate your connection:        https://cloud.testkube.io/org/123/env/123

	//         In case you want to roll back you can simply run the following command in your CLI:
	//         $ testkube cloud disconnect environment

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
		pterm.Error.Println("Failed to get organizations", err.Error())
		return "", "", fmt.Errorf("failed to get organizations: %s", err.Error())
	}

	orgNames := getNames(orgs)
	orgName, _ := pterm.DefaultInteractiveSelect.
		WithOptions(orgNames).
		Show()

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
