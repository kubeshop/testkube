package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/version"
)

func Setup(config testworkflowprocessor.ActionSetup) {
	// Copy the init process TODO: only when it is required
	fmt.Print("Configuring init process...")
	if config.CopyInit {
		err := exec.Command("cp", "/init", data.InitPath).Run()
		if err != nil {
			fmt.Println(ui.Red(" error"))
			data.Failf(data.CodeInternal, "failed to copy the /init process: %s", err.Error())
		}
		fmt.Println(" done")
	} else {
		fmt.Println(" skipped")
	}

	// Copy the shell and useful libraries TODO: only when it is required
	fmt.Print("Configuring shell...")
	if config.CopyBinaries {
		// Use `cp` on the whole directory, as it has plenty of files, which lead to the same FS block.
		// Copying individual files will lead to high FS usage
		err := exec.Command("cp", "-rf", "/bin", data.InternalBinPath).Run()
		if err != nil {
			fmt.Println(ui.Red(" error"))
			data.Failf(data.CodeInternal, "failed to copy the /init process: %s", err.Error())
		}
		fmt.Println(" done")
	} else {
		fmt.Println(" skipped")
	}

	// Expose debugging Pod information
	data.PrintOutput(data.InitStepName, "pod", map[string]string{
		"name":               os.Getenv(constants.EnvPodName),
		"nodeName":           os.Getenv(constants.EnvNodeName),
		"namespace":          os.Getenv(constants.EnvNamespaceName),
		"serviceAccountName": os.Getenv(constants.EnvServiceAccountName),
		"agent":              version.Version,
		// TODO: Recover that
		//"toolkit": stripCommonImagePrefix(getToolkitImage(), "testkube-tw-toolkit"),
		//"init": stripCommonImagePrefix(getInitImage(), "testkube-tw-init"),
	})
}
