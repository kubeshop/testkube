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

	"github.com/dustin/go-humanize"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
)

type Service struct {
	Name        string
	Count       int64
	Parallelism int64
	Timeout     string
	Matrix      map[string][]interface{}
	Shards      map[string][]interface{}
	Ready       string
	Error       string
	Content     *testworkflowsv1.SpawnContent
	PodTemplate corev1.PodTemplateSpec
}

func (svc *Service) ShardIndexAt(index int64) int64 {
	return index % svc.Count
}

func (svc *Service) CombinationIndexAt(index int64) int64 {
	return (index - svc.ShardIndexAt(index)) / svc.Count
}

func (svc *Service) Combinations() int64 {
	return CountCombinations(svc.Matrix)
}

func (svc *Service) Total() int64 {
	return svc.Count * svc.Combinations()
}

func (svc *Service) MatrixAt(index int64) map[string]interface{} {
	return GetMatrixValues(svc.Matrix, svc.CombinationIndexAt(index))
}

func (svc *Service) ShardsAt(index int64) map[string][]interface{} {
	return GetShardValues(svc.Matrix, svc.ShardIndexAt(index), svc.Count)
}

func (svc *Service) MachineAt(index int64) expressionstcl.Machine {
	// Get basic indices
	combinations := svc.Combinations()
	shardIndex := svc.ShardIndexAt(index)
	combinationIndex := svc.CombinationIndexAt(index)

	// Compute values for this instance
	matrixValues := svc.MatrixAt(index)
	shardValues := svc.ShardsAt(index)

	return expressionstcl.NewMachine().
		Register("index", index).
		Register("count", combinations*svc.Count).
		Register("matrixIndex", combinationIndex).
		Register("matrixCount", combinations).
		Register("matrix", matrixValues).
		Register("shardIndex", shardIndex).
		Register("shardsCount", svc.Count).
		Register("shard", shardValues)
}

func (svc *Service) Pod(ref string, index int64, machines ...expressionstcl.Machine) (*corev1.Pod, error) {
	// Get details for current position
	machines = append(machines, svc.MachineAt(index))

	// Build a pod
	spec := svc.PodTemplate.DeepCopy()
	err := expressionstcl.FinalizeForce(&spec, machines...)
	if err != nil {
		return nil, fmt.Errorf("resolving pod schema: %w", err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s-%s-%d", env.ExecutionId(), ref, svc.Name, index),
			Namespace:   env.Namespace(),
			Annotations: spec.Annotations,
		},
		Spec: spec.Spec,
	}
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
	}
	if pod.Spec.SecurityContext.FSGroup == nil {
		pod.Spec.SecurityContext.FSGroup = common.Ptr(testworkflowprocessor.DefaultFsGroup)
	}

	// Append defaults for the pod containers
	for i := range pod.Spec.InitContainers {
		applyContainerDefaults(&pod.Spec.InitContainers[i], i)
	}
	for i := range pod.Spec.Containers {
		applyContainerDefaults(&pod.Spec.Containers[i], i)
	}

	// Apply control labels
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[testworkflowprocessor.ExecutionIdLabelName] = env.ExecutionId()
	pod.Labels[testworkflowprocessor.ExecutionAssistingPodRefName] = ref

	// Configure the default headless service
	pod.Labels[testworkflowprocessor.AssistingPodServiceName] = "true"
	if pod.Spec.Subdomain == "" {
		pod.Spec.Subdomain = testworkflowprocessor.AssistingPodServiceName
	}
	if pod.Spec.Hostname == "" {
		pod.Spec.Hostname = fmt.Sprintf("%s-%s-%d", env.ExecutionId(), svc.Name, index)
	}

	return pod, nil
}

func (svc *Service) Files(index int64, machines ...expressionstcl.Machine) (map[string]string, error) {
	// Ignore when there are no files expected
	if svc.Content == nil || len(svc.Content.Files) == 0 {
		return nil, nil
	}

	// Prepare data for computation
	files := make(map[string]string, len(svc.Content.Files))
	machines = append(machines, svc.MachineAt(index))

	// Compute all files
	var err error
	for fileIndex, file := range svc.Content.Files {
		files[file.Path], err = expressionstcl.EvalTemplate(file.Content, machines...)
		if err != nil {
			return nil, fmt.Errorf("resolving %s file (%s): %w", humanize.Ordinal(fileIndex), file.Path, err)
		}
	}
	return files, nil
}

func applyContainerDefaults(container *corev1.Container, index int) {
	if container.Name == "" {
		container.Name = fmt.Sprintf("c%d-%s", index, rand.String(5))
	}
	if container.SecurityContext == nil {
		container.SecurityContext = &corev1.SecurityContext{}
	}
	if container.SecurityContext.RunAsGroup == nil {
		container.SecurityContext.RunAsGroup = common.Ptr(testworkflowprocessor.DefaultFsGroup)
	}
}
