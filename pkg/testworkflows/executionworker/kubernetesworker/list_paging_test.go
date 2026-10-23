package kubernetesworker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func executionJob(t *testing.T, name, resourceID string) batchv1.Job {
	t.Helper()
	cfg := testworkflowconfig.InternalConfig{
		Resource: testworkflowconfig.ResourceConfig{Id: resourceID, RootId: resourceID},
	}
	encoded, err := json.Marshal(cfg)
	require.NoError(t, err)

	return batchv1.Job{
		// Every execution job carries this, and List now asks the API server for it
		// rather than listing the namespace and sorting it out here.
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "exec-ns",
			Labels:    map[string]string{constants.ResourceIdLabelName: resourceID},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{constants.InternalAnnotationName: string(encoded)},
				},
			},
		},
	}
}

// Limit is a page size, not a bound on what exists, and every filter this list applies
// - running or not, root or not, the organization, the environment - is applied to
// whatever came back. A namespace holding more jobs than one page therefore answered
// with a truncated list that looked complete.
//
// The step cache's lease renewal is one caller that asks which executions are running,
// and an execution missing from its answer has its inbox swept while its pod is still
// writing to it: the pointer that execution publishes then names an entry that is gone,
// under a key no later run can replace.
func TestListPagesThroughEveryJob(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	// The fake client does not page, so the pages are staged here: the first response
	// carries a continue token, and only the second completes the list.
	var calls int
	clientSet.PrependReactor("list", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls == 1 {
			return true, &batchv1.JobList{
				ListMeta: metav1.ListMeta{Continue: "page-2"},
				Items:    []batchv1.Job{executionJob(t, "first", "exec-1")},
			}, nil
		}
		// The token itself is not asserted: the fake's ListAction carries the label and
		// field selectors but not Continue, so what a page request echoed is not
		// observable here. What is observable - that a page carrying a token is
		// followed by another request, and that both pages reach the result - is what
		// fails when the paging is removed.
		return true, &batchv1.JobList{
			Items: []batchv1.Job{executionJob(t, "second", "exec-2")},
		}, nil
	})

	w := &worker{
		clientSet: clientSet,
		config:    Config{Cluster: ClusterConfig{Namespaces: map[string]NamespaceConfig{"exec-ns": {}}}},
	}

	got, err := w.List(context.Background(), executionworkertypes.ListOptions{Finished: common.Ptr(false)})

	require.NoError(t, err)
	assert.Equal(t, 2, calls, "a page with a continue token is not the end of the list")

	ids := make([]string, 0, len(got))
	for _, item := range got {
		ids = append(ids, item.Resource.Id)
	}
	assert.ElementsMatch(t, []string{"exec-1", "exec-2"}, ids,
		"an execution beyond the first page is still running")
}

// Asked of the API server rather than sorted out here. Without it the request is every
// Job in every configured namespace, each one's internal annotation unmarshalled only
// to be discarded and a warning logged for every Job that was never an execution - and
// the step cache's lease renewal asks this every five minutes.
func TestListAsksTheApiServerForExecutionJobsOnly(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	var selectors []string
	clientSet.PrependReactor("list", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(k8stesting.ListAction)
		require.True(t, ok)
		selectors = append(selectors, listAction.GetListRestrictions().Labels.String())
		return true, &batchv1.JobList{}, nil
	})

	w := &worker{
		clientSet: clientSet,
		config:    Config{Cluster: ClusterConfig{Namespaces: map[string]NamespaceConfig{"exec-ns": {}}}},
	}

	_, err := w.List(context.Background(), executionworkertypes.ListOptions{Finished: common.Ptr(false)})

	require.NoError(t, err)
	require.Len(t, selectors, 1)
	assert.Contains(t, selectors[0], constants.ResourceIdLabelName,
		"the namespace holds Jobs that are not executions, and they are not this query's business")
}
