package crd

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestExecuteTemplateQuotesNamespaceRegex(t *testing.T) {
	t.Parallel()

	resource := testkube.CONTENT_TestTriggerResources
	action := testkube.RUN_TestTriggerActions
	execution := testkube.TEST_TestTriggerExecutions

	trigger := testkube.TestTrigger{
		Name:      "sample-trigger",
		Namespace: "testkube",
		Resource:  &resource,
		Event:     "modified",
		Action:    &action,
		Execution: &execution,
		ResourceSelector: &testkube.TestTriggerSelector{
			NameRegex:      "*resource",
			NamespaceRegex: "*prod",
		},
		TestSelector: &testkube.TestTriggerSelector{
			Name:           "sample-test",
			NameRegex:      "*name",
			NamespaceRegex: "*tests",
		},
	}

	output, err := ExecuteTemplate(TemplateTestTrigger, trigger)
	if err != nil {
		t.Fatalf("execute template: %v", err)
	}

	if want := "namespaceRegex: \"*prod\""; !strings.Contains(output, want) {
		t.Fatalf("expected rendered YAML to contain %q, got:\n%s", want, output)
	}

	if want := "namespaceRegex: \"*tests\""; !strings.Contains(output, want) {
		t.Fatalf("expected rendered YAML to contain %q, got:\n%s", want, output)
	}

	if want := "nameRegex: \"*resource\""; !strings.Contains(output, want) {
		t.Fatalf("expected rendered YAML to contain %q, got:\n%s", want, output)
	}

	if want := "nameRegex: \"*name\""; !strings.Contains(output, want) {
		t.Fatalf("expected rendered YAML to contain %q, got:\n%s", want, output)
	}

	var parsed struct {
		Spec struct {
			ResourceSelector struct {
				NameRegex      string `yaml:"nameRegex"`
				NamespaceRegex string `yaml:"namespaceRegex"`
			} `yaml:"resourceSelector"`
			TestSelector struct {
				NameRegex      string `yaml:"nameRegex"`
				NamespaceRegex string `yaml:"namespaceRegex"`
			} `yaml:"testSelector"`
		} `yaml:"spec"`
	}

	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("unmarshal rendered YAML: %v", err)
	}

	if parsed.Spec.ResourceSelector.NamespaceRegex != "*prod" {
		t.Fatalf("expected resourceSelector namespaceRegex to round-trip, got %q", parsed.Spec.ResourceSelector.NamespaceRegex)
	}

	if parsed.Spec.ResourceSelector.NameRegex != "*resource" {
		t.Fatalf("expected resourceSelector nameRegex to round-trip, got %q", parsed.Spec.ResourceSelector.NameRegex)
	}

	if parsed.Spec.TestSelector.NamespaceRegex != "*tests" {
		t.Fatalf("expected testSelector namespaceRegex to round-trip, got %q", parsed.Spec.TestSelector.NamespaceRegex)
	}

	if parsed.Spec.TestSelector.NameRegex != "*name" {
		t.Fatalf("expected testSelector nameRegex to round-trip, got %q", parsed.Spec.TestSelector.NameRegex)
	}
}
