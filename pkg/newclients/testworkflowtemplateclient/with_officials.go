package testworkflowtemplateclient

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"slices"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/k8s/templates"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

var errCannotModify = errors.New("cannot modify official templates")
var errCannotDelete = errors.New("cannot delete official templates")
var errDuplicate = errors.New("already exists")

var _ TestWorkflowTemplateClient = &testWorkflowTemplateClientWithBuildIn{}

type testWorkflowTemplateClientWithBuildIn struct {
	client    TestWorkflowTemplateClient
	officials []testkube.TestWorkflowTemplate
}

// newTestWorkflowTemplateClientWithOfficials wraps another TestWorkflowTemplateClient and adds build-in official officials to it when they do not yet exist.
func newTestWorkflowTemplateClientWithOfficials(client TestWorkflowTemplateClient) TestWorkflowTemplateClient {
	officialTemplates, err := parseAllOfficialTemplates()
	if err != nil {
		log.DefaultLogger.Warnf("cannot load official templates: %s", err.Error())
		// Silently continue…
	}

	return &testWorkflowTemplateClientWithBuildIn{client: client, officials: officialTemplates}
}

func parseAllOfficialTemplates() ([]testkube.TestWorkflowTemplate, error) {
	var officialTemplates []testkube.TestWorkflowTemplate
	entries, err := templates.Templates.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		t, err := parseOfficialTemplate(entry)
		if err != nil {
			log.DefaultLogger.Warnf("skipping official template which cannot be parsed: %s, %s", entry.Name(), err.Error())
			continue
		}
		officialTemplates = append(officialTemplates, *t)
	}

	return officialTemplates, nil
}

func parseOfficialTemplate(e fs.DirEntry) (*testkube.TestWorkflowTemplate, error) {
	if e.IsDir() {
		// Unexpected and due to incorrect usage of our `templates` directory. Please keep it flat!
		return nil, errors.New("expected entry to not be a directory")
	}

	file, err := templates.Templates.ReadFile(e.Name())
	if err != nil {
		return nil, err
	}

	var template v1.TestWorkflowTemplate
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(file), len(file))
	if err := decoder.Decode(&template); err != nil {
		return nil, err
	}

	result := testworkflows.MapTemplateKubeToAPI(&template)
	return result, nil
}

func (c *testWorkflowTemplateClientWithBuildIn) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflowTemplate, error) {
	for _, template := range c.officials {
		if template.Name == name {
			return &template, nil
		}
	}

	return c.client.Get(ctx, environmentId, name)
}

func (c *testWorkflowTemplateClientWithBuildIn) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	return c.client.GetKubernetesObjectUID(ctx, environmentId, name)
}

func (c *testWorkflowTemplateClientWithBuildIn) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflowTemplate, error) {
	result, err := c.client.List(ctx, environmentId, options)
	if err != nil {
		return result, err
	}

	// Avoid duplicates. We should be able to remove this over time once we
	// 1. Kubernetes client: Remove official templates from Helm Charts.
	// 2. Cloud client: Run migration to remove official templates that were synced by GitOps agent.
	for _, o := range c.officials {
		if slices.ContainsFunc(result, templateWithName(o.Name)) {
			continue
		}
		result = append(result, o)
	}

	return result, nil
}

func (c *testWorkflowTemplateClientWithBuildIn) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	labels, err := c.client.ListLabels(ctx, environmentId)
	if err != nil {
		return labels, err
	}

	for _, template := range c.officials {
		for key, value := range template.Labels {
			if !slices.Contains(labels[key], value) {
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

func (c *testWorkflowTemplateClientWithBuildIn) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	if slices.ContainsFunc(c.officials, templateWithName(workflow.Name)) {
		return errCannotModify
	}

	return c.client.Update(ctx, environmentId, workflow)
}

func (c *testWorkflowTemplateClientWithBuildIn) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	if slices.ContainsFunc(c.officials, templateWithName(workflow.Name)) {
		return errDuplicate
	}

	return c.client.Create(ctx, environmentId, workflow)
}

func (c *testWorkflowTemplateClientWithBuildIn) Delete(ctx context.Context, environmentId string, name string) error {
	if slices.ContainsFunc(c.officials, templateWithName(name)) {
		return errCannotDelete
	}

	return c.client.Delete(ctx, environmentId, name)
}

func (c *testWorkflowTemplateClientWithBuildIn) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	return c.client.DeleteByLabels(ctx, environmentId, labels)
}

func (c *testWorkflowTemplateClientWithBuildIn) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	return c.client.WatchUpdates(ctx, environmentId, includeInitialData)
}

func templateWithName(name string) func(testkube.TestWorkflowTemplate) bool {
	return func(template testkube.TestWorkflowTemplate) bool {
		return template.Name == name
	}
}
