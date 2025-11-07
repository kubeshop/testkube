package controller

import (
	"context"
	"errors"
	"testing"

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
	req error
	out *cloud.ScheduleRequest
}

func (f *fakeTestWorkflowExecutor) Execute(_ context.Context, request *cloud.ScheduleRequest) ([]testkubev1.TestWorkflowExecution, error) {
	f.out = request
	return []testkubev1.TestWorkflowExecution{}, f.req
}

func TestWorkflowExecutionExecutorController(t *testing.T) {
	tests := map[string]struct {
		objs    []client.Object
		request reconcile.Request
		expect  *cloud.ScheduleRequest
		execErr error
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &fakeTestWorkflowExecutor{req: test.execErr}
			scheme := runtime.NewScheme()
			if err := testworkflowsv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.objs...).WithStatusSubresource(test.objs...).Build()
			recorder := record.NewFakeRecorder(10)
			reconciler := testWorkflowExecutionExecutor(k8sClient, recorder, exec)

			_, err := reconciler.Reconcile(context.Background(), test.request)
			if err != nil && test.execErr == nil {
				t.Errorf("reconcile: %v", err)
			}
			if !cmp.Equal(test.expect, exec.out, scheduleCmpOpts...) {
				t.Errorf("Incorrect execution request, diff: %s", cmp.Diff(test.expect, exec.out, scheduleCmpOpts...))
			}
		})
	}

}

func TestWorkflowExecutionExecutorStatusUpdate(t *testing.T) {
	tests := map[string]struct {
		obj            *testworkflowsv1.TestWorkflowExecution
		execErr        error
		expectError    string
		expectGenSet   bool
		expectLastErr  string
		expectEventNum int
	}{
		"status updated on success": {
			obj: &testworkflowsv1.TestWorkflowExecution{
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
			execErr:        nil,
			expectGenSet:   true,
			expectLastErr:  "",
			expectEventNum: 1,
		},
		"status updated on failure": {
			obj: &testworkflowsv1.TestWorkflowExecution{
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
			execErr:        errors.New("execution failed"),
			expectError:    "executing test workflow from execution",
			expectGenSet:   true,
			expectLastErr:  "execution failed",
			expectEventNum: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &fakeTestWorkflowExecutor{req: test.execErr}
			scheme := runtime.NewScheme()
			if err := testworkflowsv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.obj).WithStatusSubresource(test.obj).Build()
			recorder := record.NewFakeRecorder(10)
			reconciler := testWorkflowExecutionExecutor(k8sClient, recorder, exec)

			request := reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      test.obj.Name,
					Namespace: test.obj.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), request)
			if test.expectError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", test.expectError)
				} else if !contains(err.Error(), test.expectError) {
					t.Errorf("expected error containing %q, got %q", test.expectError, err.Error())
				}
			}

			// Get the updated object
			var updated testworkflowsv1.TestWorkflowExecution
			if err := k8sClient.Get(context.Background(), client.ObjectKey{
				Name:      test.obj.Name,
				Namespace: test.obj.Namespace,
			}, &updated); err != nil {
				t.Fatalf("failed to get updated object: %v", err)
			}

			// Check status
			if test.expectGenSet && updated.Status.Generation != test.obj.Generation {
				t.Errorf("expected Generation to be %d, got %d", test.obj.Generation, updated.Status.Generation)
			}
			if updated.Status.LastError != test.expectLastErr {
				t.Errorf("expected LastError to be %q, got %q", test.expectLastErr, updated.Status.LastError)
			}

			// Check events
			close(recorder.Events)
			eventCount := 0
			for range recorder.Events {
				eventCount++
			}
			if eventCount != test.expectEventNum {
				t.Errorf("expected %d events, got %d", test.expectEventNum, eventCount)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
