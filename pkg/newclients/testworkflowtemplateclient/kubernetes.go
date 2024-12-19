package testworkflowtemplateclient

import (
	"context"
	"math"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	labels2 "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

var _ TestWorkflowTemplateClient = &k8sTestWorkflowTemplateClient{}

type k8sTestWorkflowTemplateClient struct {
	client    client.Client
	namespace string
}

func NewKubernetesTestWorkflowTemplateClient(client client.Client, namespace string) TestWorkflowTemplateClient {
	return &k8sTestWorkflowTemplateClient{client: client, namespace: namespace}
}

func (c *k8sTestWorkflowTemplateClient) get(ctx context.Context, name string) (*testworkflowsv1.TestWorkflowTemplate, error) {
	template := testworkflowsv1.TestWorkflowTemplate{}
	opts := client.ObjectKey{Namespace: c.namespace, Name: name}
	if err := c.client.Get(ctx, opts, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

func (c *k8sTestWorkflowTemplateClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflowTemplate, error) {
	template, err := c.get(ctx, name)
	if err != nil {
		return nil, err
	}
	return testworkflows.MapTemplateKubeToAPI(template), nil
}

func (c *k8sTestWorkflowTemplateClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflowTemplate, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range options.Labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	list := &testworkflowsv1.TestWorkflowTemplateList{}
	opts := &client.ListOptions{Namespace: c.namespace, LabelSelector: labelSelector}
	if options.Limit != 0 && options.TextSearch == "" {
		opts.Limit = int64(options.Offset + options.Limit)
	}
	if err := c.client.List(ctx, list, opts); err != nil {
		return nil, err
	}

	offset := options.Offset
	limit := options.Limit
	if limit == 0 {
		limit = math.MaxUint32
	}
	options.TextSearch = strings.ToLower(options.TextSearch)

	result := make([]testkube.TestWorkflowTemplate, 0)
	for i := range list.Items {
		if options.TextSearch != "" && !strings.Contains(strings.ToLower(list.Items[i].Name), options.TextSearch) {
			continue
		}
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, *testworkflows.MapTemplateKubeToAPI(&list.Items[i]))
		limit--
		if limit == 0 {
			break
		}
	}
	return result, nil
}

func (c *k8sTestWorkflowTemplateClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testworkflowsv1.TestWorkflowTemplateList{}
	err := c.client.List(ctx, list, &client.ListOptions{Namespace: c.namespace})
	if err != nil {
		return labels, err
	}

	for _, template := range list.Items {
		for key, value := range template.Labels {
			if !slices.Contains(labels[key], value) {
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

func (c *k8sTestWorkflowTemplateClient) Update(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error {
	original, err := c.get(ctx, template.Name)
	if err != nil {
		return err
	}
	next := testworkflows.MapTemplateAPIToKube(&template)
	next.Name = original.Name
	next.Namespace = c.namespace
	next.ResourceVersion = original.ResourceVersion
	return c.client.Update(ctx, next)
}

func (c *k8sTestWorkflowTemplateClient) Create(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error {
	next := testworkflows.MapTemplateAPIToKube(&template)
	next.Namespace = c.namespace
	return c.client.Create(ctx, next)
}

func (c *k8sTestWorkflowTemplateClient) Delete(ctx context.Context, environmentId string, name string) error {
	original, err := c.get(ctx, name)
	if err != nil {
		return err
	}
	original.Namespace = c.namespace
	return c.client.Delete(ctx, original)
}

func (c *k8sTestWorkflowTemplateClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflowTemplate")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	err := c.client.DeleteAllOf(ctx, u,
		client.InNamespace(c.namespace),
		client.MatchingLabelsSelector{Selector: labelSelector})
	// TODO: Consider if it's possible to return count
	return math.MaxInt32, err
}
