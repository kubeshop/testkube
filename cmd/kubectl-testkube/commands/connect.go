package commands

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/pterm/pterm"
	"github.com/skratchdot/open-golang/open"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"c"},
		Short:   "Testkube Cloud interactive mode",
		Long:    `Interactive mode for Testkube Cloud migrations`,
		Run:     cloudConnect,
	}

	return cmd
}

const (
	listenAddr      = "127.0.0.1:7899"
	redirectUrl     = "http://" + listenAddr
	tokenQueryParam = "token"
)

func cloudConnect(cmd *cobra.Command, args []string) {

	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgLightMagenta, pterm.Bold)).WithMargin(0)
	text := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgDarkGray)).
		WithTextStyle(pterm.NewStyle(pterm.BgDefault, pterm.FgGray)).WithMargin(0)

	header.Println("Connect your cloud environment:")
	text.Println("You can learn more about connecting your Testkube instance to the Cloud here:\nhttps://docs.testkube.io/etc")
	header.Println("You can safely switch between connecting Cloud and disconnecting without losing your data.")

	// TODO: implement context info
	header.Println("Current status of your Testkube instance")
	text.Println(`
Context:    On-Premise (local OSS context)
Cluster:    cluster
Namespace:  testkube
`)

	header.Println("Login")
	text.Println("Please open the following link in your browser and log in:\nhttps://cloud.testkube.io/login?redirect_uri=....")

	// open browser with login page and redirect to localhost:7899
	open.Run("https://cloud.testkube.io/?redirectUri=" + url.QueryEscape(redirectUrl))

	// wait for token received to browser
	s := spinner("waiting for auth token")
	tokenChan, err := getToken()
	if err != nil {
		s.Fail("Failed to get auth token", err.Error())
		return
	}

	var token string
	// TODO: add timeout
	select {
	case token = <-tokenChan:
		s.Success()
	case <-time.After(5 * time.Minute):
		s.Fail("Timeout waiting for auth token")
	}
	ui.NL()
	// Choose organization from orgs available
	orgs, err := getOrganizations(token)
	if err != nil {
		pterm.Error.Println("Failed to get organizations", err.Error())
		return
	}

	orgNames := getNames(orgs)
	orgName, _ := pterm.DefaultInteractiveSelect.
		WithOptions(orgNames).
		Show()

	orgId := findId(orgs, orgName)

	pterm.Info.Println("Selected organization", orgName, orgId)
	ui.NL()

	envName, _ := pterm.DefaultInteractiveTextInput.
		WithMultiLine(false).
		Show("Tell us the name of your environment")

	pterm.Println()

	pterm.Info.Printfln("You answered: %s", result)
}

func checkInfo() pterm.TextPrinter {
	s := pterm.Info.WithMessageStyle(pterm.NewStyle(pterm.FgWhite, pterm.BgDefault))
	s.Prefix = pterm.Prefix{Text: " ️", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)}
	return s
}

func checkOk() pterm.TextPrinter {
	return pterm.Info.
		WithMessageStyle(pterm.NewStyle(pterm.FgWhite, pterm.BgDefault)).
		WithPrefix(pterm.Prefix{Text: "✅", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)})
}
func checkFail() pterm.TextPrinter {
	return pterm.Info.
		WithMessageStyle(pterm.NewStyle(pterm.FgRed, pterm.BgDefault)).
		WithPrefix(pterm.Prefix{Text: "❗", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)})

}

func spinner(t string) *pterm.SpinnerPrinter {
	s := pterm.DefaultSpinner
	s.InfoPrinter = checkOk()
	s.SuccessPrinter = checkOk()
	s.InfoPrinter = checkInfo()
	s.FailPrinter = checkFail()
	sp, _ := s.Start(t)
	return sp
}

// TODO implement
func getToken() (chan string, error) {
	tokenChan := make(chan string)

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tokenChan <- r.URL.Query().Get("token")
		})

		log.Fatal(http.ListenAndServe(listenAddr, nil))
	}()

	return tokenChan, nil
}

type Organization struct {
	Id   string
	Name string
}

// TODO implement
func getOrganizations(token string) ([]cloudclient.Organization, error) {
	c := cloudclient.NewOrganizationsClient(token)
	return c.Get()
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
