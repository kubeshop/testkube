package common

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

func UiPrintContext(cfg config.Data) {
	ui.Warn("Your current context is set to", string(cfg.ContextType))
	ui.NL()

	if cfg.ContextType == config.ContextTypeCloud {
		contextData := map[string]string{
			"Organization ID": cfg.CloudContext.Organization,
			"Environment ID ": cfg.CloudContext.Environment,
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
