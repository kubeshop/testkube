// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func MapExecutorKubeToTestWorkflowTemplateKube(v executorv1.Executor) testworkflowsv1.TestWorkflowTemplate {
	var workingDir *string
	if v.Spec.UseDataDirAsWorkingDir {
		workingDir = common.Ptr("/data")
	}

	return testworkflowsv1.TestWorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        v.Name,
			Namespace:   v.Namespace,
			Labels:      v.Labels,
			Annotations: v.Annotations,
		},
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Container: &testworkflowsv1.ContainerConfig{
					WorkingDir: workingDir,
					Image:      v.Spec.Image,
					Args:       &v.Spec.Args,
					Command:    &v.Spec.Command,
				},
				Pod: &testworkflowsv1.PodConfig{
					ImagePullSecrets: v.Spec.ImagePullSecrets,
				},
			},
		},
	}
}
