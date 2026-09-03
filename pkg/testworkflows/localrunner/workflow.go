package localrunner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/validation"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	crdcommon "github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/k8s"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

// LoadWorkflow validates and decodes exactly one current TestWorkflow object.
// It never asks a Testkube API to validate or resolve the file.
func LoadWorkflow(filePath string) (*testworkflowsv1.TestWorkflow, []byte, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, nil, UsageError("--file is required")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, UsageError("reading workflow file %q: %v", filePath, err)
	}
	if err := exactlyOneYAMLDocument(raw); err != nil {
		return nil, nil, UsageError("workflow file %q: %v", filePath, err)
	}
	if err := crd.ValidateYAMLAgainstSchema(k8s.SchemaTestWorkflow, raw); err != nil {
		return nil, nil, UsageError("validating TestWorkflow schema: %v", err)
	}
	workflow := new(testworkflowsv1.TestWorkflow)
	if err := crdcommon.DeserializeCRD(workflow, raw); err != nil {
		return nil, nil, UsageError("decoding TestWorkflow: %v", err)
	}
	if workflow.APIVersion != testworkflowsv1.GroupVersion.String() {
		return nil, nil, UsageError("workflow apiVersion must be %q, got %q", testworkflowsv1.GroupVersion.String(), workflow.APIVersion)
	}
	if workflow.Kind != "TestWorkflow" {
		return nil, nil, UsageError("workflow kind must be %q, got %q", "TestWorkflow", workflow.Kind)
	}
	if workflow.Name == "" {
		return nil, nil, UsageError("workflow metadata.name is required")
	}
	if msgs := validation.IsDNS1123Subdomain(workflow.Name); len(msgs) > 0 {
		return nil, nil, UsageError("workflow metadata.name %q is not Kubernetes-safe: %s", workflow.Name, strings.Join(msgs, "; "))
	}
	return workflow, raw, nil
}

func exactlyOneYAMLDocument(raw []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	documents := 0
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parsing YAML: %w", err)
		}
		if len(node.Content) > 0 {
			documents++
		}
	}
	if documents != 1 {
		return fmt.Errorf("must contain exactly one YAML document, found %d", documents)
	}
	return nil
}

// ApplyConfig applies only non-sensitive workflow configuration values to a
// deep copy. Sensitive values are deliberately not accepted on a command line,
// where shell history and process inspection can disclose them.
func ApplyConfig(workflow *testworkflowsv1.TestWorkflow, config map[string]string) (*testworkflowsv1.TestWorkflow, error) {
	if len(config) == 0 {
		return workflow.DeepCopy(), nil
	}
	for key := range config {
		if schema, ok := workflow.Spec.Config[key]; ok && schema.Sensitive {
			return nil, UsageError("--config %s supplies a sensitive workflow value; create a local Kubernetes Secret and reference it instead", key)
		}
	}
	prepared := testworkflowexecutor.NewIntermediateExecution().SetWorkflow(workflow)
	if err := prepared.ApplyConfig(config); err != nil {
		return nil, UsageError("applying --config values: %v", err)
	}
	return testworkflows.MapAPIToKube(prepared.Execution().ResolvedWorkflow), nil
}

// ValidateSupported rejects features whose semantics require an installed
// Testkube API, separate execution workers, or artifact persistence. The
// returned error always names the user-facing YAML field that was rejected.
func ValidateSupported(workflow *testworkflowsv1.TestWorkflow, hasLocalSource, interactive, autoContinue bool) error {
	return ValidateSupportedInNamespace(workflow, "", hasLocalSource, interactive, autoContinue)
}

// ValidateSupportedInNamespace is the local compatibility gate with an
// optional destination namespace. Supplying the namespace lets callers reject
// a workflow that would otherwise redirect a local run to another namespace
// before any Kubernetes object is created.
func ValidateSupportedInNamespace(workflow *testworkflowsv1.TestWorkflow, namespace string, hasLocalSource, interactive, autoContinue bool) error {
	if workflow == nil {
		return UsageError("workflow is required")
	}
	if len(workflow.Spec.Use) > 0 {
		return UsageError("spec.use is not supported by testkube local because template resolution requires Testkube storage")
	}
	if len(workflow.Spec.Services) > 0 {
		return UsageError("spec.services is not supported by testkube local")
	}
	if len(workflow.Spec.Pvcs) > 0 {
		return UsageError("spec.pvcs is not supported by testkube local")
	}
	if workflow.Spec.Concurrency != nil {
		return UsageError("spec.concurrency is not supported by testkube local")
	}
	if len(workflow.Spec.Events) > 0 {
		return UsageError("spec.events is not supported by testkube local")
	}
	if workflow.Spec.Execution != nil && workflow.Spec.Execution.Target != nil {
		return UsageError("spec.execution.target is not supported by testkube local because target scheduling requires a Testkube control plane")
	}
	if workflow.Spec.Job != nil && workflow.Spec.Job.Namespace != "" && namespace != "" && workflow.Spec.Job.Namespace != namespace {
		return UsageError("spec.job.namespace %q does not match local namespace %q", workflow.Spec.Job.Namespace, namespace)
	}
	if err := validatePodSecurity(workflow.Spec.Pod, "spec.pod"); err != nil {
		return err
	}
	if err := validateContainerSecurity(workflow.Spec.Container, "spec.container"); err != nil {
		return err
	}
	if err := validateContent(workflow.Spec.Content, "spec.content", hasLocalSource, true); err != nil {
		return err
	}
	paused := false
	for _, group := range []struct {
		name  string
		steps []testworkflowsv1.Step
	}{
		{name: "spec.setup", steps: workflow.Spec.Setup},
		{name: "spec.steps", steps: workflow.Spec.Steps},
		{name: "spec.after", steps: workflow.Spec.After},
	} {
		for i := range group.steps {
			found, err := validateStep(group.steps[i], fmt.Sprintf("%s[%d]", group.name, i), hasLocalSource)
			if err != nil {
				return err
			}
			paused = paused || found
		}
	}
	if paused && !interactive && !autoContinue {
		return UsageError("workflow contains paused: true but standard input is not interactive; pass --auto-continue to run it without a breakpoint prompt")
	}
	return nil
}

