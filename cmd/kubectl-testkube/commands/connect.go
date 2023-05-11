package commands

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/pterm/pterm"
	"github.com/skratchdot/open-golang/open"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var connectOpts = HelmUpgradeOrInstalTestkubeOptions{DryRun: true}

func NewConnectCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"c"},
		Short:   "Testkube Cloud connect ",
		Run:     cloudConnect,
	}

	PopulateUpgradeInstallFlags(cmd, &connectOpts)

	return cmd
}

const (
	listenAddr      = "127.0.0.1:7899"
	redirectUrl     = "http://" + listenAddr
	authUrl         = "https://cloud.testkube.io/?redirectUri="
	docsUrl         = "https://docs.testkube.io/testkube-cloud/intro"
	tokenQueryParam = "token"
)

func cloudConnect(cmd *cobra.Command, args []string) {

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
		clusterContext, err = GetCurrentKubernetesContext()
		if err != nil {
			pterm.Error.Printfln("Failed to get current kubernetes context: %s", err.Error())
			return
		}
	}

	// TODO: implement context info
	h1.Println("Current status of your Testkube instance")
	ui.InfoGrid(map[string]string{
		"Context":   string(cfg.ContextType),
		"Namespace": cfg.Namespace,
		"Cluster":   clusterContext,
	})

	authUrlWithRedirect := authUrl + url.QueryEscape(redirectUrl)

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

		orgId, err := uiGetOrganizationId(token)
		ui.ExitOnError("getting token", err)

		envName, err := uiGetEnvName()
		ui.ExitOnError("getting environment name", err)

		envClient := cloudclient.NewEnvironmentsClient(token)
		env, err := envClient.Create(cloudclient.Environment{Name: envName, Owner: orgId})
		if err != nil {
			pterm.Error.Println("Failed to create environment", err.Error())
			return
		}
		connectOpts.AgentToken = env.AgentToken
	}

	// scale down not remove
	connectOpts.NoDashboard = false
	connectOpts.NoMinio = false
	connectOpts.NoMongo = false
	connectOpts.DashboardReplicas = 0
	connectOpts.MinioReplicas = 0
	connectOpts.MongoReplicas = 0

	ui.NL(2)

	// TODO expected statsus
	if ok := ui.Confirm("Proceed with connecting Testkube Cloud?"); !ok {
		return
	}

	spinner := ui.NewSpinner("Connecting Testkube Cloud")

	err = HelmUpgradeOrInstallTestkubeCloud(connectOpts, cfg)
	ui.ExitOnError("Installing Testkube Cloud", err)
	spinner.Success()

	ui.NL()

	spinner = ui.NewSpinner("Scaling down not needed Testkube OSS components")
	// let's scale down deployment of mongo
	if connectOpts.MongoReplicas == 0 {
		KubectlScaleDeployment(connectOpts.Namespace, "testkube-mongodb", connectOpts.MongoReplicas)
		KubectlScaleDeployment(connectOpts.Namespace, "testkube-minio-testkube", connectOpts.MinioReplicas)
		KubectlScaleDeployment(connectOpts.Namespace, "testkube-dashboard", connectOpts.DashboardReplicas)
	}
	spinner.Success()

	err = PopulateAgentDataToContext(connectOpts, cfg)
	ui.ExitOnError("Populating agent data to context", err)

	ui.NL(2)
}

// getToken returns chan to wait for token from http server
// client need to call / endpoint with token query param
func getToken() (chan string, error) {
	tokenChan := make(chan string)

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

func uiGetOrganizationId(token string) (string, error) {
	// Choose organization from orgs available
	orgs, err := getOrganizations(token)
	if err != nil {
		pterm.Error.Println("Failed to get organizations", err.Error())
		return "", fmt.Errorf("failed to get organizations: %s", err.Error())
	}

	orgNames := getNames(orgs)
	orgName, _ := pterm.DefaultInteractiveSelect.
		WithOptions(orgNames).
		Show()

	orgId := findId(orgs, orgName)

	return orgId, nil
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
