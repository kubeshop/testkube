package commands

import (
	"os"
	"os/exec"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/version"
)

// Moved from testworkflowprocessor/constants to reduce init process size
const (
	defaultInitImageBusyboxBinaryPath = "/.tktw-bin"
)

func Setup(config lite.ActionSetup) error {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Copy the init process
	stdoutUnsafe.Print("Configuring init process...")
	if config.CopyInit {
		// Use environment variable if set (for testing), otherwise use default
		initSource := "/init"
		if envPath := os.Getenv("TESTKUBE_TW_INIT_BINARY_PATH"); envPath != "" {
			initSource = envPath
		}
		err := exec.Command("cp", initSource, constants.InitPath).Run()
		if err != nil {
			stdoutUnsafe.Error(" error\n")
			stdoutUnsafe.Errorf("  failed to copy the /init process: %s\n", err.Error())
			return err
		}
		stdoutUnsafe.Print(" done\n")
	} else {
		stdoutUnsafe.Print(" skipped\n")
	}

	// Copy the toolkit
	stdoutUnsafe.Print("Configuring toolkit...")
	if config.CopyToolkit {
		// Use environment variable if set (for testing), otherwise use default
		toolkitSource := "/toolkit"
		if envPath := os.Getenv("TESTKUBE_TW_TOOLKIT_BINARY_PATH"); envPath != "" {
			toolkitSource = envPath
		}
		err := exec.Command("cp", toolkitSource, constants.ToolkitPath).Run()
		if err != nil {
			stdoutUnsafe.Error(" error\n")
			stdoutUnsafe.Errorf("  failed to copy the /toolkit utilities: %s\n", err.Error())
			return err
		}
		stdoutUnsafe.Print(" done\n")
	} else {
		stdoutUnsafe.Print(" skipped\n")
	}

	// Copy the shell and useful libraries
	stdoutUnsafe.Print("Configuring shell...")
	if config.CopyBinaries {
		// Use environment variable if set (for testing), otherwise use default
		binariesSource := defaultInitImageBusyboxBinaryPath
		if envPath := os.Getenv("TESTKUBE_TW_BUSYBOX_BINARY_PATH"); envPath != "" {
			binariesSource = envPath
		}
		// Use `cp` on the whole directory, as it has plenty of files, which lead to the same FS block.
		// Copying individual files will lead to high FS usage
		err := exec.Command("cp", "-rf", binariesSource, constants.InternalBinPath).Run()
		if err != nil {
			stdoutUnsafe.Error(" error\n")
			stdoutUnsafe.Errorf("  failed to copy the binaries: %s\n", err.Error())
			return err
		}
		stdoutUnsafe.Print(" done\n")
	} else {
		stdoutUnsafe.Print(" skipped\n")
	}

	// Expose debugging Pod information
	stdoutUnsafe.Output(constants.InitStepName, "pod", map[string]string{
		"name":               os.Getenv(constants.EnvPodName),
		"nodeName":           os.Getenv(constants.EnvNodeName),
		"namespace":          os.Getenv(constants.EnvNamespaceName),
		"serviceAccountName": os.Getenv(constants.EnvServiceAccountName),
		"agent":              version.Version,
		"toolkit":            stripCommonImagePrefix(os.Getenv("TESTKUBE_TW_TOOLKIT_IMAGE"), "testkube-tw-toolkit"),
		"init":               stripCommonImagePrefix(os.Getenv("TESTKUBE_TW_INIT_IMAGE"), "testkube-tw-init"),
	})

	return nil
}

func stripCommonImagePrefix(image, common string) string {
	if !strings.HasPrefix(image, "docker.io/") {
		return image
	}
	image = image[10:]
	if !strings.HasPrefix(image, "kubeshop/") {
		return image
	}
	image = image[9:]
	if !strings.HasPrefix(image, common+":") {
		return image
	}
	return image[len(common)+1:]
}
