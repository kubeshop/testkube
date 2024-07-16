package action

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

func TestGroup_Basic(t *testing.T) {
	input := []actiontypes.Action{
		// Configure
		setup(true, true),

		// Declare stage conditions
		declare("init", "true"),
		declare("step1", "false"),
		declare("step2", "true", "init"),
		declare("step3", "true", "init"),

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
		containerConfig("step2", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step2"),
		execute("step2", false),
		end("step2"),

		// Run the step 3
		status("init"),
		containerConfig("step3", testworkflowsv1.ContainerConfig{
			Image:   "image:3.2.1",
			Command: common.Ptr([]string{"c", "d"}),
		}),
		start("step3"),
		execute("step3", false),
		end("step3"),

		// Finish
		end("init"),
		end(""),
	}
	want := [][]actiontypes.Action{
		input[:13],   // ends before containerConfig("step2")
		input[13:18], // ends before containerConfig("step3")
		input[18:],
	}
	got := Group(input)
	assert.Equal(t, want, got)
}
