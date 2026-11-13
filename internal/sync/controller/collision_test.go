package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

func TestCheckNameCollision(t *testing.T) {
	earlier := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	later := metav1.NewTime(earlier.Add(time.Hour))
	workflow := func(namespace string, created metav1.Time) testworkflowsv1.TestWorkflow {
		return testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: namespace, CreationTimestamp: created}}
	}

	tests := []struct {
		name      string
		obj       testworkflowsv1.TestWorkflow
		existing  []testworkflowsv1.TestWorkflow
		listErr   error
		wantErr   error
		retryable bool
	}{
		{
			name:     "a later resource with the same name in the same Environment is refused",
			obj:      workflow("team-c", later),
			existing: []testworkflowsv1.TestWorkflow{workflow("team-b", earlier), workflow("team-c", later)},
			wantErr:  syncagent.ErrNameCollision,
		},
		{
			name:     "a resource created at the same time is refused when its namespace sorts later",
			obj:      workflow("team-c", earlier),
			existing: []testworkflowsv1.TestWorkflow{workflow("team-b", earlier), workflow("team-c", earlier)},
			wantErr:  syncagent.ErrNameCollision,
		},
		{
			name:     "the earlier resource syncs",
			obj:      workflow("team-b", earlier),
			existing: []testworkflowsv1.TestWorkflow{workflow("team-b", earlier), workflow("team-c", later)},
		},
		{
			name:     "the same name in another Environment syncs",
			obj:      workflow("team-b", later),
			existing: []testworkflowsv1.TestWorkflow{workflow("team-a", earlier), workflow("team-b", later)},
		},
		{
			name:     "the same namespace is the resource itself",
			obj:      workflow("team-b", later),
			existing: []testworkflowsv1.TestWorkflow{workflow("team-b", earlier)},
		},
		{
			name:      "a list error is retryable",
			obj:       workflow("team-b", later),
			listErr:   errors.New("cache not synced"),
			retryable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := fakeKubernetesClient{TestWorkflows: tt.existing, ListErr: tt.listErr}

			err := checkNameCollision(t.Context(), reader, &testworkflowsv1.TestWorkflowList{}, &tt.obj, testTarget(tt.obj.Namespace), testTarget)

			switch {
			case tt.retryable:
				require.Error(t, err)
				assert.False(t, syncagent.IsRejection(err), "a list error must stay retryable, got %v", err)
			case tt.wantErr != nil:
				assert.ErrorIs(t, err, tt.wantErr)
			default:
				assert.NoError(t, err)
			}
		})
	}
}
