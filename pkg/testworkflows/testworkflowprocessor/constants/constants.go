package constants

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	DefaultInternalPath             = "/.tktw"
	DefaultDataPath                 = "/data"
	DefaultTerminationLogPath       = "/dev/termination-log"
	DefaultFsGroup                  = int64(1001)
	ResourceIdLabelName             = "testworkflowid"
	RootResourceIdLabelName         = "testworkflowid-root"
	GroupIdLabelName                = "testworkflowid-group"
	SignatureAnnotationName         = "testworkflows.testkube.io/signature"
	SpecAnnotationName              = "testworkflows.testkube.io/spec"
	RFC3339Millis                   = "2006-01-02T15:04:05.000Z07:00"
	OpenSourceOperationErrorMessage = "operation is not available when running the Testkube Agent in the standalone mode"
	RootOperationName               = "root"
)

var (
	InternalBinPath        = filepath.Join(DefaultInternalPath, "bin")
	DefaultShellPath       = filepath.Join(InternalBinPath, "sh")
	DefaultInitPath        = filepath.Join(DefaultInternalPath, "init")
	DefaultStatePath       = filepath.Join(DefaultInternalPath, "state")
	DefaultTransferDirPath = filepath.Join(DefaultInternalPath, "transfer")
	DefaultTmpDirPath      = filepath.Join(DefaultInternalPath, "tmp")
	DefaultTransferPort    = 60433
	DefaultShellHeader     = "set -e\n"
	DefaultContainerConfig = testworkflowsv1.ContainerConfig{
		Image: DefaultInitImage,
		Env: []corev1.EnvVar{
			{Name: "CI", Value: "1"},
		},
	}
	DefaultInitImage                             = getInitImage()
	DefaultToolkitImage                          = getToolkitImage()
	ErrOpenSourceExecuteOperationIsNotAvailable  = errors.New(`"execute" ` + OpenSourceOperationErrorMessage)
	ErrOpenSourceParallelOperationIsNotAvailable = errors.New(`"parallel" ` + OpenSourceOperationErrorMessage)
	ErrOpenSourceServicesOperationIsNotAvailable = errors.New(`"services" ` + OpenSourceOperationErrorMessage)
)

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

func getInitImage() string {
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
