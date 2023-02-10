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
		ui.InfoGrid(map[string]string{
			"Organization ID": cloudContext.Organization,
			"Environment ID ": cloudContext.Environment,
			"API Key        ": text.Obfuscate(cloudContext.ApiKey),
			"API URI        ": cloudContext.ApiUri,
			"Agent Key      ": text.Obfuscate(cloudContext.AgentKey),
			"Agent URI      ": cloudContext.AgentUri,
		})
	} else {
		ui.InfoGrid(map[string]string{
			"context type": "kubeconfig",
		})
	}
}
