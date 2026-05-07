package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var scheduleCmpOpts = []cmp.Option{
	cmpopts.IgnoreUnexported(
		cloud.ScheduleRequest{},
		cloud.RunningContext{},
		cloud.ScheduleExecution{},
		cloud.ScheduleResourceSelector{},
		cloud.ExecutionTarget{},
		cloud.ExecutionTargetLabels{},
	),
}

type fakeTestWorkflowExecutor struct {
	req *cloud.ScheduleRequest
	err error
}

func (f *fakeTestWorkflowExecutor) Execute(_ context.Context, request *cloud.ScheduleRequest) ([]testkubev1.TestWorkflowExecution, error) {
	f.req = request
	return []testkubev1.TestWorkflowExecution{}, f.err
}

func TestWorkflowExecutionExecutorController(t *testing.T) {
	tests := map[string]struct {
		objs      []client.Object
		request   reconcile.Request
		expect    *cloud.ScheduleRequest
		execErr   error
		wantError string
		wantEvent string
	}{
		// Should pass through a basic test workflow execution.
		"execute": {
			objs: []client.Object{
				&testworkflowsv1.TestWorkflowExecution{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TestWorkflowExecution",
						APIVersion: "testworkflows.testkube.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-workflow-execution",
						Namespace:  "test-namespace",
						Generation: 1,
					},
					Spec: testworkflowsv1.TestWorkflowExecutionSpec{
						TestWorkflow: &v1.LocalObjectReference{Name: "test-workflow"},
					},
					Status: testworkflowsv1.TestWorkflowExecutionStatus{},
				},
			},
			request: reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test-workflow-execution",
					Namespace: "test-namespace",
				},
			},
			expect: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{{
					Selector: &cloud.ScheduleResourceSelector{Name: "test-workflow"},
				}},
				RunningContext: &cloud.RunningContext{
					Name: "test-workflow-execution",
					Type: cloud.RunningContextType_KUBERNETESOBJECT,
				},
			},
			wantEvent: "Normal ExecutionScheduled Scheduled test workflow \"test-workflow\"",
		},
		// Should pass through a basic test workflow execution with target selectors.
		"target": {
			objs: []client.Object{
				&testworkflowsv1.TestWorkflowExecution{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TestWorkflowExecution",
						APIVersion: "testworkflows.testkube.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-workflow-execution",
						Namespace:  "test-namespace",
						Generation: 1,
					},
					Spec: testworkflowsv1.TestWorkflowExecutionSpec{
						TestWorkflow: &v1.LocalObjectReference{Name: "test-workflow"},
						ExecutionRequest: &testworkflowsv1.TestWorkflowExecutionRequest{
							Target: &commonv1.Target{
								Match: map[string][]string{
									"foo": {"bar"},
									"baz": {"qux", "quux"},
								},
								Not: map[string][]string{
									"one":   {"two"},
									"three": {"four", "five"},
								},
							},
						},
					},
					Status: testworkflowsv1.TestWorkflowExecutionStatus{},
				},
			},
			request: reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test-workflow-execution",
					Namespace: "test-namespace",
				},
			},
			expect: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{{
					Selector: &cloud.ScheduleResourceSelector{Name: "test-workflow"},
					Targets: []*cloud.ExecutionTarget{{
						Match: map[string]*cloud.ExecutionTargetLabels{
							"foo": {Labels: []string{"bar"}},
							"baz": {Labels: []string{"qux", "quux"}},
						},
						Not: map[string]*cloud.ExecutionTargetLabels{
							"one":   {Labels: []string{"two"}},
							"three": {Labels: []string{"four", "five"}},
						},
					}},
				}},
				RunningContext: &cloud.RunningContext{
					Name: "test-workflow-execution",
					Type: cloud.RunningContextType_KUBERNETESOBJECT,
				},
			},
			wantEvent: "Normal ExecutionScheduled Scheduled test workflow \"test-workflow\"",
		},
		// Should not execute if the generation has not changed.
		"ignore generation": {
			objs: []client.Object{
				&testworkflowsv1.TestWorkflowExecution{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TestWorkflowExecution",
						APIVersion: "testworkflows.testkube.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-workflow-execution",
						Namespace:  "test-namespace",
						Generation: 1,
					},
					Spec: testworkflowsv1.TestWorkflowExecutionSpec{
						TestWorkflow: &v1.LocalObjectReference{Name: "test-workflow"},
						ExecutionRequest: &testworkflowsv1.TestWorkflowExecutionRequest{
							Target: &commonv1.Target{
								Match: map[string][]string{
									"foo": {"bar"},
									"baz": {"qux", "quux"},
								},
								Not: map[string][]string{
									"one":   {"two"},
									"three": {"four", "five"},
								},
							},
						},
					},
					Status: testworkflowsv1.TestWorkflowExecutionStatus{
						Generation: 1,
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test-workflow-execution",
					Namespace: "test-namespace",
				},
			},
			expect: nil,
		},
		// Should update status with error and emit a Warning event on execution failure.
		"execution error updates status": {
			objs: []client.Object{
				&testworkflowsv1.TestWorkflowExecution{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TestWorkflowExecution",
						APIVersion: "testworkflows.testkube.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-workflow-execution",
						Namespace:  "test-namespace",
						Generation: 1,
					},
					Spec: testworkflowsv1.TestWorkflowExecutionSpec{
						TestWorkflow: &v1.LocalObjectReference{Name: "test-workflow"},
					},
					Status: testworkflowsv1.TestWorkflowExecutionStatus{},
				},
			},
			request: reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test-workflow-execution",
					Namespace: "test-namespace",
				},
			},
			expect: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{{
					Selector: &cloud.ScheduleResourceSelector{Name: "test-workflow"},
				}},
				RunningContext: &cloud.RunningContext{
					Name: "test-workflow-execution",
					Type: cloud.RunningContextType_KUBERNETESOBJECT,
				},
			},
			execErr:   fmt.Errorf("dial tcp: lookup testkube-api-server on 192.168.0.1:53: no such host"),
			wantError: "dial tcp: lookup testkube-api-server on 192.168.0.1:53: no such host",
			wantEvent: "Warning ExecutionNotScheduled dial tcp: lookup testkube-api-server on 192.168.0.1:53: no such host",
		},
		// Should clear a previous error on successful execution.
		"success clears previous error": {
			objs: []client.Object{
				&testworkflowsv1.TestWorkflowExecution{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TestWorkflowExecution",
						APIVersion: "testworkflows.testkube.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-workflow-execution",
						Namespace:  "test-namespace",
						Generation: 2,
					},
					Spec: testworkflowsv1.TestWorkflowExecutionSpec{
						TestWorkflow: &v1.LocalObjectReference{Name: "test-workflow"},
					},
					Status: testworkflowsv1.TestWorkflowExecutionStatus{
						Generation: 1,
						Error:      "previous error from failed attempt",
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      "test-workflow-execution",
					Namespace: "test-namespace",
				},
			},
			expect: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{{
					Selector: &cloud.ScheduleResourceSelector{Name: "test-workflow"},
				}},
				RunningContext: &cloud.RunningContext{
					Name: "test-workflow-execution",
					Type: cloud.RunningContextType_KUBERNETESOBJECT,
				},
			},
			wantEvent: "Normal ExecutionScheduled Scheduled test workflow \"test-workflow\"",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &fakeTestWorkflowExecutor{err: test.execErr}
			scheme := runtime.NewScheme()
			if err := testworkflowsv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.objs...).WithStatusSubresource(&testworkflowsv1.TestWorkflowExecution{}).Build()
			recorder := record.NewFakeRecorder(10)
			reconciler := testWorkflowExecutionExecutor(k8sClient, recorder, exec)

			_, err := reconciler.Reconcile(context.Background(), test.request)
			if err != nil {
				t.Errorf("reconcile: %v", err)
			}
			if !cmp.Equal(test.expect, exec.req, scheduleCmpOpts...) {
				t.Errorf("Incorrect execution request, diff: %s", cmp.Diff(test.expect, exec.req, scheduleCmpOpts...))
			}

			// After a successful execution, verify that Status.Generation was updated
			// to match Generation so the deduplication guard prevents re-execution,
			// and that any previous error has been cleared.
			if exec.req != nil && test.wantError == "" {
				var twe testworkflowsv1.TestWorkflowExecution
				if err := k8sClient.Get(context.Background(), test.request.NamespacedName, &twe); err != nil {
					t.Fatalf("get TestWorkflowExecution: %v", err)
				}
				if twe.Status.Generation != twe.Generation {
					t.Errorf("Status.Generation = %d, want %d", twe.Status.Generation, twe.Generation)
				}
				if twe.Status.Error != "" {
					t.Errorf("Status.Error = %q, want empty", twe.Status.Error)
				}
			}

			// On error, verify status was updated with the error message and generation.
			if test.wantError != "" {
				var twe testworkflowsv1.TestWorkflowExecution
				if err := k8sClient.Get(context.Background(), test.request.NamespacedName, &twe); err != nil {
					t.Fatalf("get TestWorkflowExecution: %v", err)
				}
				if twe.Status.Error != test.wantError {
					t.Errorf("Status.Error = %q, want %q", twe.Status.Error, test.wantError)
				}
				if twe.Status.Generation != twe.Generation {
					t.Errorf("Status.Generation = %d, want %d", twe.Status.Generation, twe.Generation)
				}
			}

			// Verify the expected event was emitted.
			if test.wantEvent != "" {
				select {
				case got := <-recorder.Events:
					if got != test.wantEvent {
						t.Errorf("event = %q, want %q", got, test.wantEvent)
					}
				case <-time.After(time.Second):
					t.Error("expected an event to be recorded, but none was")
				}
			}
		})
	}

}
