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
		Register(testworkflowprocessor.ProcessContentMinio).
		Register(testworkflowprocessor.StubServices).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
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
		Register(testworkflowprocessor.ProcessContentMinio).
		Register(testworkflowprocessortcl.ProcessServicesStart).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		Register(testworkflowprocessortcl.ProcessExecute).
		Register(testworkflowprocessortcl.ProcessParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(testworkflowprocessortcl.ProcessServicesStop).
		Register(testworkflowprocessor.ProcessArtifacts)
}
