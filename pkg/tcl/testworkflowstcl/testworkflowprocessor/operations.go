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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

func ProcessExecute(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Execute == nil {
		return nil, nil
	}
	container = container.CreateChild()
	stage := stage.NewContainerStage(layer.NextRef(), container)
	stage.SetRetryPolicy(step.Retry)
	hasWorkflows := len(step.Execute.Workflows) > 0
	hasTests := len(step.Execute.Tests) > 0

	// Fail if there is nothing to run
	if !hasTests && !hasWorkflows {
		return nil, errors.New("no test workflows and tests provided to the 'execute' step")
	}

	container.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "execute").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))
	args := make([]string, 0)
	for _, t := range step.Execute.Tests {
		b, err := json.Marshal(t)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing Test")
		}
		args = append(args, "-t", expressions.NewStringValue(string(b)).Template())
	}
	for _, w := range step.Execute.Workflows {
		b, err := json.Marshal(w)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing TestWorkflow")
		}
		args = append(args, "-w", expressions.NewStringValue(string(b)).Template())
	}
	if step.Execute.Async {
		args = append(args, "--async")
	}
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

func ProcessParallel(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Parallel == nil {
		return nil, nil
	}

	stage := stage.NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCategory("Run in parallel")

	// Inherit container defaults
	inherited := common.Ptr(stage.Container().ToContainerConfig())
	inherited.VolumeMounts = nil
	step.Parallel.Container = testworkflowresolver.MergeContainerConfig(inherited, step.Parallel.Container)

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "parallel").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))

	v, err := json.Marshal(step.Parallel)
	if err != nil {
		return nil, errors.Wrap(err, "parallel: marshalling error")
	}
	stage.Container().SetArgs(expressions.NewStringValue(string(v)).Template())

	return stage, nil
}

func ProcessServicesStart(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if len(step.Services) == 0 {
		return nil, nil
	}

	// TODO: Think of better way to pass the data between steps
	podsRef := layer.NextRef()
	container.AppendEnv(corev1.EnvVar{Name: "TK_SVC_REF", Value: podsRef})

	stage := stage.NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCategory("Start services")

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "services", "-g", "{{env.TK_SVC_REF}}").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))

	args := make([]string, 0, len(step.Services))
	for name := range step.Services {
		v, err := json.Marshal(step.Services[name])
		if err != nil {
			return nil, errors.Wrapf(err, "services[%s]: marshalling error", name)
		}
		args = append(args, fmt.Sprintf("%s=%s", name, expressions.NewStringValue(string(v)).Template()))
	}
	stage.Container().SetArgs(args...)

	return stage, nil
}

func ProcessServicesStop(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if len(step.Services) == 0 {
		return nil, nil
	}

	stage := stage.NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCondition("always") // TODO: actually, it's enough to do it when "Start services" is not skipped
	stage.SetOptional(true)
	stage.SetCategory("Stop services")

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "kill", "{{env.TK_SVC_REF}}").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))

	args := make([]string, 0)
	for name, v := range step.Services {
		if v.Logs != nil {
			args = append(args, "-l", fmt.Sprintf("%s=%s", name, *v.Logs))
		}
	}
	stage.Container().SetArgs(args...)

	return stage, nil
}
