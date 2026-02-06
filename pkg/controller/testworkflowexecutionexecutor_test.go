package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	if f.err != nil {
		return nil, f.err
	}
	return []testkubev1.TestWorkflowExecution{}, nil
}

func TestWorkflowExecutionExecutorController(t *testing.T) {
	tests := map[string]struct {
		objs    []client.Object
		request reconcile.Request
		expect  *cloud.ScheduleRequest
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
				KubernetesObjectName: "test-workflow-execution",
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
				KubernetesObjectName: "test-workflow-execution",
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
			exec := &fakeTestWorkflowExecutor{}
			scheme := runtime.NewScheme()
			if err := testworkflowsv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
			}
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(test.objs...).
				WithStatusSubresource(&testworkflowsv1.TestWorkflowExecution{}).
				Build()
			reconciler := testWorkflowExecutionExecutor(k8sClient, exec)

			_, err := reconciler.Reconcile(context.Background(), test.request)
			if err != nil {
				t.Errorf("reconcile: %v", err)
			}
			if !cmp.Equal(test.expect, exec.req, scheduleCmpOpts...) {
				t.Errorf("Incorrect execution request, diff: %s", cmp.Diff(test.expect, exec.req, scheduleCmpOpts...))
			}
		})
	}

}

func TestWorkflowExecutionExecutorController_ErrorHandling(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := testworkflowsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
	}

	twe := &testworkflowsv1.TestWorkflowExecution{
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
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(twe).
		WithStatusSubresource(&testworkflowsv1.TestWorkflowExecution{}).
		Build()

	exec := &fakeTestWorkflowExecutor{err: errors.New("workflow not found")}
	reconciler := testWorkflowExecutionExecutor(k8sClient, exec)

	_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: client.ObjectKey{
			Name:      "test-workflow-execution",
			Namespace: "test-namespace",
		},
	})

	// Reconcile should return an error
	if err == nil {
		t.Error("expected reconcile to return an error")
	}

	// Check that the status was updated with the error
	var updated testworkflowsv1.TestWorkflowExecution
	if err := k8sClient.Get(context.Background(), client.ObjectKey{
		Name:      "test-workflow-execution",
		Namespace: "test-namespace",
	}, &updated); err != nil {
		t.Fatalf("failed to get updated TestWorkflowExecution: %v", err)
	}

	// Verify the status reflects the error
	if updated.Status.LatestExecution == nil {
		t.Fatal("expected LatestExecution to be set")
	}
	if updated.Status.LatestExecution.Result == nil {
		t.Fatal("expected Result to be set")
	}
	if updated.Status.LatestExecution.Result.Status == nil || *updated.Status.LatestExecution.Result.Status != testworkflowsv1.FAILED_TestWorkflowStatus {
		t.Errorf("expected status to be failed, got: %v", updated.Status.LatestExecution.Result.Status)
	}
	if updated.Status.LatestExecution.Result.Initialization == nil {
		t.Fatal("expected Initialization to be set")
	}
	if !strings.Contains(updated.Status.LatestExecution.Result.Initialization.ErrorMessage, "workflow not found") {
		t.Errorf("expected error message to contain 'workflow not found', got: %s", updated.Status.LatestExecution.Result.Initialization.ErrorMessage)
	}
	if updated.Status.Generation != 1 {
		t.Errorf("expected generation to be 1, got: %d", updated.Status.Generation)
	}
}
