package workflowtriggerclient

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/workflowtriggers"
)

var _ WorkflowTriggerClient = &k8sWorkflowTriggerClient{}

type k8sWorkflowTriggerClient struct {
	client    crclient.Client
	namespace string
}

// NewKubernetesWorkflowTriggerClient wraps a controller-runtime client with the
// WorkflowTriggerClient interface. The runtime client must have the
// workflowtriggersv1 scheme registered.
func NewKubernetesWorkflowTriggerClient(c crclient.Client, namespace string) WorkflowTriggerClient {
	return &k8sWorkflowTriggerClient{client: c, namespace: namespace}
}

func (c *k8sWorkflowTriggerClient) resolveNamespace(ns string) string {
	if ns != "" {
		return ns
	}
	return c.namespace
}

func (c *k8sWorkflowTriggerClient) Get(ctx context.Context, _ string, name, namespace string) (*testkube.WorkflowTrigger, error) {
	var crd workflowtriggersv1.WorkflowTrigger
	if err := c.client.Get(ctx, crclient.ObjectKey{Namespace: c.resolveNamespace(namespace), Name: name}, &crd); err != nil {
		return nil, err
	}
	api := workflowtriggers.MapCRDToAPI(&crd)
	return &api, nil
}

func (c *k8sWorkflowTriggerClient) List(ctx context.Context, _ string, options ListOptions, namespace string) ([]testkube.WorkflowTrigger, error) {
	opts := []crclient.ListOption{crclient.InNamespace(c.resolveNamespace(namespace))}
	if options.Selector != "" {
		sel, err := labels.Parse(options.Selector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, crclient.MatchingLabelsSelector{Selector: sel})
	}

	var list workflowtriggersv1.WorkflowTriggerList
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return workflowtriggers.MapListCRDToAPI(&list), nil
}

func (c *k8sWorkflowTriggerClient) Create(ctx context.Context, _ string, trigger testkube.WorkflowTrigger) error {
	if trigger.Namespace == "" {
		trigger.Namespace = c.namespace
	}
	crd := workflowtriggers.MapAPIToCRD(trigger)
	return c.client.Create(ctx, &crd)
}

func (c *k8sWorkflowTriggerClient) Update(ctx context.Context, _ string, trigger testkube.WorkflowTrigger) error {
	// Fill in the default namespace so the mapped CRD and the Get lookup agree
	// — client.Update on a namespaced resource fails if .Namespace is empty.
	trigger.Namespace = c.resolveNamespace(trigger.Namespace)
	// Preserve ResourceVersion for optimistic-concurrency on updates.
	var existing workflowtriggersv1.WorkflowTrigger
	if err := c.client.Get(ctx, crclient.ObjectKey{Namespace: trigger.Namespace, Name: trigger.Name}, &existing); err != nil {
		return err
	}
	crd := workflowtriggers.MapAPIToCRD(trigger)
	crd.ResourceVersion = existing.ResourceVersion
	return c.client.Update(ctx, &crd)
}

func (c *k8sWorkflowTriggerClient) Delete(ctx context.Context, _ string, name, namespace string) error {
	obj := &workflowtriggersv1.WorkflowTrigger{}
	obj.Name = name
	obj.Namespace = c.resolveNamespace(namespace)
	return c.client.Delete(ctx, obj)
}

func (c *k8sWorkflowTriggerClient) DeleteAll(ctx context.Context, _ string, namespace string) (uint32, error) {
	u := &unstructured.Unstructured{}
	u.SetKind(workflowtriggersv1.Kind)
	u.SetAPIVersion(workflowtriggersv1.GroupVersion.String())
	return 0, c.client.DeleteAllOf(ctx, u, crclient.InNamespace(c.resolveNamespace(namespace)))
}

func (c *k8sWorkflowTriggerClient) DeleteByLabels(ctx context.Context, _ string, selector, namespace string) (uint32, error) {
	sel, err := labels.Parse(selector)
	if err != nil {
		return 0, err
	}
	u := &unstructured.Unstructured{}
	u.SetKind(workflowtriggersv1.Kind)
	u.SetAPIVersion(workflowtriggersv1.GroupVersion.String())
	return 0, c.client.DeleteAllOf(ctx, u,
		crclient.InNamespace(c.resolveNamespace(namespace)),
		crclient.MatchingLabelsSelector{Selector: sel},
	)
}
