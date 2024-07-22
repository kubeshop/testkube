package action

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func simplify(condition string) string {
	c, err := expressions.MustCompile(condition).Resolve()
	if err != nil {
		panic(err)
	}
	return c.String()
}

func TestProcess_BasicSteps(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "step1", "init").

		// Declare group resolutions
		Result("init", "step1&&step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_Pause(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetPaused(true)
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "step1", "init").

		// Declare information about potential pauses
		Pause("step1").

		// Declare group resolutions
		Result("init", "step1&&step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_NegativeStep(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetNegative(true)
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "step1", "init").

		// Declare group resolutions
		Result("init", "step1&&step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", true).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_NegativeGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetNegative(true)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "step1", "init").

		// Declare group resolutions
		Result("init", "!step1||!step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_OptionalStep(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetOptional(true)
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "true", "init"). // because step1 is optional

		// Declare group resolutions
		Result("init", "step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", false).
		End("step1").

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_OptionalGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	group := stage.NewGroupStage("inner", false)
	group.SetOptional(true)
	group.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	group.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))
	root.Add(group)

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("inner", "true", "init").
		Declare("step1", "true", "init", "inner").
		Declare("step2", "step1", "init", "inner").

		// Declare group resolutions
		Result("inner", "step1&&step2").
		Result("init", "true").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").
		CurrentStatus("init").
		Start("inner").

		// Run the step 1
		CurrentStatus("inner&&init").
		MutateContainer("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}).
		Start("step1").
		Execute("step1", false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&inner&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("inner").
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_IgnoreExecutionOfStaticSkip(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("false")
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "false").
		Declare("step2", "true", "init"). // because step1 is skipped

		// Declare group resolutions
		Result("init", "step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		Start("step1"). // don't execute as it is skipped
		//End("step1"). // FIXME: Shouldn't be????

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_IgnoreExecutionOfStaticSkipGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetCondition("false")
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "false").
		Declare("step1", "false").
		Declare("step2", "false").

		// Declare group resolutions
		Result("", "true").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("true").
		Start("step1"). // don't execute as it is skipped
		//End("step1"). // FIXME: Shouldn't be????

		// Run the step 2
		CurrentStatus("true").
		Start("step2"). // don't execute as it is skipped
		//End("step2"). // FIXME: Shouldn't be????

		// Finish
		//End("init"). // FIXME: Shouldn't be????
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_IgnoreExecutionOfStaticSkipGroup_Pause(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetCondition("false")
	root.SetPaused(true)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "false").
		Declare("step1", "false").
		Declare("step2", "false").

		//Pause("init"). // ignored as it's not executed

		// Declare group resolutions
		Result("", "true").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("true").
		Start("step1"). // don't execute as it is skipped
		//End("step1"). // FIXME: Shouldn't be????

		// Run the step 2
		CurrentStatus("true").
		Start("step2"). // don't execute as it is skipped
		//End("step2"). // FIXME: Shouldn't be????

		// Finish
		//End("init"). // FIXME: Shouldn't be????
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}

func TestProcess_IgnoreExecutionOfStaticSkip_PauseGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetPaused(true)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("false")
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Configure
		Setup(true, true).

		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "false").
		Declare("step2", "true", "init"). // because step1 is skipped

		// Declare information about potential pauses
		Pause("init").

		// Declare group resolutions
		Result("init", "step2").
		Result("", "init").

		// Initialize
		Start("").
		CurrentStatus("true").
		Start("init").

		// Run the step 1
		CurrentStatus("init").
		Start("step1"). // don't execute as it is skipped
		//End("step1"). // FIXME: Shouldn't be????

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, actiontypes.ActionList(got))
}
