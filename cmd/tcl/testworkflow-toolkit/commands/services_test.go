// Copyright 2025 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
// 	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestServicesExecutor_RequiresGroupRef(t *testing.T) {
	executor := NewServicesExecutor("", false, ServicesDependencies{})
	err := executor.Execute(context.Background(), []string{`{}`})
	assert.ErrorContains(t, err, "missing required --group")
}

func TestProcessNotifications(t *testing.T) {
	tests := []struct {
		name          string
		notifications []executionworkertypes.StatusNotification
		hasReadiness  bool
		want          ServiceExecutionResult
	}{
		{
			name: "detects failure when result arrives before service starts",
			notifications: []executionworkertypes.StatusNotification{
				{Result: finishedResult(testkube.FAILED_TestWorkflowStatus)},
			},
			want: ServiceExecutionResult{Started: false, Ready: true, Failed: true},
		},
		{
			name: "detects success when started without readiness probe",
			notifications: []executionworkertypes.StatusNotification{
				{PodIp: "10.0.0.1", Ref: "main"},
			},
			want: ServiceExecutionResult{Started: true, Ready: true, Failed: false},
		},
		{
			name: "detects success with readiness probe when ready",
			notifications: []executionworkertypes.StatusNotification{
				{PodIp: "10.0.0.1", Ref: "main"},
				{Ready: true},
			},
			hasReadiness: true,
			want:         ServiceExecutionResult{Started: true, Ready: true, Failed: false},
		},
		{
			name: "not started without IP",
			notifications: []executionworkertypes.StatusNotification{
				{Ref: "main"},
			},
			want: ServiceExecutionResult{Started: false, Ready: true, Failed: false},
		},
		{
			name: "breaks on start without readiness probe",
			notifications: []executionworkertypes.StatusNotification{
				{PodIp: "10.0.0.1", Ref: "main"},
				{Result: finishedResult(testkube.PASSED_TestWorkflowStatus)},
			},
			want: ServiceExecutionResult{Started: true, Ready: true, Failed: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newTestRunner(tt.hasReadiness)
			watcher := newMockWatcher(tt.notifications)

			got := runner.processNotifications(watcher, "main")
			assert.Equal(t, tt.want, ServiceExecutionResult{Started: got.Started, Ready: got.Ready, Failed: got.Failed})
		})
	}
}

func TestEvaluateResult(t *testing.T) {
	tests := []struct {
		name         string
		result       ServiceExecutionResult
		hasReadiness bool
		wantSuccess  bool
	}{
		{
			name:        "error takes priority",
			result:      ServiceExecutionResult{Started: true, Ready: true, Failed: false, Error: errors.New("err")},
			wantSuccess: false,
		},
		{
			name:        "failed takes priority over started",
			result:      ServiceExecutionResult{Started: true, Ready: true, Failed: true},
			wantSuccess: false,
		},
		{
			name:        "not started is failure",
			result:      ServiceExecutionResult{Started: false, Ready: true, Failed: false},
			wantSuccess: false,
		},
		{
			name:         "not ready with readiness probe is failure",
			result:       ServiceExecutionResult{Started: true, Ready: false, Failed: false},
			hasReadiness: true,
			wantSuccess:  false,
		},
		{
			name:         "not ready without readiness probe is success",
			result:       ServiceExecutionResult{Started: true, Ready: false, Failed: false},
			hasReadiness: false,
			wantSuccess:  true,
		},
		{
			name:        "started and ready is success",
			result:      ServiceExecutionResult{Started: true, Ready: true, Failed: false},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newTestRunner(tt.hasReadiness)

			got := runner.evaluateResult(tt.result)

			assert.Equal(t, tt.wantSuccess, got)
		})
	}
}

func TestCheckForImmediateFailure(t *testing.T) {
	tests := []struct {
		name          string
		notifications []executionworkertypes.StatusNotification
		inputResult   ServiceExecutionResult
		wantFailed    bool
	}{
		{
			name: "detects failure",
			notifications: []executionworkertypes.StatusNotification{
				{Result: finishedResult(testkube.FAILED_TestWorkflowStatus)},
			},
			inputResult: ServiceExecutionResult{Started: true},
			wantFailed:  true,
		},
		{
			name: "no failure on passed",
			notifications: []executionworkertypes.StatusNotification{
				{Result: finishedResult(testkube.PASSED_TestWorkflowStatus)},
			},
			inputResult: ServiceExecutionResult{Started: true},
			wantFailed:  false,
		},
		{
			name:          "no failure on timeout (healthy service)",
			notifications: []executionworkertypes.StatusNotification{},
			inputResult:   ServiceExecutionResult{Started: true},
			wantFailed:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := &mockWorker{notifications: tt.notifications}
			runner := &ServiceRunner{
				deps:     ServicesDependencies{ExecutionWorker: worker},
				instance: &ServiceInstance{Name: "test", Index: 0},
				state:    map[string][]ServiceState{"test": {{}}},
				info:     ServiceInfo{},
				log:      func(...string) {},
			}

			got := runner.checkForImmediateFailure(
				testworkflowconfig.InternalConfig{},
				&executionworkertypes.ServiceResult{},
				tt.inputResult,
			)

			assert.Equal(t, tt.wantFailed, got.Failed)
		})
	}
}

