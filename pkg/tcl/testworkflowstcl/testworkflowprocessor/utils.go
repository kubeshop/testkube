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
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

func AnnotateControlledBy(obj metav1.Object, testWorkflowId string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[ExecutionIdLabelName] = testWorkflowId
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateControlledBy(&v.Spec.Template, testWorkflowId)
	}
}

func getRef(stage Stage) string {
	return stage.Ref()
}

func isNotOptional(stage Stage) bool {
	return !stage.Optional()
}

func buildKubernetesContainers(stage Stage, init *initProcess, machines ...expressionstcl.Machine) (containers []corev1.Container, err error) {
	if stage.Timeout() != "" {
		init.AddTimeout(stage.Timeout(), stage.Ref())
	}
	if stage.Ref() != "" {
		init.AddCondition(stage.Condition(), stage.Ref())
	}
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
			sub, serr := buildKubernetesContainers(ch, init.Children(ch.Ref()), machines...)
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
	err = c.Container().Detach().Resolve(machines...)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: resolving container: %s", stage.Ref(), stage.Name(), err.Error())
	}

	cr, err := c.Container().ToKubernetesTemplate()
	if err != nil {
		return nil, fmt.Errorf("%s: %s: building container template: %s", stage.Ref(), stage.Name(), err.Error())
	}
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

	// Ensure the container will have proper access to FS
	if cr.SecurityContext == nil {
		cr.SecurityContext = &corev1.SecurityContext{}
	}
	if cr.SecurityContext.RunAsGroup == nil {
		cr.SecurityContext.RunAsGroup = common.Ptr(defaultFsGroup)
	}

	containers = []corev1.Container{cr}
	return
}
