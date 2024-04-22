// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package constants

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	DefaultInternalPath         = "/.tktw"
	DefaultDataPath             = "/data"
	DefaultTerminationLogPath   = "/dev/termination-log"
	DefaultFsGroup              = int64(1001)
	ExecutionIdLabelName        = "testworkflowid"
	ExecutionIdMainPodLabelName = "testworkflowid-main"
	SignatureAnnotationName     = "testworkflows.testkube.io/signature"
)

var (
	InternalBinPath        = filepath.Join(DefaultInternalPath, "bin")
	DefaultShellPath       = filepath.Join(InternalBinPath, "sh")
	DefaultInitPath        = filepath.Join(DefaultInternalPath, "init")
	DefaultStatePath       = filepath.Join(DefaultInternalPath, "state")
	DefaultTransferDirPath = filepath.Join(DefaultInternalPath, "transfer")
	DefaultTransferPort    = 60433
	InitScript             = strings.TrimSpace(strings.NewReplacer(
		"<bin>", InternalBinPath,
		"<init>", DefaultInitPath,
		"<state>", DefaultStatePath,
		"<terminationLog>", DefaultTerminationLogPath,
	).Replace(`
set -e
trap '[ $? -eq 0 ] && exit 0 || echo -n "failed,1" > <terminationLog> && exit 1' EXIT
echo "Configuring state..."
touch <state> && chmod 777 <state>
echo "Configuring init process..."
cp /init <init>
echo "Configuring shell..."
cp -rf /bin /.tktw/bin
echo -n ',0' > <terminationLog> && echo 'Done.' && exit 0
	`))
	DefaultShellHeader     = "set -e\n"
	DefaultContainerConfig = testworkflowsv1.ContainerConfig{
		Image: DefaultInitImage,
		Env: []corev1.EnvVar{
			{Name: "CI", Value: "1"},
		},
	}
	DefaultInitImage    = getInitImage()
	DefaultToolkitImage = getToolkitImage()
)

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