func newTestRunner(hasReadiness bool) *ServiceRunner {
	var probe *corev1.Probe
	if hasReadiness {
		probe = &corev1.Probe{}
	}
	return &ServiceRunner{
		instance: &ServiceInstance{Name: "test", Index: 0, ReadinessProbe: probe},
		state:    map[string][]ServiceState{"test": {{}}},
		info:     ServiceInfo{},
		deps:     ServicesDependencies{},
		log:      func(...string) {},
	}
}

func finishedResult(status testkube.TestWorkflowStatus) *testkube.TestWorkflowResult {
	return &testkube.TestWorkflowResult{
		Status:     common.Ptr(status),
		FinishedAt: time.Now(),
	}
}

type mockWatcher struct {
	ch  chan executionworkertypes.StatusNotification
	err error
}

func newMockWatcher(notifications []executionworkertypes.StatusNotification) *mockWatcher {
	ch := make(chan executionworkertypes.StatusNotification, len(notifications))
	for _, n := range notifications {
		ch <- n
	}
	close(ch)
	return &mockWatcher{ch: ch}
}

func (m *mockWatcher) Channel() <-chan executionworkertypes.StatusNotification { return m.ch }
func (m *mockWatcher) All() ([]executionworkertypes.StatusNotification, error) { return nil, nil }
func (m *mockWatcher) Err() error                                              { return m.err }

type mockWorker struct {
	executionworkertypes.Worker
	notifications []executionworkertypes.StatusNotification
}

func (m *mockWorker) StatusNotifications(ctx context.Context, id string, opts executionworkertypes.StatusNotificationsOptions) executionworkertypes.StatusNotificationsWatcher {
	return newMockWatcher(m.notifications)
}

func TestServiceEnvExpressionPreservation(t *testing.T) {
	const (
		envVar      = "TEST_VAR"
		parentValue = "parent-value"
	)

	cfg := &testworkflowconfig.InternalConfig{
		Execution: testworkflowconfig.ExecutionConfig{Id: "test-exec"},
	}
	shellScript := `echo "{{ env.` + envVar + ` }}"`

	t.Run("without EnvMachine expression preserved", func(t *testing.T) {
		executor := &ServicesExecutor{
			groupRef: "test-group",
			deps: ServicesDependencies{
				Config:      cfg,
				BaseMachine: createTestMachineWithoutEnv(cfg),
				Namespace:   "test-ns",
			},
		}
		instance := buildServiceInstance(t, executor, shellScript)
		assert.Contains(t, *instance.Spec.Steps[0].Run.Shell, "{{env."+envVar+"}}")
	})

	t.Run("with EnvMachine parent value leaks", func(t *testing.T) {
		machineWithEnv := expressions.CombinedMachines(
			createTestMachineWithoutEnv(cfg),
			expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
				if name == "env."+envVar {
					return parentValue, true
				}
				return nil, false
			}),
		)
		executor := &ServicesExecutor{
			groupRef: "test-group",
			deps: ServicesDependencies{
				Config:      cfg,
				BaseMachine: machineWithEnv,
				Namespace:   "test-ns",
			},
		}
		instance := buildServiceInstance(t, executor, shellScript)
		assert.Contains(t, *instance.Spec.Steps[0].Run.Shell, parentValue)
	})
}

func buildServiceInstance(t *testing.T, executor *ServicesExecutor, shellScript string) ServiceInstance {
	t.Helper()
	svc := testworkflowsv1.ServiceSpec{
		IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
			StepRun: testworkflowsv1.StepRun{Shell: &shellScript},
			Pod:     &testworkflowsv1.PodConfig{},
		},
	}
	params := &commontcl.ParamsSpec{Count: 1, ShardCount: 1, MatrixCount: 1}
	instances, _, err := executor.buildServiceInstances("test-service", svc, params)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	return instances[0]
}

func createTestMachineWithoutEnv(cfg *testworkflowconfig.InternalConfig) expressions.Machine {
	return expressions.CombinedMachines(
		testworkflowconfig.CreateExecutionMachine(&cfg.Execution),
		testworkflowconfig.CreateWorkflowMachine(&cfg.Workflow),
	)
}
