// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"crypto/sha256"
	"fmt"
	"maps"

	"github.com/dustin/go-humanize"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
)

const maxConfigMapFileSize = 950 * 1024

type configMapFiles struct {
	prefix  string
	Labels  map[string]string             `expr:"force"`
	Cfgs    []corev1.ConfigMap            `expr:"force"`
	Vols    []corev1.Volume               `expr:"force"`
	Mounts  map[string]corev1.VolumeMount `expr:"force"`
	VolRefs map[string]int
}

type ConfigMapFiles interface {
	Volumes() []corev1.Volume
	ConfigMaps() []corev1.ConfigMap
	AddTextFile(content string) (corev1.VolumeMount, corev1.Volume, error)
	AddFile(content []byte) (corev1.VolumeMount, corev1.Volume, error)
	FilesCount() int
}

func NewConfigMapFiles(prefix string, labels map[string]string) ConfigMapFiles {
	return &configMapFiles{
		prefix:  prefix,
		Labels:  labels,
		Mounts:  make(map[string]corev1.VolumeMount),
		VolRefs: make(map[string]int),
	}
}

func (c *configMapFiles) Volumes() []corev1.Volume {
	return c.Vols
}

func (c *configMapFiles) ConfigMaps() []corev1.ConfigMap {
	return c.Cfgs
}

func (c *configMapFiles) FilesCount() int {
	return len(c.Mounts)
}

func (c *configMapFiles) next(minBytes int) (*corev1.ConfigMap, *corev1.Volume, int) {
	for i := range c.Cfgs {
		size := 0
		for k := range c.Cfgs[i].Data {
			size += len(c.Cfgs[i].Data[k])
		}
		for k := range c.Cfgs[i].BinaryData {
			size += len(c.Cfgs[i].BinaryData[k])
		}
		if size+minBytes < maxConfigMapFileSize {
			return &c.Cfgs[i], &c.Vols[i], i
		}
	}
	name := fmt.Sprintf("%s-c%d", c.prefix, len(c.Cfgs))
	cfg := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Data:       map[string]string{},
		BinaryData: map[string][]byte{},
		Immutable:  common.Ptr(true),
	}
	maps.Copy(cfg.Labels, c.Labels)
	index := len(c.Cfgs)
	c.Cfgs = append(c.Cfgs, cfg)
	c.Vols = append(c.Vols, corev1.Volume{
		Name: cfg.Name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cfg.Name},
			},
		},
	})
	return &c.Cfgs[index], &c.Vols[index], index
}

func (c *configMapFiles) AddTextFile(file string) (corev1.VolumeMount, corev1.Volume, error) {
	if len(file) > maxConfigMapFileSize {
		return corev1.VolumeMount{}, corev1.Volume{}, fmt.Errorf("the maximum file size is %s", humanize.Bytes(maxConfigMapFileSize))
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(file)))
	if _, ok := c.Mounts[hash]; !ok {
		cfg, vol, index := c.next(len(file))
		key := fmt.Sprintf("%d", len(cfg.Data)+len(cfg.BinaryData))
		cfg.Data[key] = file
		c.Mounts[hash] = corev1.VolumeMount{Name: vol.Name, ReadOnly: true, SubPath: key}
		c.VolRefs[hash] = index
	}
	return c.Mounts[hash], c.Vols[c.VolRefs[hash]], nil
}

func (c *configMapFiles) AddFile(file []byte) (corev1.VolumeMount, corev1.Volume, error) {
	if len(file) > maxConfigMapFileSize {
		return corev1.VolumeMount{}, corev1.Volume{}, fmt.Errorf("the maximum file size is %s", humanize.Bytes(maxConfigMapFileSize))
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(file))
	if _, ok := c.Mounts[hash]; !ok {
		cfg, vol, index := c.next(len(file))
		key := fmt.Sprintf("%d", len(cfg.Data)+len(cfg.BinaryData))
		cfg.BinaryData[key] = file
		c.Mounts[hash] = corev1.VolumeMount{Name: vol.Name, ReadOnly: true, SubPath: key}
		c.VolRefs[hash] = index
	}
	return c.Mounts[hash], c.Vols[c.VolRefs[hash]], nil
}
