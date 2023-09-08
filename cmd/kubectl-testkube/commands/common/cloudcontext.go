package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

func UiPrintContext(cfg config.Data) {
	ui.Warn("Your current context is set to", string(cfg.ContextType))
	ui.NL()

	if cfg.ContextType == config.ContextTypeCloud {
		contextData := map[string]string{
			"Organization   ": cfg.CloudContext.OrganizationName + ui.DarkGray(" ("+cfg.CloudContext.OrganizationId+")"),
			"Environment    ": cfg.CloudContext.EnvironmentName + ui.DarkGray(" ("+cfg.CloudContext.EnvironmentId+")"),
			"API Key        ": text.Obfuscate(cfg.CloudContext.ApiKey),
			"API URI        ": cfg.CloudContext.ApiUri,
			"Namespace      ": cfg.Namespace,
		}

		// add agent information only when need to change agent data, it's usually not needed in usual workflow
		if ui.Verbose {
			contextData["Agent Key"] = text.Obfuscate(cfg.CloudContext.AgentKey)
			contextData["Agent URI"] = cfg.CloudContext.AgentUri
		}

		ui.InfoGrid(contextData)
	} else {
		ui.InfoGrid(map[string]string{
			"Namespace        ": cfg.Namespace,
			"Telemetry Enabled": fmt.Sprintf("%t", cfg.TelemetryEnabled),
		})
	}
}

func UiCloudContextValidationError(err error) {
	ui.NL()
	ui.Errf("Validating cloud context failed: %s", err.Error())
	ui.NL()
	ui.Info("Please set valid cloud context using `testkube set context` with valid values")
	ui.NL()
	ui.ShellCommand(" testkube set context -c cloud -e tkcenv_XXX -o tkcorg_XXX -k tkcapi_XXX")
	ui.NL()
}

func UiContextHeader(cmd *cobra.Command, cfg config.Data) {
	// only show header when output is pretty
	if cmd.Flag("output") != nil && cmd.Flag("output").Value.String() != "pretty" {
		return
	}

	header := "\n"
	separator := "   "

	orgName := cfg.CloudContext.OrganizationName
	if orgName == "" {
		orgName = cfg.CloudContext.OrganizationId
	}
	envName := cfg.CloudContext.EnvironmentName
	if envName == "" {
		envName = cfg.CloudContext.EnvironmentId
	}

	if cfg.ContextType == config.ContextTypeCloud {
		header += ui.DarkGray("Context: ") + ui.White(cfg.ContextType) + ui.DarkGray(" ("+Version+")") + separator
		header += ui.DarkGray("Namespace: ") + ui.White(cfg.Namespace) + separator
		header += ui.DarkGray("Org: ") + ui.White(orgName) + separator
		header += ui.DarkGray("Env: ") + ui.White(envName)
	} else {
		header += ui.DarkGray("Context: ") + ui.White(cfg.ContextType) + ui.DarkGray(" ("+Version+")") + separator
		header += ui.DarkGray("Namespace: ") + ui.White(cmd.Flag("namespace").Value.String())
	}

	fmt.Println(header)
	fmt.Println(strings.Repeat("-", calculateStringSize(header)))
}

// calculateStringSize calculates the length of a string, excluding shell color codes.
func calculateStringSize(s string) int {
	// Regular expression to match ANSI escape codes.
	re := regexp.MustCompile(`\x1b[^m]*m`)
	// Remove the escape codes from the string.
	s = re.ReplaceAllString(s, "")
	// Return the length of the string.
	return len(s) - 1
}
