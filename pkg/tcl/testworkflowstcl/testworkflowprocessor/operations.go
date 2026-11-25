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
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
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

	// Allow to combine it within other containers
	stage.SetPure(true)

	// Fail if there is nothing to run
	if !hasTests && !hasWorkflows {
		return nil, errors.New("no test workflows and tests provided to the 'execute' step")
	}

	container.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(constants.DefaultToolkitPath, "execute").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))
	// Marshal all data for base64 encoding
	type ExecuteData struct {
		Tests       []json.RawMessage `json:"tests,omitempty"`
		Workflows   []json.RawMessage `json:"workflows,omitempty"`
		Async       bool              `json:"async,omitempty"`
		Parallelism int               `json:"parallelism,omitempty"`
	}

	executeData := ExecuteData{
		Async: step.Execute.Async,
	}
	if step.Execute.Parallelism > 0 {
		executeData.Parallelism = int(step.Execute.Parallelism)
	}

	// Marshal tests
	for _, t := range step.Execute.Tests {
		b, err := json.Marshal(t)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing Test")
		}
		executeData.Tests = append(executeData.Tests, json.RawMessage(b))
	}

	// Marshal workflows
	for _, w := range step.Execute.Workflows {
		b, err := json.Marshal(w)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing TestWorkflow")
		}
		executeData.Workflows = append(executeData.Workflows, json.RawMessage(b))
	}

	// Base64 encode to prevent testworkflow-init from prematurely resolving expressions.
	// Execute workflows can contain expressions like {{ index + 1 }} and {{ count }} that
	// need to be evaluated when the workflows are distributed to workers, not in the init context.
	encoded, err := expressionstcl.EncodeBase64JSON(executeData)
	if err != nil {
		return nil, errors.Wrap(err, "execute: encoding error")
	}
	container.SetArgs("--base64", encoded)

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

// ProcessAgent builds the stage that runs one turn of an AI Agent. The AI Agent runs in the
// control plane, so the stage only needs the toolkit, as the execute stage does.
func ProcessAgent(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Agent == nil {
		return nil, nil
	}
	if step.Agent.Name == "" {
		return nil, errors.New("no AI Agent name provided to the 'agent' step")
	}
	// The control plane compares the name with the stored workflow before it runs the AI Agent.
	// This check runs after the config values resolve, so an expression that remains has a value
	// only in the pod.
	if !expressions.IsTemplateStringWithoutExpressions(step.Agent.Name) {
		return nil, errors.Errorf("agent: name %q: the name of the AI Agent accepts a literal or a config value only", step.Agent.Name)
	}
	if before := operationsBeforeAgent(step); len(before) > 0 {
		return nil, errors.Errorf("agent: an agent step runs only the AI Agent. Move %s to an earlier step", strings.Join(before, ", "))
	}
	container = container.CreateChild()
	stage := stage.NewContainerStage(layer.NextRef(), container)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Run AI Agent")

	// Allow to combine it within other containers
	stage.SetPure(true)

	// Base64 encode to prevent testworkflow-init from resolving the expressions. The prompt
	// reads the outputs of earlier steps, and the value of an output can contain an expression.
	// The toolkit resolves the prompt one time, so that value stays text.
	encoded, err := expressionstcl.EncodeBase64JSON(step.Agent)
	if err != nil {
		return nil, errors.Wrap(err, "agent: encoding error")
	}

	container.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(constants.DefaultToolkitPath, "agent").
		SetArgs("--base64", encoded).
		EnableToolkit(stage.Ref())

	return stage, nil
}

