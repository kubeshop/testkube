package localrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestLoadWorkflowRequiresOneObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.yaml")
	if err := os.WriteFile(path, []byte("apiVersion: testworkflows.testkube.io/v1\nkind: TestWorkflow\nmetadata:\n  name: one\nspec: {}\n---\napiVersion: testworkflows.testkube.io/v1\nkind: TestWorkflow\nmetadata:\n  name: two\nspec: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadWorkflow(path); err == nil {
		t.Fatal("LoadWorkflow unexpectedly accepted two YAML documents")
	}
}

func TestValidateSupportedRejectsNestedGitWithSource(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "local"},
		Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{
			StepSource: testworkflowsv1.StepSource{Content: &testworkflowsv1.Content{Git: &testworkflowsv1.ContentGit{Uri: "https://example.invalid/repo.git"}}},
		}}},
	}
	if err := ValidateSupported(workflow, true, true, false); err == nil {
		t.Fatal("ValidateSupported unexpectedly accepted nested git content with --source")
	}
}

func TestValidateSupportedRejectsNonInteractivePause(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "local"},
		Spec:       testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{StepControl: testworkflowsv1.StepControl{Paused: true}, StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}}}},
	}
	if err := ValidateSupported(workflow, false, false, false); err == nil {
		t.Fatal("ValidateSupported unexpectedly accepted a noninteractive pause")
	}
	if err := ValidateSupported(workflow, false, false, true); err != nil {
		t.Fatalf("ValidateSupported with --auto-continue: %v", err)
	}
}

func TestValidateSupportedRejectsUnsafeOrRedirectingWorkflowFields(t *testing.T) {
	trueValue := true
	falseValue := false
	for _, test := range []struct {
		name     string
		workflow *testworkflowsv1.TestWorkflow
		expected string
	}{
		{
			name: "other namespace",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Job: &testworkflowsv1.JobConfig{Namespace: "another-namespace"}},
			}},
			expected: "spec.job.namespace",
		},
		{
			name: "host pid even false",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Pod: &testworkflowsv1.PodConfig{HostPID: &falseValue}},
			}},
			expected: "spec.pod.hostPID",
		},
		{
			name: "host path",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Pod: &testworkflowsv1.PodConfig{Volumes: []corev1.Volume{{Name: "host", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}}}}},
			}},
			expected: "spec.pod.volumes[0].hostPath",
		},
		{
			name: "privileged root container",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Container: &testworkflowsv1.ContainerConfig{SecurityContext: &testworkflowsv1.WorkflowSecurityContext{Privileged: &trueValue}}},
			}},
			expected: "spec.container.securityContext.privileged",
		},
		{
			name: "privilege escalation",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Container: &testworkflowsv1.ContainerConfig{SecurityContext: &testworkflowsv1.WorkflowSecurityContext{AllowPrivilegeEscalation: &trueValue}}},
			}},
			expected: "spec.container.securityContext.allowPrivilegeEscalation",
		},
		{
			name: "added capability",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Container: &testworkflowsv1.ContainerConfig{SecurityContext: &testworkflowsv1.WorkflowSecurityContext{Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"SYS_ADMIN"}}}}},
			}},
			expected: "spec.container.securityContext.capabilities.add",
		},
		{
			name: "privileged nested step container",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{
				StepDefaults: testworkflowsv1.StepDefaults{Container: &testworkflowsv1.ContainerConfig{SecurityContext: &testworkflowsv1.WorkflowSecurityContext{Privileged: &trueValue}}},
			}}}},
			expected: "spec.steps[0].container.securityContext.privileged",
		},
		{
			name: "privileged run container",
			workflow: &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{
				StepOperations: testworkflowsv1.StepOperations{Run: &testworkflowsv1.StepRun{ContainerConfig: testworkflowsv1.ContainerConfig{SecurityContext: &testworkflowsv1.WorkflowSecurityContext{Privileged: &trueValue}}}},
			}}}},
			expected: "spec.steps[0].run.securityContext.privileged",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSupportedInNamespace(test.workflow, DefaultNamespace, false, true, false)
			if err == nil {
				t.Fatal("ValidateSupportedInNamespace unexpectedly accepted unsupported field")
			}
			if !IsUsageError(err) {
				t.Fatalf("ValidateSupportedInNamespace returned non-usage error: %v", err)
			}
			if !strings.Contains(err.Error(), test.expected) {
				t.Fatalf("ValidateSupportedInNamespace error %q does not name %q", err, test.expected)
			}
		})
	}
}

func TestValidateSourceMountAvailableRejectsCollisionBeforeRelayCreation(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
		TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{Content: &testworkflowsv1.Content{Tarball: []testworkflowsv1.ContentTarball{{Path: "/data/repo"}}}},
	}}
	err := ValidateSourceMountAvailable(workflow, "/data/repo")
	if err == nil {
		t.Fatal("ValidateSourceMountAvailable unexpectedly accepted a tarball collision")
	}
	if !IsUsageError(err) || !strings.Contains(err.Error(), "spec.content.tarball path") {
		t.Fatalf("ValidateSourceMountAvailable returned unexpected error: %v", err)
	}
}
