package presets

import (
	"context"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	testworkflowprocessortcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

func NewOpenSource(inspector imageinspector.Inspector) testworkflowprocessor.Processor {
	return newOpenSource(inspector, testworkflowprocessor.ProcessArtifacts)
}

// NewOpenSourceWithLocalArtifacts builds the standalone processor used by
// testkube local when the caller requested an artifact export directory. It
// changes only the generated artifacts stage: normal workflow steps retain the
// standard open-source processing behavior, while artifacts receive the
// run-owned relay URL and token Secret reference.
func NewOpenSourceWithLocalArtifacts(inspector imageinspector.Inspector, uploadURL, tokenSecretName string) testworkflowprocessor.Processor {
	return &localArtifactsProcessor{
		Processor: newOpenSource(inspector, testworkflowprocessor.ProcessArtifactsWithLocalUpload(uploadURL, tokenSecretName)),
	}
}

// localArtifactsProcessor forces the local artifact stage into its own
// Kubernetes action group. The relay token is delivered through a Secret
// environment reference, and the normal purity optimizer may otherwise merge
// a local artifacts stage with a user shell that declares pure: true (or is
// made pure by spec.system.pureByDefault). The deep copy preserves the
// caller's workflow object while making this security boundary intrinsic to
// every local-artifact processor caller.
type localArtifactsProcessor struct {
	testworkflowprocessor.Processor
}

// Register preserves the wrapper when callers extend the preset fluently.
// Without this override, the promoted method would return the embedded
// processor and silently lose the local artifact isolation guarantee.
func (p *localArtifactsProcessor) Register(operation testworkflowprocessor.Operation) testworkflowprocessor.Processor {
	p.Processor.Register(operation)
	return p
}

func (p *localArtifactsProcessor) Bundle(
	ctx context.Context,
	workflow *testworkflowsv1.TestWorkflow,
	options testworkflowprocessor.BundleOptions,
	machines ...expressions.Machine,
) (*testworkflowprocessor.Bundle, error) {
	if workflow == nil {
		return p.Processor.Bundle(ctx, nil, options, machines...)
	}
	workflow = workflow.DeepCopy()
	if workflow.Spec.System == nil {
		workflow.Spec.System = &testworkflowsv1.TestWorkflowSystem{}
	}
	isolateContainers := true
	workflow.Spec.System.IsolatedContainers = &isolateContainers
	return p.Processor.Bundle(ctx, workflow, options, machines...)
}

func newOpenSource(inspector imageinspector.Inspector, processArtifacts testworkflowprocessor.Operation) testworkflowprocessor.Processor {
	return testworkflowprocessor.New(inspector).
		Register(testworkflowprocessor.ProcessDelay).
		Register(testworkflowprocessor.ProcessContentFiles).
		Register(testworkflowprocessor.ProcessContentGit).
		Register(testworkflowprocessor.ProcessContentTarball).
		Register(testworkflowprocessor.StubServices).
		Register(testworkflowprocessor.ProcessNestedSetupSteps).
		Register(testworkflowprocessor.ProcessRunCommand).
		Register(testworkflowprocessor.ProcessShellCommand).
		Register(testworkflowprocessor.StubExecute).
		Register(testworkflowprocessor.StubParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(processArtifacts)
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
		Register(testworkflowprocessortcl.ProcessExecute).
		Register(testworkflowprocessortcl.ProcessParallel).
		Register(testworkflowprocessor.ProcessNestedSteps).
		Register(testworkflowprocessortcl.ProcessServicesStop).
		Register(testworkflowprocessor.ProcessArtifacts)
}
