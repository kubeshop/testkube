package presets

import (
	"github.com/kubeshop/testkube/pkg/imageinspector"
	testworkflowprocessortcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

func NewOpenSource(inspector imageinspector.Inspector) testworkflowprocessor.Processor {
	return testworkflowprocessor.New(inspector).
		Register(testworkflowprocessor.ProcessDelay).
		Register(testworkflowprocessor.ProcessContentFiles).
		Register(testworkflowprocessor.ProcessContentGit).
		Register(testworkflowprocessor.ProcessContentTarball).
		Register(testworkflowprocessor.StubServices).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		// After the operations that create the container a policy attaches to,
		// so it sees the step whose report it describes.
		Register(testworkflowprocessor.StubTestCases).
		Register(testworkflowprocessor.StubExecute).
		Register(testworkflowprocessor.StubParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(testworkflowprocessor.ProcessArtifacts)
}

func NewPro(inspector imageinspector.Inspector) testworkflowprocessor.Processor {
	return testworkflowprocessor.New(inspector).
		Register(testworkflowprocessor.ProcessDelay).
		Register(testworkflowprocessor.ProcessContentFiles).
		Register(testworkflowprocessor.ProcessContentGit).
		Register(testworkflowprocessor.ProcessContentTarball).
		Register(testworkflowprocessortcl.ProcessServicesStart).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		// After the operations that create the container a policy attaches to,
		// so it sees the step whose report it describes.
		Register(testworkflowprocessortcl.ProcessTestCases).
		Register(testworkflowprocessortcl.ProcessExecute).
		Register(testworkflowprocessortcl.ProcessParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(testworkflowprocessortcl.ProcessServicesStop).
		Register(testworkflowprocessor.ProcessArtifacts)
}
