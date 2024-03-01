// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowresolver"
)

type Scope interface {
	Resources() Resources
	RootStage() GroupStage
	ContainerDefaults() Container

	AppendJobConfig(cfg *testworkflowsv1.JobConfig) Scope
	AppendPodConfig(cfg *testworkflowsv1.PodConfig) Scope

	Job(inspector imageinspector.Inspector) (batchv1.Job, error)
	Signature() []Signature
}

type scope struct {
	RootStageValue GroupStage                `expr:"include"`
	ResourcesValue Resources                 `expr:"include"`
	ContainerValue Container                 `expr:"include"`
	JobConfigValue testworkflowsv1.JobConfig `expr:"include"`
	PodConfigValue testworkflowsv1.PodConfig `expr:"include"`
}

func NewScope() Scope {
	return &scope{
		RootStageValue: NewGroupStage(""),
		ResourcesValue: NewResources(),
		ContainerValue: NewContainer(),
	}
}

func (s *scope) Resources() Resources {
	return s.ResourcesValue
}

func (s *scope) ContainerDefaults() Container {
	return s.ContainerValue
}

func (s *scope) AppendJobConfig(cfg *testworkflowsv1.JobConfig) Scope {
	s.JobConfigValue = *testworkflowresolver.MergeJobConfig(&s.JobConfigValue, cfg)
	return s
}

func (s *scope) AppendPodConfig(cfg *testworkflowsv1.PodConfig) Scope {
	s.PodConfigValue = *testworkflowresolver.MergePodConfig(&s.PodConfigValue, cfg)
	return s
}

func (s *scope) RootStage() GroupStage {
	return s.RootStageValue
}

func (s *scope) Signature() []Signature {
	return s.RootStageValue.Signature().Children()
}

func getRef(stage Stage) string {
	return stage.Ref()
}

func isNotOptional(stage Stage) bool {
	return !stage.Optional()
}

func initializeContainers(r Resources, stage Stage, init *initProcess, images map[string]*imageinspector.Info) (containers []corev1.Container, err error) {
	if stage.Timeout() != "" {
		init.AddTimeout(stage.Timeout(), stage.Ref())
	}

	init.AddCondition(stage.Condition(), stage.Ref())
	init.AddRetryPolicy(stage.RetryPolicy(), stage.Ref())

	group, ok := stage.(GroupStage)
	if ok {
		recursiveRefs := common.MapSlice(group.RecursiveChildren(), getRef)
		directRefResults := common.MapSlice(common.FilterSlice(group.Children(), isNotOptional), getRef)

		init.AddCondition(stage.Condition(), recursiveRefs...)

		if group.Negative() {
			// Create virtual layer that will be put down into actual negative step
			init.SetRef(stage.Ref() + ".v")
			init.AddCondition(stage.Condition(), stage.Ref()+".v")
			init.PrependInitialStatus(stage.Ref() + ".v")
			init.AddResult("!"+stage.Ref()+".v", stage.Ref())
		} else if stage.Ref() != "" {
			init.PrependInitialStatus(stage.Ref())
		}

		if group.Optional() {
			init.ResetResults()
		}

		if group.Negative() {
			init.AddResult(strings.Join(directRefResults, "&&"), ""+stage.Ref()+".v")
		} else {
			init.AddResult(strings.Join(directRefResults, "&&"), ""+stage.Ref())
		}

		for i, ch := range group.Children() {
			// Condition should be executed only in the first leaf
			if i == 1 {
				init.ResetCondition()
			}
			// Pass down to another group or container
			sub, serr := initializeContainers(r, ch, init.Children(ch.Ref()), images)
			if serr != nil {
				return nil, fmt.Errorf("%s: %s: resolving children: %s", stage.Ref(), stage.Name(), serr.Error())
			}
			containers = append(containers, sub...)
		}
		return
	}
	c, ok := stage.(ContainerStage)
	if !ok {
		return nil, fmt.Errorf("%s: %s: stage that is neither container nor group", stage.Ref(), stage.Name())
	}
	err = c.Container().Detach().Resolve()
	if err != nil {
		return nil, fmt.Errorf("%s: %s: resolving container: %s", stage.Ref(), stage.Name(), err.Error())
	}

	// Provide default image data
	image := images[c.Container().Image()]
	init.SetImageData(image)
	if image != nil {
		_ = c.Resolve(expressionstcl.NewMachine().
			Register("image.command", image.Entrypoint).
			Register("image.args", image.Cmd).
			Register("image.workingDir", image.WorkingDir))
	}

	cr := c.Container().ToKubernetesTemplate()
	cr.Name = c.Ref()

	if c.Optional() {
		init.ResetResults()
	}

	init.
		SetNegative(c.Negative()).
		AddRetryPolicy(c.RetryPolicy(), c.Ref()).
		SetCommand(cr.Command...).
		SetArgs(cr.Args...)

	for _, env := range cr.Env {
		if strings.Contains(env.Value, "{{") {
			init.AddComputedEnvs(env.Name)
		}
	}

	if init.Error() != nil {
		return nil, init.Error()
	}

	cr.Command = init.Command()
	cr.Args = init.Args()

	containers = []corev1.Container{cr}
	return
}

