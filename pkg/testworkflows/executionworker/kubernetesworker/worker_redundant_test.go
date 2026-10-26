package kubernetesworker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestWorker_ExecuteWhenTheJobAlreadyExists(t *testing.T) {
	existingJob := func(runnerId string) *batchv1.Job {
		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Namespace: "ns"}}
		if runnerId != "" {
			job.Spec.Template.Labels = map[string]string{constants.RunnerIdLabelName: runnerId}
		}
		return job
	}

	tests := []struct {
		name          string
		existing      *batchv1.Job
		wantRedundant bool
	}{
		{name: "creates the job when there is none", existing: nil},
		{name: "is redundant when this runner already started the execution", existing: existingJob("runner-1"), wantRedundant: true},
		{name: "fails when the job belongs to a different runner", existing: existingJob("runner-2")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientSet := fake.NewSimpleClientset()
			if tt.existing != nil {
				clientSet = fake.NewSimpleClientset(tt.existing)
			}
			processor := testworkflowprocessor.NewMockProcessor(gomock.NewController(t))
			processor.EXPECT().Bundle(gomock.Any(), gomock.Any(), gomock.Any()).Return(&testworkflowprocessor.Bundle{
				Job: batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Namespace: "ns"}},
			}, nil)
			w := NewWorker(clientSet, processor, Config{
				Cluster:  ClusterConfig{DefaultNamespace: "ns", Namespaces: map[string]NamespaceConfig{"ns": {}}},
				RunnerId: "runner-1",
			})

			result, err := w.Execute(context.Background(), executionworkertypes.ExecuteRequest{
				Execution: testworkflowconfig.ExecutionConfig{Id: "exec-1"},
			})

			switch {
			case tt.existing == nil:
				require.NoError(t, err)
				assert.False(t, result.Redundant)
			case tt.wantRedundant:
				require.NoError(t, err, "a second start of a running execution must not decline it")
				assert.True(t, result.Redundant)
				assert.Equal(t, "ns", result.Namespace)
			default:
				require.Error(t, err)
				assert.Equal(t, testkube.StartReasonJobCreateFailed, executionworkertypes.StartReasonOf(err))
			}
		})
	}
}
