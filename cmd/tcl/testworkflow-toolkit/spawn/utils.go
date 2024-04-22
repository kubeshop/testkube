// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

func ServiceLabel(name string) string {
	return ui.LightCyan(name)
}

func InstanceLabel(name string, index int64, total int64) string {
	zeros := strings.Repeat("0", len(fmt.Sprintf("%d", total))-len(fmt.Sprintf("%d", index+1)))
	return ServiceLabel(name) + ui.Cyan(fmt.Sprintf("/%s%d", zeros, index+1))
}

func PodName(ref, svcName string, index int64) string {
	return fmt.Sprintf("%s-%s-%s-%d", env.ExecutionId(), ref, svcName, index)
}

func BuildResources(services []Service, ref string, machines ...expressionstcl.Machine) ([][]*corev1.Pod, testworkflowprocessor.ConfigMapFiles, transfer.Server, error) {
	// Initialize state
	srv := transfer.NewServer(constants.DefaultTransferDirPath, env.IP(), constants.DefaultTransferPort)
	pods := make([][]*corev1.Pod, len(services))
	storage := testworkflowprocessor.NewConfigMapFiles(fmt.Sprintf("%s-%s-vol", env.ExecutionId(), ref), map[string]string{
		constants.ExecutionIdLabelName:         env.ExecutionId(),
		constants.ExecutionAssistingPodRefName: ref,
	})

	for svcIndex, svc := range services {
		pods[svcIndex] = make([]*corev1.Pod, svc.Params.Count)
		for i := int64(0); i < svc.Params.Count; i++ {
			pod, err := svc.Pod(ref, i, machines...)
			if err != nil {
				return nil, nil, nil, err
			}
			files, err := svc.FilesMap(i, machines...)
			if err != nil {
				return nil, nil, nil, err
			}
			transfer, err := svc.ComputedTransfer(i, machines...)
			if err != nil {
				return nil, nil, nil, err
			}
			for path, content := range files {
				// Apply file
				mount, volume, err := storage.AddTextFile(content)
				if err != nil {
					return nil, nil, nil, errors.Wrapf(err, "%s: %s instance: file %s", svc.Name, humanize.Ordinal(int(i+1)), path)
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

			transferArgs := make([]string, len(transfer))
			transferVolumeMounts := make([]corev1.VolumeMount, len(transfer))
			for transferIndex, v := range transfer {
				volumeName := fmt.Sprintf("%s-t%d", ref, transferIndex)
				entry, err := srv.Include(v.From, v.Files)
				if err != nil {
					return nil, nil, nil, errors.Wrapf(err, "%s: %s instance: transfer.%d", svc.Name, humanize.Ordinal(int(i+1)), transferIndex)
				}
				transferArgs[transferIndex] = shellquote.Join(fmt.Sprintf("%s=%s", v.MountPath, entry.Url))
				transferVolumeMounts[transferIndex] = corev1.VolumeMount{Name: volumeName, MountPath: v.MountPath}
				pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
					Name:         volumeName,
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				})
			}

			if len(transfer) > 0 {
				init := corev1.Container{
					Name:            fmt.Sprintf("%s-tktw-init", ref),
					Image:           env.Config().Images.Toolkit,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/toolkit", "tarball"},
					Args:            transferArgs,
					Env: []corev1.EnvVar{
						{Name: "TK_NS", Value: env.Namespace()},
						{Name: "TK_REF", Value: fmt.Sprintf("%s-%d", ref, i)},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsGroup: common.Ptr(constants.DefaultFsGroup),
					},
				}
				applyContainerDefaults(&init, 0)
				pod.Spec.InitContainers = append([]corev1.Container{init}, pod.Spec.InitContainers...)
			}

			for i := range pod.Spec.InitContainers {
				pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts, transferVolumeMounts...)
			}

			for i := range pod.Spec.Containers {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, transferVolumeMounts...)
			}

			pods[svcIndex][i] = pod
		}
	}

	return pods, storage, srv, nil
}

func EachService(services []Service, pods [][]*corev1.Pod, fn func(svc Service, svcIndex int, pod *corev1.Pod, index int64, combinations int64)) {
	// Prepare wait group to wait for all services
	var wg sync.WaitGroup
	wg.Add(len(services))

	// Initialize all the services
	for i, v := range services {
		go func(svc Service, svcIndex int) {
			combinations := svc.Params.MatrixCount

			var swg sync.WaitGroup
			swg.Add(int(combinations * svc.Params.ShardCount))
			sema := make(chan struct{}, svc.Parallelism)

			for index, pod := range pods[svcIndex] {
				sema <- struct{}{}
				go func(index int64, pod *corev1.Pod) {
					defer func() {
						<-sema
						swg.Done()
					}()

					fn(svc, svcIndex, pod, index, combinations)
				}(int64(index), pod)
			}

			swg.Wait()
			wg.Done()
		}(v, i)
	}

	// Wait until all processes will be finished
	wg.Wait()
}

func FetchLogs(ctx context.Context, clientSet kubernetes.Interface, svc Service, pod *corev1.Pod) (io.Reader, error) {
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			req := clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Timestamps: true,
				Container:  container.Name,
			})
			stream, err := req.Stream(ctx)
			if err == nil {
				_, err = io.Copy(writer, stream)
				if err != nil && !errors.Is(err, io.EOF) {
					writer.Write([]byte(fmt.Sprintf("\n%s error: cannot read '%s' container logs further: %s", time.Time{}.Format(testworkflowcontroller.KubernetesLogTimeFormat), container.Name, strings.ReplaceAll(err.Error(), "\n", " "))))
				}
			} else {
				writer.Write([]byte(fmt.Sprintf("%s error: cannot read '%s' container logs: %s", time.Time{}.Format(testworkflowcontroller.KubernetesLogTimeFormat), container.Name, strings.ReplaceAll(err.Error(), "\n", " "))))
			}
			writer.Write([]byte("\n"))
		}
	}()
	return reader, nil
}

func DeletePod(ctx context.Context, clientSet kubernetes.Interface, pod *corev1.Pod) error {
	err := clientSet.CoreV1().Pods(env.Namespace()).Delete(ctx, pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
	})
	if err != nil && errors2.IsNotFound(err) {
		err = nil
	}
	return err
}

func DeletePodAndSaveLogs(ctx context.Context, clientSet kubernetes.Interface, storage artifacts.InternalArtifactStorage, svc Service, pod *corev1.Pod, index int64) error {
	logs, err := FetchLogs(context.Background(), clientSet, svc, pod)
	if err != nil {
		fmt.Printf("%s: warning: failed to fetch logs from finished pod: %s\n", InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
	} else {
		filePath := fmt.Sprintf("logs/%s/%d.log", svc.Name, index)
		err = storage.SaveStream(filePath, logs)
		if err == nil {
			data.PrintOutput(env.Ref(), "service-status", ServiceStatus{
				Name:  svc.Name,
				Index: index,
				Logs:  storage.FullPath(filePath),
			})
		} else {
			fmt.Printf("%s: warning: error while saving logs: %s\n", InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
		}
	}
	return DeletePod(ctx, clientSet, pod)
}