func (s *scope) Job(inspector imageinspector.Inspector) (batchv1.Job, error) {
	root := s.RootStage()

	ctx := context.Background() // TODO: Get from outside
	registry := ""              // TODO: Get from outside
	pullSecretNames := make([]string, len(s.PodConfigValue.ImagePullSecrets))
	for i, v := range s.PodConfigValue.ImagePullSecrets {
		pullSecretNames[i] = v.Name
	}

	// Initialize map of image data
	imageNames := root.GetImages()
	images := make(map[string]*imageinspector.Info)
	for image := range imageNames {
		info, err := inspector.Inspect(ctx, registry, image, corev1.PullIfNotPresent, pullSecretNames)
		if err != nil {
			return batchv1.Job{}, fmt.Errorf("resolving image error: %s: %s", image, err.Error())
		}
		images[image] = info
	}

	// Build list of containers
	containers, err := initializeContainers(s.Resources(), root, NewInitProcess().SetRef(root.Ref()), images)
	if err != nil {
		return batchv1.Job{}, err
	}

	// Resolve static data in the containers
	_ = expressionstcl.SimplifyForce(containers)

	// Append the TestWorkflow signature
	v, _ := json.Marshal(s.RootStage().Signature().Children())
	annotations := map[string]string{
		"testworkflows.testkube.io/signature": string(v),
		// TODO: to all resources, probably as LABEL
		//"testworkflows.testkube.io/controlled-by": "{{execution.id}}",
	}
	maps.Copy(annotations, s.JobConfigValue.Annotations)

	initContainer := corev1.Container{
		// TODO: Resources, SecurityContext?
		Name:            "copy-init",
		Image:           defaultInitImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
		VolumeMounts:    s.ContainerDefaults().VolumeMounts(),
	}

	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "{{execution.id}}",
			Annotations: annotations,
			Labels:      s.JobConfigValue.Labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Annotations: s.PodConfigValue.Annotations,
					Labels:      s.PodConfigValue.Labels,
				},
				Spec: corev1.PodSpec{
					InitContainers:     append([]corev1.Container{initContainer}, containers[:len(containers)-1]...),
					Containers:         containers[len(containers)-1:],
					RestartPolicy:      corev1.RestartPolicyNever,
					Volumes:            s.Resources().Volumes(),
					ImagePullSecrets:   s.PodConfigValue.ImagePullSecrets,
					ServiceAccountName: s.PodConfigValue.ServiceAccountName,
					NodeSelector:       s.PodConfigValue.NodeSelector,
				},
			},
		},
	}

	return job, nil
}
