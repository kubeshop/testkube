// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

const (
	defaultImage         = "busybox:1.36.1"
	defaultShell         = "/bin/sh"
	defaultInternalPath  = "/.tktw"
	defaultDataPath      = "/data"
	executionIdLabelName = "testworkflowid"
)

var (
	defaultInitPath  = filepath.Join(defaultInternalPath, "init")
	defaultStatePath = filepath.Join(defaultInternalPath, "state")
)

var (
	defaultInitImage       = getInitImage()
	defaultContainerConfig = testworkflowsv1.ContainerConfig{
		Image: defaultImage,
		Env: []corev1.EnvVar{
			{Name: "CI", Value: "1"},
		},
	}
)

func getInitImage() string {
	img := os.Getenv("TESTKUBE_TW_INIT_IMAGE")
	if img == "" {
		version := common.Version
		if version == "" || version == "dev" {
			version = "latest"
		}
		img = fmt.Sprintf("kubeshop/testkube-tw-init:%s", version)
	}
	return img
}
