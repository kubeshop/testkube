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
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const testResourceID = "test-exec-db-0"

var podsResource = schema.GroupResource{Resource: "pods"}

// serviceUnavailable is the error a DELETE produces when the API server answers HTTP 503, which
// is what the reported incident carried.
func serviceUnavailable() error {
	return apierrors.NewGenericServerResponse(503, "delete", podsResource, "", "", 0, true)
}

func forbidden() error {
	return apierrors.NewForbidden(podsResource, "", errors.New("cannot deletecollection resource \"pods\""))
}

// shortenCleanupRetryDelay keeps the retry-exhaustion cases fast.
func shortenCleanupRetryDelay(t *testing.T) {
	t.Helper()
	previous := cleanupRetryDelay
	cleanupRetryDelay = time.Millisecond
	t.Cleanup(func() { cleanupRetryDelay = previous })
}

// captureStdout collects what the command prints, because the warnings are plain fmt.Printf.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	previous := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = previous }()

	fn()

	require.NoError(t, writer.Close())
	out, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(out)
}

func oneService() []executionworkertypes.ListResultItem {
	return []executionworkertypes.ListResultItem{{
		Resource:  testworkflowconfig.ResourceConfig{Id: testResourceID},
		Namespace: "testkube",
	}}
}

// healthyClientSet holds a service pod with no termination or waiting problems.
func healthyClientSet() kubernetes.Interface {
	return k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-pod",
			Namespace: "testkube",
			Labels:    map[string]string{"testkube.io/resource": testResourceID},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "main",
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
		},
	})
}

// oomKilledClientSet holds a service pod whose container the kernel killed, which TKC-5014 made
// a step failure.
func oomKilledClientSet() kubernetes.Interface {
	return k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-pod",
			Namespace: "testkube",
			Labels:    map[string]string{"testkube.io/resource": testResourceID},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					Reason: "OOMKilled",
				}},
			}},
		},
	})
}

func TestIsTransientCleanupError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"service unavailable 503", serviceUnavailable(), true},
		{"too many requests 429", apierrors.NewTooManyRequests("slow down", 1), true},
		{"server timeout", apierrors.NewServerTimeout(podsResource, "delete", 1), true},
		{"request timeout", apierrors.NewTimeoutError("timed out", 1), true},
		{"bare network error", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, true},
		{"forbidden 403", forbidden(), false},
		{"not found 404", apierrors.NewNotFound(podsResource, "svc-pod"), false},
		{"bad request 400", apierrors.NewBadRequest("malformed"), false},
		// Dropped from the transient set deliberately: a 409 cannot realistically come from a
		// label-selector DeleteCollection, and a 500 is usually a failing admission webhook,
		// which is persistent and must stay loud.
		{"conflict 409", apierrors.NewConflict(podsResource, "svc-pod", errors.New("conflict")), false},
		{"internal error 500", apierrors.NewInternalError(errors.New("webhook refused")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isTransientCleanupError(tt.err))
		})
	}
}

// The group cleanup joins the errors of its concurrent deletes, so the verdict must not depend on
// which of them happened to finish first.
func TestIsTransientCleanupErrorOnAggregates(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"single transient branch", errors.Join(serviceUnavailable()), true},
		{"every branch transient", errors.Join(serviceUnavailable(), serviceUnavailable()), true},
		{"permanent then transient", errors.Join(forbidden(), serviceUnavailable()), true},
		{"transient then permanent", errors.Join(serviceUnavailable(), forbidden()), true},
		{"no transient branch", errors.Join(forbidden(), apierrors.NewNotFound(podsResource, "svc-pod")), false},
		{"nested aggregate", errors.Join(errors.Join(forbidden()), errors.Join(serviceUnavailable())), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isTransientCleanupError(tt.err))
		})
	}
}

