package export

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
)

func PrintInstructions(opts Options, localPath string) {
	ui.NL()
	ui.H2("Usage export complete")
	ui.Info("Saved export to:", localPath)
	ui.NL()
	ui.H2("Next steps")
	ui.Info("1. Upload the zip via the Testkube backoffice license usage import.")
	ui.Info("2. Review the import preview and confirm manifest checks pass.")
	if !opts.KeepRelease {
		ui.Info("3. Clean up the export Job release when finished:")
		uninstall := fmt.Sprintf("   helm uninstall %s", opts.Release)
		if opts.Namespace != "" {
			uninstall += fmt.Sprintf(" -n %s", opts.Namespace)
		}
		if opts.KubeContext != "" {
			uninstall += fmt.Sprintf(" --kube-context %s", opts.KubeContext)
		}
		ui.Info(uninstall)
	} else {
		ui.Info("3. Release kept (--keep-release). Uninstall manually when finished.")
	}
	ui.NL()
	ui.Info("Chart values reference:")
	ui.Info("  https://github.com/kubeshop/testkube-cloud-api/tree/main/helm/testkube-usage-export")
	ui.NL()
}
