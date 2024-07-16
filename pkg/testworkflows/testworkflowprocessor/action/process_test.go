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

func setup(copyInit, copyBinaries bool) actiontypes.Action {
	return actiontypes.Action{Setup: &actiontypes.ActionSetup{CopyInit: copyInit, CopyBinaries: copyBinaries}}
}

func declare(ref, condition string, parents ...string) actiontypes.Action {
	return actiontypes.Action{Declare: &actiontypes.ActionDeclare{Ref: ref, Condition: simplify(condition), Parents: parents}}
}

func start(ref string) actiontypes.Action {
	return actiontypes.Action{Start: common.Ptr(ref)}
}

func pause(ref string) actiontypes.Action {
	return actiontypes.Action{Pause: &actiontypes.ActionPause{Ref: ref}}
}

func end(ref string) actiontypes.Action {
	return actiontypes.Action{End: common.Ptr(ref)}
}

func status(status string) actiontypes.Action {
	return actiontypes.Action{CurrentStatus: common.Ptr(simplify(status))}
}

func result(ref, value string) actiontypes.Action {
	return actiontypes.Action{Result: &actiontypes.ActionResult{Ref: ref, Value: simplify(value)}}
}

func execute(ref string, negative bool) actiontypes.Action {
	return actiontypes.Action{Execute: &actiontypes.ActionExecute{Ref: ref, Negative: negative}}
}

func containerConfig(ref string, config testworkflowsv1.ContainerConfig) actiontypes.Action {
	return actiontypes.Action{Container: &actiontypes.ActionContainer{Ref: ref, Config: config}}
}

func TestProcess_BasicSteps(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.Add(stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "true", "init"),
		declare("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "step1 && step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", false),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "true", "init"),
		declare("step2", "step1", "init"),

		// Declare information about potential pauses
		pause("step1"),

		// Declare stage resolutions
		result("init", "step1 && step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", false),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "true", "init"),
		declare("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "step1 && step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", true),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "true", "init"),
		declare("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "!step1 || !step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", false),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "true", "init"),
		declare("step2", "true", "init"), // because step1 is optional

		// Declare stage resolutions
		result("init", "step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", false),
		end("step1"),

		// Run the step 2
		status("init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("inner", "true", "init"),
		declare("step1", "true", "init", "inner"),
		declare("step2", "step1", "init", "inner"),

		// Declare stage resolutions
		result("inner", "step1 && step2"),
		result("init", "true"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),
		status("init"),
		start("inner"),

		// Run the step 1
		status("inner && init"),
		containerConfig("step1", testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		start("step1"),
		execute("step1", false),
		end("step1"),

		// Run the step 2
		status("step1 && inner && init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("inner"),
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "false"),
		declare("step2", "true", "init"), // because step1 is skipped

		// Declare stage resolutions
		result("init", "step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		start("step1"),

		// Run the step 2
		status("init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "false"),
		declare("step1", "false"),
		declare("step2", "false"),

		// Declare stage resolutions
		result("", "true"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("true"),
		start("step1"),

		// Run the step 2
		status("true"),
		start("step2"),

		// Finish
		end(""),
	}

	// Assert
	got, err := Process(root)
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
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "false"),
		declare("step1", "false"),
		declare("step2", "false"),

		// Declare stage resolutions
		result("", "true"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("true"),
		start("step1"),

		// Run the step 2
		status("true"),
		start("step2"),

		// Finish
		end(""),
	}

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

// TODO: Check how the sole `paused: true` step works
func TestProcess_IgnoreExecutionOfStaticSkip_PauseGroup(t *testing.T) {
	// Build the structure
	root := stage.NewGroupStage("init", false)
	root.SetPaused(true)
	step1 := stage.NewContainerStage("step1", stage.NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("false")
	root.Add(step1)
	root.Add(stage.NewContainerStage("step2", stage.NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "false"),
		declare("step2", "true", "init"), // because step1 is skipped

		// Declare information about potential pauses
		pause("init"),

		// Declare stage resolutions
		result("init", "step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		start("step1"),

		// Run the step 2
		status("init"),
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := Process(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