func TestRetryTransient(t *testing.T) {
	t.Run("stops as soon as it succeeds", func(t *testing.T) {
		attempts := 0
		err := retryTransient(context.Background(), cleanupRetryCount, time.Millisecond, func() error {
			attempts++
			if attempts < 2 {
				return serviceUnavailable()
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("gives up after the full budget", func(t *testing.T) {
		attempts := 0
		err := retryTransient(context.Background(), cleanupRetryCount, time.Millisecond, func() error {
			attempts++
			return serviceUnavailable()
		})
		require.Error(t, err)
		assert.True(t, apierrors.IsServiceUnavailable(err))
		assert.Equal(t, cleanupRetryCount, attempts)
	})

	t.Run("does not retry a permanent error", func(t *testing.T) {
		attempts := 0
		err := retryTransient(context.Background(), cleanupRetryCount, time.Millisecond, func() error {
			attempts++
			return forbidden()
		})
		require.Error(t, err)
		assert.True(t, apierrors.IsForbidden(err))
		assert.Equal(t, 1, attempts)
	})

	t.Run("returns promptly when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		err := retryTransient(ctx, cleanupRetryCount, time.Hour, func() error {
			attempts++
			cancel()
			return serviceUnavailable()
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, attempts, "must not sleep through the remaining budget")
	})
}

func TestRunKillTeardown(t *testing.T) {
	t.Run("passes when teardown succeeds", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).Return(nil).Times(1)

		var err error
		out := captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		assert.NoError(t, err)
		assert.NotContains(t, out, "warning: cleaning up resources")
	})

	t.Run("recovers when a transient error clears", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

		attempts := 0
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
			DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
				attempts++
				if attempts < 3 {
					return serviceUnavailable()
				}
				return nil
			}).AnyTimes()

		var err error
		out := captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
		assert.NotContains(t, out, "warning: cleaning up resources")
	})

	// The reported regression: the steps passed, so an exhausted transient teardown must not
	// take the run down with it.
	t.Run("passes with a warning when a transient error exhausts the budget", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

		attempts := 0
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
			DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
				attempts++
				return serviceUnavailable()
			}).AnyTimes()

		var err error
		out := captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		assert.NoError(t, err)
		assert.Equal(t, cleanupRetryCount, attempts)
		assert.Contains(t, out, "warning: cleaning up resources")
		assert.Contains(t, out, "the server is currently unable to handle the request")
	})

	t.Run("fails fast on a permanent teardown error", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

		attempts := 0
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
			DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
				attempts++
				return forbidden()
			}).AnyTimes()

		var err error
		_ = captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cleaning up resources")
		assert.Equal(t, 1, attempts, "a permanent error must not consume the retry budget")
	})

	t.Run("treats a mixed aggregate as transient", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)

		attempts := 0
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
			DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
				attempts++
				return errors.Join(forbidden(), serviceUnavailable())
			}).AnyTimes()

		var err error
		out := captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		assert.NoError(t, err)
		assert.Equal(t, cleanupRetryCount, attempts)
		assert.Contains(t, out, "warning: cleaning up resources")
	})
}

func TestRunKillHealth(t *testing.T) {
	t.Run("fails when a service is unhealthy", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).Return(nil).Times(1)

		var err error
		_ = captureStdout(t, func() {
			err = RunKill(context.Background(), worker, oomKilledClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unhealthy services detected")
		assert.Contains(t, err.Error(), "OOMKilled")
	})

	// The teardown downgrade must not swallow a health failure.
	t.Run("reports the health failure even when teardown also exhausts", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).Return(oneService(), nil).Times(1)
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).
			Return(serviceUnavailable()).AnyTimes()

		var err error
		out := captureStdout(t, func() {
			err = RunKill(context.Background(), worker, oomKilledClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unhealthy services detected")
		assert.Contains(t, out, "warning: cleaning up resources")
	})
}

func TestRunKillList(t *testing.T) {
	t.Run("recovers when a transient list error clears", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)

		attempts := 0
		worker.EXPECT().List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, executionworkertypes.ListOptions) ([]executionworkertypes.ListResultItem, error) {
				attempts++
				if attempts < 3 {
					return nil, serviceUnavailable()
				}
				return oneService(), nil
			}).AnyTimes()
		worker.EXPECT().DestroyGroup(gomock.Any(), "svc-group", gomock.Any()).Return(nil).Times(1)

		var err error
		_ = captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	// A failed list leaves the namespace set and the per-service instructions unknown, so the
	// step cannot do its job at all and stays fatal.
	t.Run("fails when the list never succeeds", func(t *testing.T) {
		shortenCleanupRetryDelay(t)
		ctrl := gomock.NewController(t)
		worker := executionworkertypes.NewMockWorker(ctrl)
		worker.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(nil, serviceUnavailable()).Times(cleanupRetryCount)

		var err error
		_ = captureStdout(t, func() {
			err = RunKill(context.Background(), worker, healthyClientSet(), "testkube", "ref", "svc-group", nil, nil)
		})

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "listing service instances"))
	})
}
