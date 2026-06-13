package crd

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestExecuteTemplateQuotesSelectorRegexFields(t *testing.T) {
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

func TestExecuteTemplateRendersContentSelectorGit(t *testing.T) {
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
		TestSelector: &testkube.TestTriggerSelector{
			Name: "sample-test",
		},
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/example/repo.git",
				Branches: []string{"main"},
				UsernameFrom: &testkube.EnvVarSource{
					SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "git-creds", Key: "username"},
				},
				PullRequest: &testkube.TestTriggerContentGitPullRequest{Types: []string{"opened"}},
			},
		},
	}

	output, err := ExecuteTemplate(TemplateTestTrigger, trigger)
	if err != nil {
		t.Fatalf("execute template: %v", err)
	}

	var parsed struct {
		Spec struct {
			ContentSelector struct {
				Git struct {
					Uri          string   `yaml:"uri"`
					Branches     []string `yaml:"branches"`
					UsernameFrom struct {
						SecretKeyRef struct {
							Name string `yaml:"name"`
							Key  string `yaml:"key"`
						} `yaml:"secretKeyRef"`
					} `yaml:"usernameFrom"`
					PullRequest struct {
						Types []string `yaml:"types"`
					} `yaml:"pullRequest"`
				} `yaml:"git"`
			} `yaml:"contentSelector"`
		} `yaml:"spec"`
	}

	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("unmarshal rendered YAML: %v", err)
	}

	if parsed.Spec.ContentSelector.Git.Uri != "https://github.com/example/repo.git" {
		t.Fatalf("expected contentSelector.git.uri to round-trip, got %q", parsed.Spec.ContentSelector.Git.Uri)
	}

	if len(parsed.Spec.ContentSelector.Git.Branches) != 1 || parsed.Spec.ContentSelector.Git.Branches[0] != "main" {
		t.Fatalf("expected contentSelector.git.branches to round-trip, got %v", parsed.Spec.ContentSelector.Git.Branches)
	}

	if parsed.Spec.ContentSelector.Git.UsernameFrom.SecretKeyRef.Name != "git-creds" || parsed.Spec.ContentSelector.Git.UsernameFrom.SecretKeyRef.Key != "username" {
		t.Fatalf("expected contentSelector.git.usernameFrom.secretKeyRef to round-trip, got name=%q key=%q", parsed.Spec.ContentSelector.Git.UsernameFrom.SecretKeyRef.Name, parsed.Spec.ContentSelector.Git.UsernameFrom.SecretKeyRef.Key)
	}

	if len(parsed.Spec.ContentSelector.Git.PullRequest.Types) != 1 || parsed.Spec.ContentSelector.Git.PullRequest.Types[0] != "opened" {
		t.Fatalf("expected contentSelector.git.pullRequest.types to round-trip, got %v", parsed.Spec.ContentSelector.Git.PullRequest.Types)
	}
}
