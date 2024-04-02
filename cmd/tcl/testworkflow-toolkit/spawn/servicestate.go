// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

type ServiceState struct {
	Name             string     `json:"name"`
	Host             string     `json:"host"`
	Ip               string     `json:"ip"`
	Started          bool       `json:"started"`
	ContainerStarted bool       `json:"containerStarted"`
	Ready            bool       `json:"ready"`
	Deleted          bool       `json:"deleted"`
	Success          bool       `json:"success"`
	Failed           bool       `json:"failed"`
	Finished         bool       `json:"finished"`
	Pod              corev1.Pod `json:"pod"`
}

func (s *ServiceState) Update(pod *corev1.Pod) {
	if pod == nil {
		return
	}
	s.Pod = *pod.DeepCopy()

	// Clean up huge unnecessary data
	s.Pod.ManagedFields = nil

	// Set basics
	s.Name = pod.Name
	s.Host = fmt.Sprintf("%s.%s.%s.svc.cluster.local", pod.Spec.Hostname, pod.Spec.Subdomain, pod.Namespace)

	// Compute data
	s.Started = pod.Status.StartTime != nil
	s.Deleted = pod.DeletionTimestamp != nil
	s.Success = pod.Status.Phase == "Succeeded"
	s.Failed = pod.Status.Phase == "Failed"
	s.Finished = s.Deleted || s.Success || s.Failed
	s.Ip = pod.Status.PodIP
	for _, c := range pod.Status.ContainerStatuses {
		if c.State.Running != nil || c.State.Terminated != nil {
			s.ContainerStarted = true
		}
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			s.Ready = true
		}
	}
}

func (s *ServiceState) Machine(index int64) expressionstcl.Machine {
	return expressionstcl.NewMachine().
		Register("started", s.Started).
		Register("containerStarted", s.ContainerStarted).
		Register("deleted", s.Deleted).
		Register("success", s.Success).
		Register("failed", s.Failed).
		Register("finished", s.Finished).
		Register("ready", s.Ready).
		Register("ip", s.Ip).
		Register("host", s.Host).
		Register("pod", s.Pod).
		Register("index", index)
}
