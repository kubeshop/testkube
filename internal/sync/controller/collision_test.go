package controller

import (
	"context"
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

func TestSyncDelete(t *testing.T) {
	earlier := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	later := metav1.NewTime(earlier.Add(time.Hour))
	now := metav1.Now()
	workflow := func(namespace string, created metav1.Time) testworkflowsv1.TestWorkflow {
		return testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: namespace, CreationTimestamp: created}}
	}
	deleting := workflow("team-x", earlier)
	deleting.DeletionTimestamp = &now

	tests := []struct {
		name          string
		namespace     string
		existing      []testworkflowsv1.TestWorkflow
		upsertErr     error
		wantUpdated   string
		wantDeleted   bool
		wantRetryable bool
	}{
		{
			name:        "without another resource of the name the record goes",
			namespace:   "team-b",
			existing:    []testworkflowsv1.TestWorkflow{workflow("team-a", earlier)},
			wantDeleted: true,
		},
		{
			name:        "the delete of a refused resource sends the winner instead",
			namespace:   "team-c",
			existing:    []testworkflowsv1.TestWorkflow{workflow("team-b", earlier)},
			wantUpdated: "team-b",
		},
		{
			name:        "the delete of the winner hands the record to the next resource",
			namespace:   "team-b",
			existing:    []testworkflowsv1.TestWorkflow{workflow("team-c", later)},
			wantUpdated: "team-c",
		},
		{
			name:        "the earliest of the other resources wins",
			namespace:   "default",
			existing:    []testworkflowsv1.TestWorkflow{workflow("team-z", later), workflow("team-y", earlier)},
			wantUpdated: "team-y",
		},
		{
			name:        "a resource marked for deletion does not win",
			namespace:   "default",
			existing:    []testworkflowsv1.TestWorkflow{deleting},
			wantDeleted: true,
		},
		{
			name:        "an ownership conflict on the update deletes the record",
			namespace:   "team-b",
			existing:    []testworkflowsv1.TestWorkflow{workflow("team-c", later)},
			upsertErr:   syncagent.ErrOwnershipConflict,
			wantUpdated: "team-c",
			wantDeleted: true,
		},
		{
			name:          "another update error is returned",
			namespace:     "team-b",
			existing:      []testworkflowsv1.TestWorkflow{workflow("team-c", later)},
			upsertErr:     errors.New("unavailable"),
			wantUpdated:   "team-c",
			wantRetryable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := fakeKubernetesClient{TestWorkflows: tt.existing}
			var updated string
			var deleted bool
			upsert := func(_ context.Context, target syncagent.Target, obj testworkflowsv1.TestWorkflow) error {
				assert.Equal(t, testTarget(obj.Namespace), target)
				updated = obj.Namespace
				return tt.upsertErr
			}
			del := func(_ context.Context, target syncagent.Target, name string) error {
				assert.Equal(t, testTarget(tt.namespace), target)
				assert.Equal(t, "smoke", name)
				deleted = true
				return nil
			}

			err := syncDelete(t.Context(), reader, &testworkflowsv1.TestWorkflowList{}, "smoke", testTarget(tt.namespace), testTarget, upsert, del)

			if tt.wantRetryable {
				require.Error(t, err)
				assert.False(t, syncagent.IsRejection(err))
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantUpdated, updated)
			assert.Equal(t, tt.wantDeleted, deleted)
		})
	}
}
