// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
)

func ProcessDelay(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Delay == "" {
		return nil, nil
	}
	t, err := time.ParseDuration(step.Delay)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid duration: %s", step.Delay))
	}
	shell := container.CreateChild().
		SetCommand("sleep").
		SetArgs(fmt.Sprintf("%g", t.Seconds()))
	stage := NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory(fmt.Sprintf("Delay: %s", step.Delay))
	return stage, nil
}

func ProcessShellCommand(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Shell == "" {
		return nil, nil
	}
	shell := container.CreateChild().SetCommand(defaultShell).SetArgs("-c", step.Shell)
	stage := NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory("Run shell command")
	return stage, nil
}

func ProcessRunCommand(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Run == nil {
		return nil, nil
	}
	container = container.CreateChild().ApplyCR(&step.Run.ContainerConfig)
	stage := NewContainerStage(layer.NextRef(), container)
	stage.SetCategory("Run")
	return stage, nil
}

func ProcessNestedSteps(p InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	group := NewGroupStage(layer.NextRef(), true)
	for _, n := range step.Steps {
		stage, err := p.Process(layer, container.CreateChild(), n)
		if err != nil {
			return nil, err
		}
		group.Add(stage)
	}
	return group, nil
}

func ProcessExecute(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Execute == nil {
		return nil, nil
	}
	container = container.CreateChild()
	stage := NewContainerStage(layer.NextRef(), container)
	hasWorkflows := len(step.Execute.Workflows) > 0
	hasTests := len(step.Execute.Tests) > 0

	// Fail if there is nothing to run
	if !hasTests && !hasWorkflows {
		return nil, errors.New("no test workflows and tests provided to the 'execute' step")
	}

	container.
		SetImage(defaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "execute").
		AppendEnvMap(map[string]string{
			"TK_REF":     stage.Ref(),
			"TK_NS":      "{{internal.namespace}}",
			"TK_API_URL": "{{internal.api.url}}",
			"TK_WF":      "{{workflow.name}}",
			"TK_EX":      "{{execution.id}}",
		})
	args := make([]string, 0)
	for _, t := range step.Execute.Tests {
		args = append(args, "-t", t.Name)
	}
	for _, w := range step.Execute.Workflows {
		if len(w.Config) == 0 {
			args = append(args, "-w", w.Name)
		} else {
			v, _ := json.Marshal(testworkflows.MapConfigValueKubeToAPI(w.Config))
			args = append(args, "-w", fmt.Sprintf(`%s={"config":%s}`, w.Name, v))
		}
	}
	// TODO: Support "async"
	if step.Execute.Parallelism > 0 {
		args = append(args, "-p", strconv.Itoa(int(step.Execute.Parallelism)))
	}
	container.SetArgs(args...)

	// Add default label
	types := make([]string, 0)
	if hasWorkflows {
		types = append(types, "test workflows")
	}
	if hasTests {
		types = append(types, "tests")
	}
	stage.SetCategory("Execute " + strings.Join(types, " & "))

	return stage, nil
}

func ProcessContentFiles(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Content == nil {
		return nil, nil
	}
	for _, f := range step.Content.Files {
		if f.ContentFrom == nil {
			vm, err := layer.AddTextFile(f.Content)
			if err != nil {
				return nil, fmt.Errorf("file %s: could not append: %s", f.Path, err.Error())
			}
			vm.MountPath = f.Path
			container.AppendVolumeMounts(vm)
			continue
		}

		volRef := "{{execution.id}}-" + layer.NextRef()

		if f.ContentFrom.ConfigMapKeyRef != nil {
			layer.AddVolume(corev1.Volume{
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
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else if f.ContentFrom.SecretKeyRef != nil {
			layer.AddVolume(corev1.Volume{
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
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else if f.ContentFrom.FieldRef != nil || f.ContentFrom.ResourceFieldRef != nil {
			layer.AddVolume(corev1.Volume{
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
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else {
			return nil, fmt.Errorf("file %s: unrecognized ContentFrom provided for file", f.Path)
		}
	}
	return nil, nil
}
