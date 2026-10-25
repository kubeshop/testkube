// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// MinIO no longer publishes public images, so devbox has to use the image the
// chart deploys rather than drift back to a minio/minio tag that cannot be pulled.
func TestMinioPod_UsesChartImage(t *testing.T) {
	raw, err := os.ReadFile("../../../../../k8s/helm/testkube-api/values.yaml")
	require.NoError(t, err)
	var values struct {
		Minio struct {
			Image struct {
				Repository string `json:"repository"`
				Tag        string `json:"tag"`
			} `json:"image"`
		} `json:"minio"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &values))

	pod := minioPod()
	require.Len(t, pod.Spec.Containers, 1)
	assert.Equal(t, values.Minio.Image.Repository+":"+values.Minio.Image.Tag, pod.Spec.Containers[0].Image)
}

// The image runs as a non-root user that cannot create /data, so MinIO exits
// with "file access denied" unless the data directory is a mounted volume.
func TestMinioPod_DataDirIsVolume(t *testing.T) {
	pod := minioPod()
	container := pod.Spec.Containers[0]
	assert.Equal(t, []string{"minio"}, container.Command)
	require.Contains(t, container.Args, "/data")

	require.Len(t, container.VolumeMounts, 1)
	mount := container.VolumeMounts[0]
	assert.Equal(t, "/data", mount.MountPath)
	require.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, mount.Name, pod.Spec.Volumes[0].Name)
	assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
}
