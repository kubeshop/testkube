package controller

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	commonv1 "github.com/kubeshop/testkube-operator/api/common/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
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
}

func (f *fakeTestWorkflowExecutor) Execute(_ context.Context, _ string, request *cloud.ScheduleRequest) testworkflowexecutor.TestWorkflowExecutionStream {
	f.req = request
	return testworkflowexecutor.NewStream[*testkubev1.TestWorkflowExecution](make(chan *testkubev1.TestWorkflowExecution))
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
			exec := &fakeTestWorkflowExecutor{}
			scheme := runtime.NewScheme()
			if err := testworkflowsv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add testworkflowsv1 to scheme: %v", err)
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.objs...).Build()
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
