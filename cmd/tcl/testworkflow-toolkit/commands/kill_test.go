// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func terminated(reason string) *corev1.ContainerStateTerminated {
	return &corev1.ContainerStateTerminated{Reason: reason}
}

func TestGetServiceHealth(t *testing.T) {
	tests := []struct {
		name   string
		status corev1.PodStatus
		want   []string
	}{
		{
			name: "a healthy pod has no issues",
			status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{
				{Name: "2", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			}},
		},
		{
			name: "a restart after an out-of-memory kill names each reason one time",
			status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:                 "2",
					LastTerminationState: corev1.ContainerState{Terminated: terminated("OOMKilled")},
					State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
				},
			}},
			want: []string{"OOMKilled", "CrashLoopBackOff"},
		},
		{
			name: "the same reason before and after a restart shows one time",
			status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:                 "2",
					LastTerminationState: corev1.ContainerState{Terminated: terminated("OOMKilled")},
					State:                corev1.ContainerState{Terminated: terminated("OOMKilled")},
				},
			}},
			want: []string{"OOMKilled"},
		},
		{
			name: "a failed pod without a reason gets words",
			status: corev1.PodStatus{Phase: corev1.PodFailed, ContainerStatuses: []corev1.ContainerStatus{
				{Name: "2", State: corev1.ContainerState{Terminated: terminated("Error")}},
			}},
			want: []string{"Error", "the pod failed"},
		},
		{
			name: "a container that completed is healthy",
			status: corev1.PodStatus{Phase: corev1.PodSucceeded, ContainerStatuses: []corev1.ContainerStatus{
				{Name: "2", State: corev1.ContainerState{Terminated: terminated("Completed")}},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientSet := fake.NewClientset(&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "svc-pod", Namespace: "ns", Labels: map[string]string{"testkube.io/resource": "exec-ref-svc-0"}},
				Status:     tt.status,
			})

			assert.Equal(t, tt.want, getServiceHealth(context.Background(), clientSet, "ns", "exec-ref-svc-0"))
		})
	}
}

func TestUnhealthyServicesError(t *testing.T) {
	type instance struct {
		name   string
		issues []string
	}
	tests := []struct {
		name          string
		instances     []instance
		want          string
		wantOOMKilled bool
	}{
		{
			name:          "one service killed for its memory",
			instances:     []instance{{"db", []string{"OOMKilled", "CrashLoopBackOff"}}},
			want:          `The service "db" is not healthy: OOMKilled, CrashLoopBackOff.`,
			wantOOMKilled: true,
		},
		{
			name:      "each instance gets its own sentence",
			instances: []instance{{"api/1", []string{"Error", "the pod failed"}}, {"api/2", []string{"Error"}}},
			want:      `The service "api/1" is not healthy: Error, the pod failed. The service "api/2" is not healthy: Error.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &UnhealthyServicesError{}
			for _, i := range tt.instances {
				err.add(i.name, i.issues)
			}

			assert.Equal(t, tt.want, err.Error())
			assert.Equal(t, tt.wantOOMKilled, err.OOMKilled)
		})
	}
}
