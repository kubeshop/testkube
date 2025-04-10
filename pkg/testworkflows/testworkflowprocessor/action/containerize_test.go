package action

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func TestCreateContainer_MergingDefaultStepBefore(t *testing.T) {
	input := actiontypes.NewActionList().
		MutateContainer("ref1", testworkflowsv1.ContainerConfig{
			Command: common.Ptr([]string{"sleep"}),
		}).
		Start("ref1").
		Execute("ref1", false, true).
		End("ref1").
		CurrentStatus("ref1").
		MutateContainer("ref2", testworkflowsv1.ContainerConfig{
			Image:   "custom-image:1.2.3",
			Command: common.Ptr([]string{"my-test"}),
		}).
		Start("ref2").
		Execute("ref2", false, false).
		End("ref2").
		End(constants.RootOperationName).
		End("")
	result, _, _, err := CreateContainer(1, stage.NewContainer(), input, false)

	assert.NoError(t, err)
	assert.Equal(t, result.Image, "custom-image:1.2.3")
}

func TestCreateContainer_MergingDefaultStepAfter(t *testing.T) {
	input := actiontypes.NewActionList().
		MutateContainer("ref1", testworkflowsv1.ContainerConfig{
			Image:   "custom-image:1.2.3",
			Command: common.Ptr([]string{"my-test"}),
		}).
		Start("ref1").
		Execute("ref1", false, false).
		End("ref1").
		CurrentStatus("ref1").
		MutateContainer("ref2", testworkflowsv1.ContainerConfig{
			Command: common.Ptr([]string{"sleep"}),
		}).
		Start("ref2").
		Execute("ref2", false, true).
		End("ref2").
		End(constants.RootOperationName).
		End("")
	result, _, _, err := CreateContainer(1, stage.NewContainer(), input, false)

	assert.NoError(t, err)
	assert.Equal(t, result.Image, "custom-image:1.2.3")
}
