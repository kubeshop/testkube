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
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

const maxConfigMapSize = 950 * 1024

//go:generate mockgen -destination=./mock_resources.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" Resources
type Resources interface {
	// Output

	ConfigMaps() []corev1.ConfigMap
	Secrets() []corev1.Secret
	Volumes() []corev1.Volume

	// Internal data

	NextRef() string
	IsComputedFile(volume corev1.VolumeMount) bool

	// Mutations

	AddConfigMap(configMap corev1.ConfigMap) Resources
	AddSecret(secret corev1.Secret) Resources
	AddVolume(volume corev1.Volume) Resources
	AddTextFile(file string) (corev1.VolumeMount, error)
	AddBinaryFile(file []byte) (corev1.VolumeMount, error)
}

type resources struct {
	// Job & Pod resources & data
	VolumesValue []corev1.Volume `expr:"force"`

	// Actual Kubernetes resources to use
	SecretsValue    []corev1.Secret    `expr:"force"`
	ConfigMapsValue []corev1.ConfigMap `expr:"force"`

	// Computation data
	currentConfigMapStorage   *corev1.ConfigMap
	estimatedConfigMapStorage int
	refCount                  uint64
}

func NewResources() Resources {
	return &resources{}
}

func (r *resources) NextRef() string {
	return fmt.Sprintf("r%s%s", rand.String(5), strconv.FormatUint(r.refCount, 36))
}

func (r *resources) ConfigMaps() []corev1.ConfigMap {
	return r.ConfigMapsValue
}

func (r *resources) Secrets() []corev1.Secret {
	return r.SecretsValue
}

func (r *resources) Volumes() []corev1.Volume {
	return r.VolumesValue
}

func (r *resources) IsComputedFile(volumeMount corev1.VolumeMount) bool {
	if volumeMount.SubPath == "" {
		return false
	}
	for _, v := range r.VolumesValue {
		if v.Name == volumeMount.Name {
			if v.VolumeSource.ConfigMap == nil {
				return false
			}
			for _, c := range r.ConfigMapsValue {
				if c.Name == v.VolumeSource.ConfigMap.Name {
					file := c.Data[volumeMount.SubPath]
					return !expressionstcl.IsTemplateStringWithoutExpressions(file)
				}
			}
		}
	}
	return false
}

func (r *resources) AddVolume(volume corev1.Volume) Resources {
	r.VolumesValue = append(r.VolumesValue, volume)
	return r
}

func (r *resources) AddConfigMap(configMap corev1.ConfigMap) Resources {
	r.ConfigMapsValue = append(r.ConfigMapsValue, configMap)
	return r
}

func (r *resources) AddSecret(secret corev1.Secret) Resources {
	r.SecretsValue = append(r.SecretsValue, secret)
	return r
}

func (r *resources) getInternalConfigMapStorage(size int) *corev1.ConfigMap {
	if size > maxConfigMapSize {
		return nil
	}
	if size+r.estimatedConfigMapStorage > maxConfigMapSize || r.currentConfigMapStorage == nil {
		ref := r.NextRef()
		r.ConfigMapsValue = append(r.ConfigMapsValue, corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "{{execution.id}}-" + ref},
			Immutable:  common.Ptr(true),
			Data:       map[string]string{},
			BinaryData: map[string][]byte{},
		})
		r.currentConfigMapStorage = &r.ConfigMapsValue[len(r.ConfigMapsValue)-1]
		r.VolumesValue = append(r.VolumesValue, corev1.Volume{
			Name: r.currentConfigMapStorage.Name + "-vol",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: r.currentConfigMapStorage.Name},
				},
			},
		})
	}
	return r.currentConfigMapStorage
}

func (r *resources) AddTextFile(file string) (corev1.VolumeMount, error) {
	cfg := r.getInternalConfigMapStorage(len(file))
	if cfg == nil {
		return corev1.VolumeMount{}, errors.New("the maximum file size is 950KiB")
	}
	r.estimatedConfigMapStorage += len(file)
	ref := r.NextRef()
	cfg.Data[ref] = file
	return corev1.VolumeMount{
		Name:     cfg.Name + "-vol",
		ReadOnly: true,
		SubPath:  ref,
	}, nil
}

func (r *resources) AddBinaryFile(file []byte) (corev1.VolumeMount, error) {
	cfg := r.getInternalConfigMapStorage(len(file))
	if cfg == nil {
		return corev1.VolumeMount{}, errors.New("the maximum file size is 950KiB")
	}
	r.estimatedConfigMapStorage += len(file)
	ref := r.NextRef()
	cfg.BinaryData[ref] = file
	return corev1.VolumeMount{
		Name:     cfg.Name + "-vol",
		ReadOnly: true,
		SubPath:  ref,
	}, nil
}
