package action

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

func TestGroup_Basic(t *testing.T) {
	input := actiontypes.NewActionList().
		// Configure
		Setup(true, true).
		Declare("init", "true").
		Declare("step1", "false").
		Declare("step2", "true", "init").
		Declare("step3", "true", "init").
		Result("init", "step2 && step3").
		Result("", "init").
		Start("").
		CurrentStatus("true").
		Start("init").
		// Step 1 is never running
		CurrentStatus("init").
		Start("step1").
		CurrentStatus("init").

		// Run step 2
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").
		CurrentStatus("init").

		// Run step 3
		MutateContainer("step3", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step3").
		Execute("step3", false).
		End("step3").
		End("init").
		End("")

	want := [][]actiontypes.Action{
		input[:13],   // ends before containerConfig("step2")
		input[13:18], // ends before containerConfig("step3")
		input[18:],
	}
	got := Group(input)
	assert.Equal(t, want, got)
}
