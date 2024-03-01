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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

//go:generate mockgen -destination=./mock_processor.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" Processor
type Processor interface {
	Process(resolvedWorkflow *testworkflowsv1.TestWorkflow) (Scope, error)
}

type processor struct {
}

func New() Processor {
	return &processor{}
}

func (p *processor) ProcessStep(s Scope, parent GroupStage, container Container, step testworkflowsv1.Step) error {
	// Configure defaults
	container.ApplyCR(step.Container)

	// Build an initial group for the inner items
	self := NewGroupStage(s.Resources().NextRef())
	self.SetName(step.Name)
	self.SetOptional(step.Optional).SetNegative(step.Negative)
	if step.Condition != "" {
		self.SetCondition(step.Condition)
	} else {
		self.SetCondition("passed")
	}

	// Load files
	if step.Content != nil {
		for _, f := range step.Content.Files {
			if f.ContentFrom != nil {
				volRef := "{{execution.id}}-" + s.Resources().NextRef()

				if f.ContentFrom.ConfigMapKeyRef != nil {
					s.Resources().AddVolume(corev1.Volume{
						Name: volRef,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: f.ContentFrom.ConfigMapKeyRef.LocalObjectReference,
								Items:                []corev1.KeyToPath{{Key: f.ContentFrom.ConfigMapKeyRef.Key, Path: "file"}},
								DefaultMode:          f.Mode,
								Optional:             f.ContentFrom.ConfigMapKeyRef.Optional,
							},
						},
					})
					container.AppendVolumeMounts(corev1.VolumeMount{
						Name:      volRef,
						ReadOnly:  true,
						MountPath: f.Path,
						SubPath:   "file",
					})
				} else if f.ContentFrom.SecretKeyRef != nil {
					s.Resources().AddVolume(corev1.Volume{
						Name: volRef,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  f.ContentFrom.SecretKeyRef.Name,
								Items:       []corev1.KeyToPath{{Key: f.ContentFrom.SecretKeyRef.Key, Path: "file"}},
								DefaultMode: f.Mode,
								Optional:    f.ContentFrom.SecretKeyRef.Optional,
							},
						},
					})
					container.AppendVolumeMounts(corev1.VolumeMount{
						Name:      volRef,
						ReadOnly:  true,
						MountPath: f.Path,
						SubPath:   "file",
					})
				} else if f.ContentFrom.FieldRef != nil || f.ContentFrom.ResourceFieldRef != nil {
					s.Resources().AddVolume(corev1.Volume{
						Name: volRef,
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{{
									DownwardAPI: &corev1.DownwardAPIProjection{
										Items: []corev1.DownwardAPIVolumeFile{{
											Path:             "file",
											FieldRef:         f.ContentFrom.FieldRef,
											ResourceFieldRef: f.ContentFrom.ResourceFieldRef,
											Mode:             f.Mode,
										}},
									},
								}},
								DefaultMode: f.Mode,
							},
						},
					})
					container.AppendVolumeMounts(corev1.VolumeMount{
						Name:      volRef,
						ReadOnly:  true,
						MountPath: f.Path,
						SubPath:   "file",
					})
				} else {
					return fmt.Errorf("file %s: unrecognized ContentFrom provided for file", f.Path)
				}
			} else {
				vm, err := s.Resources().AddTextFile(f.Content)
				if err != nil {
					return fmt.Errorf("file %s: could not append: %s", f.Path, err.Error())
				}
				vm.MountPath = f.Path
				container.AppendVolumeMounts(vm)
			}
		}
	}
	// TODO: Load Git repository

	// Resolve steps
	if step.Shell != "" {
		// TODO: Create Stage from Resources?
		shell := container.CreateChild().SetCommand(defaultShell).SetArgs("-c", step.Shell)
		stage := NewContainerStage(s.Resources().NextRef(), shell)
		self.Add(stage)
	}

	if step.Run != nil {
		run := container.CreateChild().ApplyCR(&step.Run.ContainerConfig)
		stage := NewContainerStage(s.Resources().NextRef(), run)
		self.Add(stage)
	}

	// Nested steps
	for _, n := range step.Steps {
		err := p.ProcessStep(s, self, container.CreateChild(), n)
		if err != nil {
			return err
		}
	}

	// TODO: Other steps

	// TODO: Artifacts steps

	// Include the stage
	parent.Add(self)

	return nil
}

func (p *processor) Process(resolvedWorkflow *testworkflowsv1.TestWorkflow) (Scope, error) {
	var err error
	s := NewScope().
		AppendPodConfig(resolvedWorkflow.Spec.Pod).
		AppendJobConfig(resolvedWorkflow.Spec.Job)

	internalRef := s.Resources().NextRef()
	dataRef := s.Resources().NextRef()

	s.Resources().
		AddVolume(corev1.Volume{Name: internalRef, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}).
		AddVolume(corev1.Volume{Name: dataRef, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})

	// Process container defaults
	s.ContainerDefaults().
		ApplyCR(defaultContainerConfig.DeepCopy()).
		ApplyCR(resolvedWorkflow.Spec.Container).
		AppendVolumeMounts(corev1.VolumeMount{Name: internalRef, MountPath: defaultInternalPath}).
		AppendVolumeMounts(corev1.VolumeMount{Name: dataRef, MountPath: defaultDataPath})

	// Process steps
	for i := range resolvedWorkflow.Spec.Setup {
		err = p.ProcessStep(s, s.RootStage(), s.ContainerDefaults().CreateChild(), resolvedWorkflow.Spec.Setup[i])
		if err != nil {
			return nil, errors.Wrap(err, "error processing `setup`")
		}
	}
	for i := range resolvedWorkflow.Spec.Steps {
		err = p.ProcessStep(s, s.RootStage(), s.ContainerDefaults().CreateChild(), resolvedWorkflow.Spec.Steps[i])
		if err != nil {
			return nil, errors.Wrap(err, "error processing `steps`")
		}
	}
	for i := range resolvedWorkflow.Spec.After {
		err = p.ProcessStep(s, s.RootStage(), s.ContainerDefaults().CreateChild(), resolvedWorkflow.Spec.After[i])
		if err != nil {
			return nil, errors.Wrap(err, "error processing `after`")
		}
	}

	l := len(s.RootStage().Flatten())

	if l == 0 {
		return nil, errors.New("test workflow has no steps to run")
	}

	return s, nil
}
