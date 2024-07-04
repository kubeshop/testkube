package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func simplify(condition string) string {
	c, err := expressions.MustCompile(condition).Resolve()
	if err != nil {
		panic(err)
	}
	return c.String()
}

func condition(ref, condition string, parents ...string) Action {
	return Action{Condition: &ActionCondition{Ref: ref, Condition: simplify(condition), Parents: parents}}
}

func start(ref string) Action {
	return Action{Start: common.Ptr(ref)}
}

func pause(ref string) Action {
	return Action{Pause: &ActionPause{Ref: ref}}
}

func end(ref string) Action {
	return Action{End: common.Ptr(ref)}
}

func status(status string) Action {
	return Action{CurrentStatus: common.Ptr(simplify(status))}
}

func result(ref, value string) Action {
	return Action{Result: &ActionResult{Ref: ref, Value: simplify(value)}}
}

func execute(ref string, parents []string, negative bool, config testworkflowsv1.ContainerConfig) Action {
	return Action{Execute: &ActionExecute{Ref: ref, Parents: parents, Negative: negative, Config: config}}
}

func TestAnalyzeOperations_BasicSteps(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	root.Add(NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "true", "init"),
		condition("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "step1 && step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		start("step1"),
		execute("step1", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_Pause(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	step1 := NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetPaused(true)
	root.Add(step1)
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "true", "init"),
		condition("step2", "step1", "init"),

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
		start("step1"),
		execute("step1", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_NegativeStep(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	step1 := NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetNegative(true)
	root.Add(step1)
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "true", "init"),
		condition("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "step1 && step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		start("step1"),
		execute("step1", []string{"init"}, true, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_NegativeGroup(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	root.SetNegative(true)
	root.Add(NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "true", "init"),
		condition("step2", "step1", "init"),

		// Declare stage resolutions
		result("init", "!step1 || !step2"),
		result("", "init"),

		// Initialize
		start(""),
		status("true"),
		start("init"),

		// Run the step 1
		status("init"),
		start("step1"),
		execute("step1", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("step1 && init"),
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_OptionalStep(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	step1 := NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetOptional(true)
	root.Add(step1)
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "true", "init"),
		condition("step2", "true", "init"), // because step1 is optional

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
		execute("step1", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("init"),
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_OptionalGroup(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	group := NewGroupStage("inner", false)
	group.SetOptional(true)
	group.Add(NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	group.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))
	root.Add(group)

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("inner", "true", "init"),
		condition("step1", "true", "init", "inner"),
		condition("step2", "step1", "init", "inner"),

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
		start("step1"),
		execute("step1", []string{"init", "inner"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:1.2.3",
			Command: common.Ptr([]string{"a", "b"}),
		}),
		end("step1"),

		// Run the step 2
		status("step1 && inner && init"),
		start("step2"),
		execute("step2", []string{"init", "inner"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("inner"),
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_IgnoreExecutionOfStaticSkip(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	step1 := NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("false")
	root.Add(step1)
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "false"),
		condition("step2", "true", "init"), // because step1 is skipped

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
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_IgnoreExecutionOfStaticSkipGroup(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	root.SetCondition("false")
	root.Add(NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "false"),
		condition("step1", "false"),
		condition("step2", "false"),

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
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_IgnoreExecutionOfStaticSkipGroup_Pause(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	root.SetCondition("false")
	root.SetPaused(true)
	root.Add(NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b")))
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "false"),
		condition("step1", "false"),
		condition("step2", "false"),

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
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAnalyzeOperations_IgnoreExecutionOfStaticSkip_PauseGroup(t *testing.T) {
	// Build the structure
	root := NewGroupStage("init", false)
	root.SetPaused(true)
	step1 := NewContainerStage("step1", NewContainer().SetImage("image:1.2.3").SetCommand("a", "b"))
	step1.SetCondition("false")
	root.Add(step1)
	root.Add(NewContainerStage("step2", NewContainer().SetImage("image:3.2.1").SetCommand("c", "d")))

	// Build the expectations
	want := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "false"),
		condition("step2", "true", "init"), // because step1 is skipped

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
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Finish
		end("init"),
		end(""),
	}

	// Assert
	got, err := AnalyzeOperations(root)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGroupActions_Basic(t *testing.T) {
	input := []Action{
		// Declare stage conditions
		condition("init", "true"),
		condition("step1", "false"),
		condition("step2", "true", "init"),
		condition("step3", "true", "init"),

		// Declare stage resolutions
		result("init", "step2 && step3"),
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
		start("step2"),
		execute("step2", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step2"),

		// Run the step 3
		status("init"),
		start("step3"),
		execute("step3", []string{"init"}, false, testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		end("step3"),

		// Finish
		end("init"),
		end(""),
	}
	want := [][]Action{
		input[:16], // ends before start("step3")
		input[16:],
	}
	got := GroupActions(input)
	assert.Equal(t, want, got)
}
