package context

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

func uiPrintCloudContext(contextType string, cloudContext config.CloudContext) {
	ui.Warn("Your current context is set to", contextType)
	ui.NL()

	if contextType == string(config.ContextTypeCloud) {
		contextData := map[string]string{
			"Organization ID": cloudContext.Organization,
			"Environment ID ": cloudContext.Environment,
			"API Key        ": text.Obfuscate(cloudContext.ApiKey),
			"API URI        ": cloudContext.ApiUri,
		}

		// add agent information only when need to change agent data, it's usually not needed in usual workflow
		if ui.Verbose {
			contextData["Agent Key"] = text.Obfuscate(cloudContext.AgentKey)
			contextData["Agent URI"] = cloudContext.AgentUri
		}

		ui.InfoGrid(contextData)
	} else {
		ui.InfoGrid(map[string]string{
			"context type": "kubeconfig",
		})
	}
}
