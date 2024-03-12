// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowresolver"
)

const maxConfigMapFileSize = 950 * 1024

//go:generate mockgen -destination=./mock_intermediate.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" Intermediate
type Intermediate interface {
	RefCounter

	ContainerDefaults() Container
	PodConfig() testworkflowsv1.PodConfig
	JobConfig() testworkflowsv1.JobConfig

	ConfigMaps() []corev1.ConfigMap
	Secrets() []corev1.Secret
	Volumes() []corev1.Volume

	AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate
	AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate

	AddConfigMap(configMap corev1.ConfigMap) Intermediate
	AddSecret(secret corev1.Secret) Intermediate
	AddVolume(volume corev1.Volume) Intermediate

	AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount

	AddTextFile(file string) (corev1.VolumeMount, error)
	AddBinaryFile(file []byte) (corev1.VolumeMount, error)
}

type intermediate struct {
	refCounter

	// Routine
	Root      GroupStage `expr:"include"`
	Container Container  `expr:"include"`

	// Job & Pod resources & data
	Pod testworkflowsv1.PodConfig `expr:"include"`
	Job testworkflowsv1.JobConfig `expr:"include"`

	// Actual Kubernetes resources to use
	Secs []corev1.Secret    `expr:"force"`
	Cfgs []corev1.ConfigMap `expr:"force"`

	// Storing files
	currentConfigMapStorage   *corev1.ConfigMap
	estimatedConfigMapStorage int
}

func NewIntermediate() Intermediate {
	return &intermediate{
		Root:      NewGroupStage("", true),
		Container: NewContainer(),
	}
}

func (s *intermediate) ContainerDefaults() Container {
	return s.Container
}

func (s *intermediate) JobConfig() testworkflowsv1.JobConfig {
	return s.Job
}

func (s *intermediate) PodConfig() testworkflowsv1.PodConfig {
	return s.Pod
}

func (s *intermediate) ConfigMaps() []corev1.ConfigMap {
	return s.Cfgs
}

func (s *intermediate) Secrets() []corev1.Secret {
	return s.Secs
}

func (s *intermediate) Volumes() []corev1.Volume {
	return s.Pod.Volumes
}

func (s *intermediate) AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate {
	s.Job = *testworkflowresolver.MergeJobConfig(&s.Job, cfg)
	return s
}

func (s *intermediate) AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate {
	s.Pod = *testworkflowresolver.MergePodConfig(&s.Pod, cfg)
	return s
}

func (s *intermediate) AddVolume(volume corev1.Volume) Intermediate {
	s.Pod.Volumes = append(s.Pod.Volumes, volume)
	return s
}

func (s *intermediate) AddConfigMap(configMap corev1.ConfigMap) Intermediate {
	s.Cfgs = append(s.Cfgs, configMap)
	return s
}

func (s *intermediate) AddSecret(secret corev1.Secret) Intermediate {
	s.Secs = append(s.Secs, secret)
	return s
}

func (s *intermediate) AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount {
	if source == nil {
		source = &corev1.EmptyDirVolumeSource{}
	}
	ref := s.NextRef()
	s.AddVolume(corev1.Volume{Name: ref, VolumeSource: corev1.VolumeSource{EmptyDir: source}})
	return corev1.VolumeMount{Name: ref, MountPath: mountPath}
}

// Handling files

func (s *intermediate) getInternalConfigMapStorage(size int) *corev1.ConfigMap {
	if size > maxConfigMapFileSize {
		return nil
	}
	if size+s.estimatedConfigMapStorage > maxConfigMapFileSize || s.currentConfigMapStorage == nil {
		ref := s.NextRef()
		s.Cfgs = append(s.Cfgs, corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "{{execution.id}}-" + ref},
			Immutable:  common.Ptr(true),
			Data:       map[string]string{},
			BinaryData: map[string][]byte{},
		})
		s.currentConfigMapStorage = &s.Cfgs[len(s.Cfgs)-1]
		s.Pod.Volumes = append(s.Pod.Volumes, corev1.Volume{
			Name: s.currentConfigMapStorage.Name + "-vol",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: s.currentConfigMapStorage.Name},
				},
			},
		})
	}
	return s.currentConfigMapStorage
}

func (s *intermediate) AddTextFile(file string) (corev1.VolumeMount, error) {
	cfg := s.getInternalConfigMapStorage(len(file))
	if cfg == nil {
		return corev1.VolumeMount{}, errors.New("the maximum file size is 950KiB")
	}
	s.estimatedConfigMapStorage += len(file)
	ref := s.NextRef()
	cfg.Data[ref] = file
	return corev1.VolumeMount{
		Name:     cfg.Name + "-vol",
		ReadOnly: true,
		SubPath:  ref,
	}, nil
}

func (s *intermediate) AddBinaryFile(file []byte) (corev1.VolumeMount, error) {
	cfg := s.getInternalConfigMapStorage(len(file))
	if cfg == nil {
		return corev1.VolumeMount{}, errors.New("the maximum file size is 950KiB")
	}
	s.estimatedConfigMapStorage += len(file)
	ref := s.NextRef()
	cfg.BinaryData[ref] = file
	return corev1.VolumeMount{
		Name:     cfg.Name + "-vol",
		ReadOnly: true,
		SubPath:  ref,
	}, nil
}
