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

func TestProcess_BasicSteps(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestProcess_Grouping(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	group1 := stage.NewGroupStage("group1", false)
	group1.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))
	group1.Add(stage.NewContainerStage("step3", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))
	root.Add(group1)
	root.Add(stage.NewContainerStage("step4", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("group1", "step1", "init").
		Declare("step2", "step1", "init", "group1").
		Declare("step3", "step2&&step1", "init", "group1").
		Declare("step4", "group1&&step1", "init").

		// Declare group resolutions
		Result("group1", "step2&&step3").
		Result("init", "step1&&group1&&step4").
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
		Execute("step1", false, false).
		End("step1").

		// Start the group 1
		CurrentStatus("step1&&init").
		Start("group1").

		// Run the step 2
		CurrentStatus("group1&&step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Run the step 2
		CurrentStatus("step2&&group1&&step1&&init").
		MutateContainer("step3", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step3").
		Execute("step3", false, false).
		End("step3").

		// End the group 1
		End("group1").

		// Run the step 4
		CurrentStatus("group1&&step1&&init").
		MutateContainer("step4", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step4").
		Execute("step4", false, false).
		End("step4").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		Execute("step1", true, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestProcess_NegativeGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetNegative(true)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&inner&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("inner").
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		End("step1").

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestProcess_IgnoreExecutionOfStaticSkipGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetCondition("false")
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := actiontypes.NewActionList().
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
		End("step1").

		// Run the step 2
		CurrentStatus("true").
		Start("step2"). // don't execute as it is skipped
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		End("step1").

		// Run the step 2
		CurrentStatus("true").
		Start("step2"). // don't execute as it is skipped
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
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
		End("step1").

		// Run the step 2
		CurrentStatus("init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestProcess_ConsecutiveAlways(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("always")
	root.Add(step1)
	step2 := stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d"))
	step2.SetCondition("always")
	root.Add(step2)

	// Build the expectations
	want := actiontypes.NewActionList().
		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "true", "init").

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
		Execute("step1", false, false).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}).
		Start("step2").
		Execute("step2", false, false).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestProcess_PureShellAtTheEnd(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand(constants.DefaultShellPath))
	step1.SetPure(true)
	step1.SetCondition("always")
	root.Add(step1)
	step2 := stage.NewContainerStage("step2", stage.NewContainer().SetCommand(constants.DefaultShellPath))
	step2.SetPure(true)
	step2.SetCondition("always")
	root.Add(step2)

	// Build the expectations
	want := actiontypes.NewActionList().
		// Declare stage conditions
		Declare("init", "true").
		Declare("step1", "true", "init").
		Declare("step2", "true", "init").

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
			Command: common.Ptr([]string{constants.DefaultShellPath}),
		}).
		Start("step1").
		Execute("step1", false, true).
		End("step1").

		// Run the step 2
		CurrentStatus("step1&&init").
		MutateContainer("step2", testworkflowsv1.ContainerConfig{
			Command: common.Ptr([]string{constants.DefaultShellPath}),
		}).
		Start("step2").
		Execute("step2", false, true).
		End("step2").

		// Finish
		End("init").
		End("")

	// Assert
	got, err := Process(root, nil)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
