// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"fmt"
	"slices"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

func BuildResources(services []Service, ref string, machines ...expressionstcl.Machine) ([][]*corev1.Pod, testworkflowprocessor.ConfigMapFiles, error) {
	// Initialize list of pods to schedule
	pods := make([][]*corev1.Pod, len(services))
	storage := testworkflowprocessor.NewConfigMapFiles(fmt.Sprintf("%s-%s-vol", env.ExecutionId(), ref), map[string]string{
		constants.ExecutionIdLabelName:         env.ExecutionId(),
		constants.ExecutionAssistingPodRefName: ref,
	})

	for svcIndex, svc := range services {
		combinations := CountCombinations(svc.Matrix)
		pods[svcIndex] = make([]*corev1.Pod, svc.Count*combinations)
		for i := int64(0); i < svc.Count*combinations; i++ {
			pod, err := svc.Pod(ref, i, machines...)
			if err != nil {
				return nil, nil, err
			}
			files, err := svc.FilesMap(i, machines...)
			if err != nil {
				return nil, nil, err
			}
			for path, content := range files {
				// Apply file
				mount, volume, err := storage.AddTextFile(content)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "%s: %s instance: file %s", svc.Name, humanize.Ordinal(int(i)), path)
				}

				// Append the volume mount
				mount.MountPath = path
				for i := range pod.Spec.InitContainers {
					pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts, mount)
				}
				for i := range pod.Spec.Containers {
					pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, mount)
				}

				// Append the volume if it's not yet added
				if !slices.ContainsFunc(pod.Spec.Volumes, func(v corev1.Volume) bool {
					return v.Name == mount.Name
				}) {
					pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
				}
			}

			pods[svcIndex][i] = pod
		}
	}

	return pods, storage, nil
}