func validateContent(content *testworkflowsv1.Content, path string, hasLocalSource, topLevel bool) error {
	if content == nil {
		return nil
	}
	if hasLocalSource && !topLevel && content.Git != nil {
		return UsageError("%s.git is not supported with --source; only top-level spec.content.git can be replaced by the local source archive", path)
	}
	return nil
}

func validateStep(step testworkflowsv1.Step, path string, hasLocalSource bool) (bool, error) {
	if len(step.Use) > 0 {
		return false, UsageError("%s.use is not supported by testkube local", path)
	}
	if step.Template != nil {
		return false, UsageError("%s.template is not supported by testkube local", path)
	}
	if len(step.Services) > 0 {
		return false, UsageError("%s.services is not supported by testkube local", path)
	}
	if step.Execute != nil {
		return false, UsageError("%s.execute is not supported by testkube local", path)
	}
	if step.Parallel != nil {
		return false, UsageError("%s.parallel is not supported by testkube local", path)
	}
	if step.Artifacts != nil {
		return false, UsageError("%s.artifacts is not supported by testkube local", path)
	}
	if err := validateContainerSecurity(step.Container, path+".container"); err != nil {
		return false, err
	}
	if step.Run != nil {
		if err := validateContainerSecurity(&step.Run.ContainerConfig, path+".run"); err != nil {
			return false, err
		}
	}
	if err := validateContent(step.Content, path+".content", hasLocalSource, false); err != nil {
		return false, err
	}
	paused := step.Paused
	for _, children := range []struct {
		name  string
		steps []testworkflowsv1.Step
	}{
		{name: path + ".setup", steps: step.Setup},
		{name: path + ".steps", steps: step.Steps},
	} {
		for i := range children.steps {
			childPaused, err := validateStep(children.steps[i], fmt.Sprintf("%s[%d]", children.name, i), hasLocalSource)
			if err != nil {
				return false, err
			}
			paused = paused || childPaused
		}
	}
	return paused, nil
}

func validatePodSecurity(pod *testworkflowsv1.PodConfig, path string) error {
	if pod == nil {
		return nil
	}
	// The normal processor refuses the presence of this field when low-security
	// execution is disabled, even if the value is false. Reject it here so the
	// failure is a stable, pre-mutation local-command error.
	if pod.HostPID != nil {
		return UsageError("%s.hostPID is not supported by testkube local", path)
	}
	for index, volume := range pod.Volumes {
		if volume.HostPath != nil {
			return UsageError("%s.volumes[%d].hostPath is not supported by testkube local", path, index)
		}
	}
	return nil
}

func validateContainerSecurity(container *testworkflowsv1.ContainerConfig, path string) error {
	if container == nil || container.SecurityContext == nil {
		return nil
	}
	securityContext := container.SecurityContext
	if securityContext.Privileged != nil && *securityContext.Privileged {
		return UsageError("%s.securityContext.privileged is not supported by testkube local", path)
	}
	if securityContext.AllowPrivilegeEscalation != nil && *securityContext.AllowPrivilegeEscalation {
		return UsageError("%s.securityContext.allowPrivilegeEscalation is not supported by testkube local", path)
	}
	if securityContext.Capabilities != nil && len(securityContext.Capabilities.Add) > 0 {
		return UsageError("%s.securityContext.capabilities.add is not supported by testkube local", path)
	}
	if securityContext.ProcMount != nil && *securityContext.ProcMount == "Unmasked" {
		return UsageError("%s.securityContext.procMount Unmasked is not supported by testkube local", path)
	}
	return nil
}

// ValidateSourceMountAvailable rejects a local source mount that would
// collide with an existing top-level tarball before the source relay is
// created. It is deliberately separate from RewriteWorkflowWithSource so
// Prepare can perform the check before mutating Kubernetes.
func ValidateSourceMountAvailable(workflow *testworkflowsv1.TestWorkflow, mountPath string) error {
	if workflow == nil {
		return UsageError("workflow is required")
	}
	if _, err := ResolveSourceMount(workflow, mountPath); err != nil {
		return err
	}
	if workflow.Spec.Content == nil {
		return nil
	}
	for _, tarball := range workflow.Spec.Content.Tarball {
		if tarball.Path == mountPath {
			return UsageError("spec.content.tarball path %q conflicts with --source-mount", mountPath)
		}
	}
	return nil
}
