package export

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
)

func PrintInstructions(opts Options, localPath string) {
	ui.NL()
	ui.H2("Usage export complete")
	ui.NL()
	ui.H2("Next steps")
	ui.Info(fmt.Sprintf("1. Your usage export is saved at: %s", localPath))
	ui.Info("2. Send this file to Testkube staff for processing.")
	if opts.KeepRelease {
		ui.Info("3. When you are finished, uninstall the export release:")
		uninstall := fmt.Sprintf("   helm uninstall %s", opts.Release)
		if opts.Namespace != "" {
			uninstall += fmt.Sprintf(" -n %s", opts.Namespace)
		}
		if opts.KubeContext != "" {
			uninstall += fmt.Sprintf(" --kube-context %s", opts.KubeContext)
		}
		ui.Info(uninstall)
	} else {
		ui.Info(fmt.Sprintf("3. Helm release %q was removed from the cluster.", opts.Release))
	}
	ui.NL()
}
