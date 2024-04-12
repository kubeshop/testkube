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
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

type ServiceStatus struct {
	Name        string     `json:"name"`
	Index       int64      `json:"index"`
	Description string     `json:"description,omitempty"`
	Logs        string     `json:"logs,omitempty"`
	Status      string     `json:"status,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

type Service struct {
	Name        string
	Description string
	Strategy    testworkflowsv1.SpawnStrategy
	Count       int64
	Parallelism int64
	Logs        bool
	Timeout     string
	Matrix      map[string][]interface{}
	Shards      map[string][]interface{}
	Ready       string
	Error       string
	Files       []testworkflowsv1.ContentFile
	Transfer    []testworkflowsv1.ContentTransfer
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
	return GetShardValues(svc.Shards, svc.ShardIndexAt(index), svc.Count)
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

func (svc *Service) TimeoutDuration(index int64, machines ...expressionstcl.Machine) (*time.Duration, error) {
	if svc.Timeout == "" {
		return nil, nil
	}
	// Get details for current position
	machines = append(machines, svc.MachineAt(index))
	durationStr, err := expressionstcl.EvalTemplate(svc.Timeout, machines...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve duration template: %s", svc.Timeout)
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse duration: %s: %s", svc.Timeout, durationStr)
	}
	return &duration, nil
}

func (svc *Service) StrategyAt(index int64, machines ...expressionstcl.Machine) (testworkflowsv1.SpawnStrategy, error) {
	if svc.Strategy == "" {
		return "", nil
	}
	// Get details for current position
	machines = append(machines, svc.MachineAt(index))
	text, err := expressionstcl.EvalTemplate(string(svc.Strategy), machines...)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve strategy template: %s", svc.Strategy)
	}
	if text != "" && text != string(testworkflowsv1.SpawnStrategyEven) {
		return "", errors.Wrapf(err, "unknown distribution strategy: %s", text)
	}
	return testworkflowsv1.SpawnStrategy(text), nil
}

func (svc *Service) Pod(ref string, index int64, machines ...expressionstcl.Machine) (*corev1.Pod, error) {
	// Get details for current position
	machines = append(machines, svc.MachineAt(index))

	// Copy a pod
	spec := svc.PodTemplate.DeepCopy()

	// Resolve pod
	err := expressionstcl.FinalizeForce(&spec, machines...)
	if err != nil {
		return nil, fmt.Errorf("resolving pod schema: %w", err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        PodName(ref, svc.Name, index),
			Namespace:   env.Namespace(),
			Annotations: spec.Annotations,
		},
		Spec: spec.Spec,
	}

	// Apply security rules
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
	}
	if pod.Spec.SecurityContext.FSGroup == nil {
		pod.Spec.SecurityContext.FSGroup = common.Ptr(constants.DefaultFsGroup)
	}

	// Append defaults for the pod containers
	for i := range pod.Spec.InitContainers {
		applyContainerDefaults(&pod.Spec.InitContainers[i], i)
	}
	for i := range pod.Spec.Containers {
		applyContainerDefaults(&pod.Spec.Containers[i], len(pod.Spec.InitContainers)+i)
	}

	// Apply control labels
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[constants.ExecutionIdLabelName] = env.ExecutionId()
	pod.Labels[constants.ExecutionAssistingPodRefName] = ref

	// Configure the default headless service
	pod.Labels[constants.AssistingPodServiceName] = "true"
	if pod.Spec.Subdomain == "" {
		pod.Spec.Subdomain = constants.AssistingPodServiceName
	}
	if pod.Spec.Hostname == "" {
		pod.Spec.Hostname = fmt.Sprintf("%s-%s-%d", env.ExecutionId(), svc.Name, index)
	}

	// Apply strategy
	strategy, err := svc.StrategyAt(index, machines...)
	if err != nil {
		return nil, err
	}
	if strategy == testworkflowsv1.SpawnStrategyEven {
		topology := rand.String(5)
		label := fmt.Sprintf("skew%s", topology)
		value := fmt.Sprintf("%s-%s", env.ExecutionId(), topology)
		pod.Labels[label] = value
		spec.Spec.TopologySpreadConstraints = append(spec.Spec.TopologySpreadConstraints, corev1.TopologySpreadConstraint{
			MaxSkew:           1,
			TopologyKey:       "kubernetes.io/hostname",
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{label: value},
			},
		})
	}

	return pod, nil
}

func (svc *Service) DescriptionAt(index int64, machines ...expressionstcl.Machine) (string, error) {
	if svc.Description == "" {
		return svc.Description, nil
	}
	// Get details for current position
	machines = append(machines, svc.MachineAt(index))
	description, err := expressionstcl.EvalTemplate(svc.Description, machines...)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve description template: %s", svc.Description)
	}
	return description, nil
}

func (svc *Service) FilesMap(index int64, machines ...expressionstcl.Machine) (map[string]string, error) {
	// Ignore when there are no files expected
	if len(svc.Files) == 0 {
		return nil, nil
	}

	// Prepare data for computation
	files := make(map[string]string, len(svc.Files))
	machines = append(machines, svc.MachineAt(index))

	// Compute all files
	var err error
	for fileIndex, file := range svc.Files {
		files[file.Path], err = expressionstcl.EvalTemplate(file.Content, machines...)
		if err != nil {
			return nil, fmt.Errorf("resolving %s file (%s): %w", humanize.Ordinal(fileIndex+1), file.Path, err)
		}
	}
	return files, nil
}

func (svc *Service) ComputedTransfer(index int64, machines ...expressionstcl.Machine) ([]testworkflowsv1.ContentTransfer, error) {
	// Ignore when there are no files expected for transfer
	if len(svc.Transfer) == 0 {
		return nil, nil
	}

	// Prepare data for computation
	transfer := make([]testworkflowsv1.ContentTransfer, len(svc.Transfer))
	machines = append(machines, svc.MachineAt(index))

	// Compute
	for i, content := range svc.Transfer {
		err := expressionstcl.Finalize(&content, machines...)
		if err != nil {
			return nil, fmt.Errorf("resolving %s transfer (%s -> %s): %w", humanize.Ordinal(i+1), content.From, content.MountPath, err)
		}
		if len(content.Files) == 0 {
			content.Files = []string{"**/*"}
		}
		for i := range content.Files {
			if !filepath.IsAbs(content.Files[i]) {
				content.Files[i] = filepath.Join(content.From, content.Files[i])
			}
		}
		transfer[i] = content
	}
	return transfer, nil
}

func (svc *Service) Eval(expr string, state ServiceState, index int64, machines ...expressionstcl.Machine) (*bool, error) {
	machines = append([]expressionstcl.Machine{state.Machine(), svc.MachineAt(index)}, machines...)
	ex, err := expressionstcl.EvalExpressionPartial(expr, machines...)
	if err != nil {
		return nil, err
	}
	if ex.Static() == nil {
		return nil, nil
	}
	v, _ := ex.Static().BoolValue()
	return &v, nil
}

func (svc *Service) EvalReady(state ServiceState, index int64, machines ...expressionstcl.Machine) (*bool, error) {
	return svc.Eval(svc.Ready, state, index, machines...)
}

func (svc *Service) EvalError(state ServiceState, index int64, machines ...expressionstcl.Machine) (*bool, error) {
	return svc.Eval(svc.Error, state, index, machines...)
}

func applyContainerDefaults(container *corev1.Container, index int) {
	if container.Name == "" {
		container.Name = fmt.Sprintf("c%d-%s", index, rand.String(5))
	}
	if container.SecurityContext == nil {
		container.SecurityContext = &corev1.SecurityContext{}
	}
	if container.SecurityContext.RunAsGroup == nil {
		container.SecurityContext.RunAsGroup = common.Ptr(constants.DefaultFsGroup)
	}
}