// operationsBeforeAgent lists the operations of the step that make a stage before the agent
// stage. The processor gives the ID of a step to its first stage only. The init process collects
// the outputs of the stage with the ID, so with one of these operations it loses the session output.
func operationsBeforeAgent(step testworkflowsv1.Step) []string {
	var before []string
	if step.Delay != "" {
		before = append(before, "delay")
	}
	// Files only mount volumes and make no stage.
	if step.Content != nil && step.Content.Git != nil {
		before = append(before, "content.git")
	}
	if step.Content != nil && len(step.Content.Tarball) > 0 {
		before = append(before, "content.tarball")
	}
	if step.Cache != nil {
		before = append(before, "cache")
	}
	if len(step.Services) > 0 {
		before = append(before, "services")
	}
	if len(step.Setup) > 0 {
		before = append(before, "setup")
	}
	if step.Run != nil {
		before = append(before, "run")
	}
	if step.Shell != "" {
		before = append(before, "shell")
	}
	if step.Execute != nil {
		before = append(before, "execute")
	}
	return before
}

func ProcessParallel(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Parallel == nil {
		return nil, nil
	}

	stage := stage.NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCategory("Run in parallel")

	// Allow to combine it within other containers
	stage.SetPure(true)

	// Inherit container defaults
	inherited := common.Ptr(stage.Container().ToContainerConfig())
	inherited.VolumeMounts = nil
	step.Parallel.Container = testworkflowresolver.MergeContainerConfig(inherited, step.Parallel.Container)

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(constants.DefaultToolkitPath, "parallel").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))

	// Parallel workers should inherit the resolved root pod config, while allowing
	// the parallel block to override any pod-level fields explicitly.
	parallel := step.Parallel.DeepCopy()
	pod := layer.PodConfig()
	parallel.Pod = testworkflowresolver.MergePodConfig(pod.DeepCopy(), parallel.Pod)

	// Base64 encode to prevent testworkflow-init from prematurely resolving expressions.
	// The parallel spec can contain expressions that need matrix/shard/count variables
	// which are only available during parallel execution, not in the init context.
	encoded, err := expressionstcl.EncodeBase64JSON(parallel)
	if err != nil {
		return nil, errors.Wrap(err, "parallel: encoding error")
	}
	stage.Container().SetArgs("--base64", encoded)

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

	// Allow to combine it within other containers
	stage.SetPure(true)

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(constants.DefaultToolkitPath, "services", "-g", "{{env.TK_SVC_REF}}").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))

	// Pass down image pull secrets
	services := make(map[string]testworkflowsv1.ServiceSpec)
	if pod := layer.PodConfig(); len(pod.ImagePullSecrets) > 0 {
		for name, svc := range step.Services {
			if svc.Pod == nil {
				svc.Pod = &testworkflowsv1.PodConfig{}
			} else {
				svc.Pod = svc.Pod.DeepCopy()
			}
			svc.Pod.ImagePullSecrets = append(svc.Pod.ImagePullSecrets, pod.ImagePullSecrets...)
			services[name] = svc
		}
	} else {
		services = step.Services
	}

	// Build arguments
	servicesMap := make(map[string]json.RawMessage)
	for name, svc := range services {
		v, err := json.Marshal(svc)
		if err != nil {
			return nil, errors.Wrapf(err, "services[%s]: marshalling error", name)
		}
		servicesMap[name] = json.RawMessage(v)
	}

	// Base64 encode to prevent testworkflow-init from prematurely resolving expressions.
	// Services can contain expressions like {{ matrix.browser.driver }} that need
	// to be evaluated in the services command context where matrix variables are available,
	// not in the init context where they would fail with "unknown variable" errors.
	encoded, err := expressionstcl.EncodeBase64JSON(servicesMap)
	if err != nil {
		return nil, errors.Wrap(err, "services: encoding error")
	}
	stage.Container().SetArgs("--base64", encoded)

	return stage, nil
}

func ProcessServicesStop(_ testworkflowprocessor.InternalProcessor, layer testworkflowprocessor.Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if len(step.Services) == 0 {
		return nil, nil
	}

	stage := stage.NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCondition("always") // TODO: actually, it's enough to do it when "Start services" is not skipped
	stage.SetCategory("Stop services")

	// Allow to combine it within other containers
	stage.SetPure(true)

	stage.Container().
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(constants.DefaultToolkitPath, "kill", "{{env.TK_SVC_REF}}").
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
