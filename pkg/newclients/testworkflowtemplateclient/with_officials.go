package testworkflowtemplateclient

import (
	"context"
	"errors"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/k8s/templates"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

var errCannotModify = errors.New("cannot modify official templates")
var errCannotDelete = errors.New("cannot delete official templates")
var errDuplicate = errors.New("already exists")

var _ TestWorkflowTemplateClient = &testWorkflowTemplateClientWithBuildIn{}

type testWorkflowTemplateClientWithBuildIn struct {
	client    TestWorkflowTemplateClient
	officials []testkube.TestWorkflowTemplate
}

// NewTestWorkflowTemplateClientWithOfficials wraps another TestWorkflowTemplateClient and adds build-in official officials to it when they do not yet exist.
func NewTestWorkflowTemplateClientWithOfficials(client TestWorkflowTemplateClient) TestWorkflowTemplateClient {
	officialTemplates, skipped, err := templates.ParseAllOfficialTemplates()
	if err != nil {
		log.DefaultLogger.Warnf("cannot load official templates: %s", err.Error())
		// Silently continue…
	}
	if len(skipped) > 0 {
		log.DefaultLogger.Warnf("skipped official templates: %s", strings.Join(skipped, ","))
		// Silently continue…
	}

	return &testWorkflowTemplateClientWithBuildIn{client: client, officials: officialTemplates}
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

	// note: duplicates will always be possible as we might have e.g. OfficialWrapper -> CloudClient -> OfficialWrapper -> MongoRepo.
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
