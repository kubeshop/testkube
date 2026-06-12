package testworkflowprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestBundle_InvalidEmptyDirSizeLimit_ReturnsError(t *testing.T) {
	proc := New(nil)
	workflow := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), workflow, BundleOptions{
		Config: testworkflowconfig.InternalConfig{
			Resource: testworkflowconfig.ResourceConfig{
				Id:     "resource-id",
				RootId: "resource-root-id",
			},
			Worker: testworkflowconfig.WorkerConfig{
				EmptyDirSizeLimit: "not-a-quantity",
			},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, `invalid worker emptyDir sizeLimit "not-a-quantity"`)
}
