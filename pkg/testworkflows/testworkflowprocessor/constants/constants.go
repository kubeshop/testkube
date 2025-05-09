package constants

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	DefaultInternalPath = "/.tktw"
	DefaultDataPath     = "/data"
	DefaultFsGroup      = int64(1001)
	// TODO: move to the execution worker (?)
	ResourceIdLabelName             = "testkube.io/resource"
	RootResourceIdLabelName         = "testkube.io/root"
	RunnerIdLabelName               = "testkube.io/runner"
	GroupIdLabelName                = "testkube.io/contextGroup"
	SignatureAnnotationName         = "testkube.io/signature"
	SignatureAnnotationFieldPath    = "metadata.annotations['" + SignatureAnnotationName + "']"
	ScheduledAtAnnotationName       = "testkube.io/at"
	SpecAnnotationName              = "testkube.io/spec"
	SpecAnnotationFieldPath         = "metadata.annotations['" + SpecAnnotationName + "']"
	InternalAnnotationName          = "testkube.io/config"
	InternalAnnotationFieldPath     = "metadata.annotations['" + InternalAnnotationName + "']"
	OpenSourceOperationErrorMessage = "operation is not available when running the Testkube Agent in the standalone mode"
	RootOperationName               = "root"
)

var (
	InternalBinPath        = filepath.Join(DefaultInternalPath, "bin")
	DefaultShellPath       = filepath.Join(InternalBinPath, "sh")
	DefaultInitPath        = filepath.Join(DefaultInternalPath, "init")
	DefaultToolkitPath     = filepath.Join(DefaultInternalPath, "toolkit")
	DefaultTransferDirPath = filepath.Join(DefaultInternalPath, "transfer")
	DefaultTmpDirPath      = "/tmp"
	DefaultTransferPort    = 60433
	DefaultShellHeader     = "set -e\n"
	DefaultContainerConfig = testworkflowsv1.ContainerConfig{
		Image: DefaultInitImage,
		Env: []testworkflowsv1.EnvVar{
			{EnvVar: corev1.EnvVar{Name: "CI", Value: "1"}},
		},
	}
	DefaultInitImage                             = getInitImage()
	DefaultToolkitImage                          = getToolkitImage()
	DefaultInitImageBusyboxBinaryPath            = "/.tktw-bin"
	ErrOpenSourceExecuteOperationIsNotAvailable  = errors.New(`"execute" ` + OpenSourceOperationErrorMessage)
	ErrOpenSourceParallelOperationIsNotAvailable = errors.New(`"parallel" ` + OpenSourceOperationErrorMessage)
	ErrOpenSourceServicesOperationIsNotAvailable = errors.New(`"services" ` + OpenSourceOperationErrorMessage)
)

func getInitImage() string {
	// Handle getter in the toolkit
	if os.Getenv("TK_CFG") != "" {
		return config.Config().Worker.InitImage
	}

	// Handle executor's getter
	img := os.Getenv("TESTKUBE_TW_INIT_IMAGE")
	if img == "" {
		ver := version.Version
		if ver == "" || ver == "dev" {
			ver = "latest"
		}
		img = fmt.Sprintf("kubeshop/testkube-tw-init:%s", ver)
	}
	return img
}

func getToolkitImage() string {
	// Handle getter in the toolkit
	if os.Getenv("TK_CFG") != "" {
		return config.Config().Worker.ToolkitImage
	}

	// Handle executor's getter
	img := os.Getenv("TESTKUBE_TW_TOOLKIT_IMAGE")
	if img == "" {
		ver := version.Version
		if ver == "" || ver == "dev" {
			ver = "latest"
		}
		img = fmt.Sprintf("kubeshop/testkube-tw-toolkit:%s", ver)
	}
	return img
}
