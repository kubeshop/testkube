package presets

import (
	"errors"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	testworkflowprocessortcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// errAgentOperationHasNoProcessor is the answer of the commercial preset to an agent step. The
// message of the open source preset names the standalone mode, which is false for this preset.
var errAgentOperationHasNoProcessor = errors.New(`"agent" operation is not available in this version of the Runner`)

// refuseAgent refuses the agent step. A step that no processor claims disappears from the
// workflow with no error.
func refuseAgent(_ testworkflowprocessor.InternalProcessor, _ testworkflowprocessor.Intermediate, _ stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Agent != nil {
		return nil, errAgentOperationHasNoProcessor
	}
	return nil, nil
}

func NewOpenSource(inspector imageinspector.Inspector) testworkflowprocessor.Processor {
	return testworkflowprocessor.New(inspector).
		Register(testworkflowprocessor.ProcessDelay).
		Register(testworkflowprocessor.ProcessContentFiles).
		Register(testworkflowprocessor.ProcessContentGit).
		Register(testworkflowprocessor.ProcessContentTarball).
		// After the content operations, so the repository is checked out when the key
		// is resolved, and before anything that consumes the cache.
		Register(testworkflowprocessor.ProcessCacheRestore).
		Register(testworkflowprocessor.StubServices).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		Register(testworkflowprocessor.StubExecute).
		Register(testworkflowprocessor.StubAgent).
		Register(testworkflowprocessor.StubParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		// After every producer, and before artifacts, which run on "always" and are
		// the step's reporting path.
		Register(testworkflowprocessor.ProcessCacheSave).
		Register(testworkflowprocessor.ProcessArtifacts)
}

func NewPro(inspector imageinspector.Inspector) testworkflowprocessor.Processor {
	return testworkflowprocessor.New(inspector).
		Register(testworkflowprocessor.ProcessDelay).
		Register(testworkflowprocessor.ProcessContentFiles).
		Register(testworkflowprocessor.ProcessContentGit).
		Register(testworkflowprocessor.ProcessContentTarball).
		// After the content operations, so the repository is checked out when the key
		// is resolved, and before anything that consumes the cache.
		Register(testworkflowprocessor.ProcessCacheRestore).
		Register(testworkflowprocessortcl.ProcessServicesStart).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		Register(testworkflowprocessortcl.ProcessExecute).
		Register(refuseAgent).
		Register(testworkflowprocessortcl.ProcessParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(testworkflowprocessortcl.ProcessServicesStop).
		// After the services stop, so nothing is still writing into a cached directory
		// while it is being packed.
		Register(testworkflowprocessor.ProcessCacheSave).
		Register(testworkflowprocessor.ProcessArtifacts)
}
