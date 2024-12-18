package testworkflowclient

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

var _ TestWorkflowClient = &k8sTestWorkflowClient{}

type k8sTestWorkflowClient struct {
	client    client.Client
	namespace string
}

func NewKubernetesTestWorkflowClient(client client.Client, namespace string) TestWorkflowClient {
	return &k8sTestWorkflowClient{
		client:    client,
		namespace: namespace,
	}
}

func (c *k8sTestWorkflowClient) get(ctx context.Context, name string) (*testworkflowsv1.TestWorkflow, error) {
	workflow := testworkflowsv1.TestWorkflow{}
	opts := client.ObjectKey{Namespace: c.namespace, Name: name}
	if err := c.client.Get(ctx, opts, &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (c *k8sTestWorkflowClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error) {
	workflow, err := c.get(ctx, name)
	if err != nil {
		return nil, err
	}
	return testworkflows.MapKubeToAPI(workflow), nil
}

func (c *k8sTestWorkflowClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflow, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range options.Labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	list := &testworkflowsv1.TestWorkflowList{}
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

	result := make([]testkube.TestWorkflow, 0)
	for i := range list.Items {
		if options.TextSearch != "" && !strings.Contains(strings.ToLower(list.Items[i].Name), options.TextSearch) {
			continue
		}
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, *testworkflows.MapKubeToAPI(&list.Items[i]))
		limit--
		if limit == 0 {
			break
		}
	}
	return result, nil
}

func (c *k8sTestWorkflowClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testworkflowsv1.TestWorkflowList{}
	err := c.client.List(ctx, list, &client.ListOptions{Namespace: c.namespace})
	if err != nil {
		return labels, err
	}

	for _, workflow := range list.Items {
		for key, value := range workflow.Labels {
			if !slices.Contains(labels[key], value) {
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

func (c *k8sTestWorkflowClient) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	original, err := c.get(ctx, workflow.Name)
	if err != nil {
		return err
	}
	next := testworkflows.MapAPIToKube(&workflow)
	next.Name = original.Name
	next.Namespace = c.namespace
	next.ResourceVersion = original.ResourceVersion
	return c.client.Update(ctx, next)
}

func (c *k8sTestWorkflowClient) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	next := testworkflows.MapAPIToKube(&workflow)
	next.Namespace = c.namespace
	return c.client.Create(ctx, next)
}

func (c *k8sTestWorkflowClient) Delete(ctx context.Context, environmentId string, name string) error {
	original, err := c.get(ctx, name)
	if err != nil {
		return err
	}
	original.Namespace = c.namespace
	return c.client.Delete(ctx, original)
}

func (c *k8sTestWorkflowClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestWorkflow")
	u.SetAPIVersion(testworkflowsv1.GroupVersion.String())
	err := c.client.DeleteAllOf(ctx, u,
		client.InNamespace(c.namespace),
		client.MatchingLabelsSelector{Selector: labelSelector})
	// TODO: Consider if it's possible to return count
	return math.MaxInt32, err
}
