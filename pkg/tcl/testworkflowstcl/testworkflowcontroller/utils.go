// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"regexp"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	KubernetesLogTimeFormat = "2006-01-02T15:04:05.000000000Z"
)

func GetEventContainerName(event *corev1.Event) string {
	regex := regexp.MustCompile(`^spec\.(?:initContainers|containers)\{([^]]+)}`)
	path := event.InvolvedObject.FieldPath
	if regex.Match([]byte(path)) {
		name := regex.ReplaceAllString(event.InvolvedObject.FieldPath, "$1")
		return name
	}
	return ""
}

func IsPodDone(pod *corev1.Pod) bool {
	return (pod.Status.Phase != corev1.PodPending && pod.Status.Phase != corev1.PodRunning) || pod.ObjectMeta.DeletionTimestamp != nil
}

func IsJobDone(job *batchv1.Job) bool {
	return (job.Status.Active == 0 && (job.Status.Succeeded > 0 || job.Status.Failed > 0)) || job.ObjectMeta.DeletionTimestamp != nil
}

type ContainerResult struct {
	Status     testkube.TestWorkflowStepStatus
	Details    string
	ExitCode   int
	FinishedAt time.Time
}

var UnknownContainerResult = ContainerResult{
	Status:   testkube.ABORTED_TestWorkflowStepStatus,
	ExitCode: -1,
}

func GetContainerResult(c corev1.ContainerStatus) ContainerResult {
	if c.State.Waiting != nil {
		return ContainerResult{Status: testkube.QUEUED_TestWorkflowStepStatus, ExitCode: -1}
	}
	if c.State.Running != nil {
		return ContainerResult{Status: testkube.RUNNING_TestWorkflowStepStatus, ExitCode: -1}
	}
	re := regexp.MustCompile(`^([^,]*),(0|[1-9]\d*)$`)

	// Workaround - GKE sends SIGKILL after the container is already terminated,
	// and the pod gets stuck then.
	if c.State.Terminated.Reason != "Completed" {
		return ContainerResult{Status: testkube.ABORTED_TestWorkflowStepStatus, Details: c.State.Terminated.Reason, ExitCode: -1, FinishedAt: c.State.Terminated.FinishedAt.Time}
	}

	msg := c.State.Terminated.Message
	match := re.FindStringSubmatch(msg)
	if match == nil {
		return ContainerResult{Status: testkube.ABORTED_TestWorkflowStepStatus, ExitCode: -1, FinishedAt: c.State.Terminated.FinishedAt.Time}
	}
	status := testkube.TestWorkflowStepStatus(match[1])
	exitCode, _ := strconv.Atoi(match[2])
	if status == "" {
		status = testkube.PASSED_TestWorkflowStepStatus
	}
	return ContainerResult{Status: status, ExitCode: exitCode, FinishedAt: c.State.Terminated.FinishedAt.Time}
}
