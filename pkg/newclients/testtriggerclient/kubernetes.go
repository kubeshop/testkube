package testtriggerclient

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testtriggers"
	testtriggersclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testtriggers/v1"
)

var _ TestTriggerClient = &k8sTestTriggerClient{}

type k8sTestTriggerClient struct {
	client testtriggersclientv1.Interface
}

func NewKubernetesTestTriggerClient(client testtriggersclientv1.Interface) TestTriggerClient {
	return &k8sTestTriggerClient{client: client}
}

func (c *k8sTestTriggerClient) Get(ctx context.Context, environmentId string, name string, namespace string) (*testkube.TestTrigger, error) {
	crd, err := c.client.Get(name, namespace)
	if err != nil {
		return nil, err
	}
	apiTrigger := testtriggers.MapCRDToAPI(crd)
	return &apiTrigger, nil
}

func (c *k8sTestTriggerClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string, namespace string) (types.UID, error) {
	// This would need to be implemented to get the actual Kubernetes object UID
	// For now, return empty as this is primarily used for cloud scenarios
	return "", nil
}

func (c *k8sTestTriggerClient) List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.TestTrigger, error) {
	selector := ""
	if options.Selector != "" {
		selector = options.Selector
	}

	list, err := c.client.List(selector, namespace)
	if err != nil {
		return nil, err
	}

	return testtriggers.MapTestTriggerListKubeToAPI(list), nil
}

func (c *k8sTestTriggerClient) ListLabels(ctx context.Context, environmentId string, namespace string) (map[string][]string, error) {
	// This would need to be implemented by querying all triggers and extracting labels
	// For now, return empty map
	return make(map[string][]string), nil
}

func (c *k8sTestTriggerClient) Update(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	// Get the existing CRD to preserve its metadata (including ResourceVersion)
	existingCRD, err := c.client.Get(trigger.Name, trigger.Namespace)
	if err != nil {
		return err
	}

	// Convert testkube.TestTrigger to TestTriggerUpsertRequest for mapping
	upsertRequest := testkube.TestTriggerUpsertRequest{
		Name:              trigger.Name,
		Namespace:         trigger.Namespace,
		Labels:            trigger.Labels,
		Annotations:       trigger.Annotations,
		Selector:          trigger.Selector,
		Resource:          trigger.Resource,
		ResourceSelector:  trigger.ResourceSelector,
		Event:             trigger.Event,
		ConditionSpec:     trigger.ConditionSpec,
		ProbeSpec:         trigger.ProbeSpec,
		Action:            trigger.Action,
		ActionParameters:  trigger.ActionParameters,
		Execution:         trigger.Execution,
		TestSelector:      trigger.TestSelector,
		ConcurrencyPolicy: trigger.ConcurrencyPolicy,
		Disabled:          trigger.Disabled,
	}

	// Use the new mapper function that preserves existing metadata
	crd := testtriggers.MapTestTriggerUpsertRequestToTestTriggerCRDWithExistingMeta(upsertRequest, existingCRD.ObjectMeta)
	_, err = c.client.Update(&crd)
	return err
}

func (c *k8sTestTriggerClient) Create(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	// Convert testkube.TestTrigger to TestTriggerUpsertRequest for mapping
	upsertRequest := testkube.TestTriggerUpsertRequest{
		Name:              trigger.Name,
		Namespace:         trigger.Namespace,
		Labels:            trigger.Labels,
		Annotations:       trigger.Annotations,
		Resource:          trigger.Resource,
		ResourceSelector:  trigger.ResourceSelector,
		Event:             trigger.Event,
		ConditionSpec:     trigger.ConditionSpec,
		ProbeSpec:         trigger.ProbeSpec,
		Action:            trigger.Action,
		ActionParameters:  trigger.ActionParameters,
		Execution:         trigger.Execution,
		TestSelector:      trigger.TestSelector,
		ConcurrencyPolicy: trigger.ConcurrencyPolicy,
		Disabled:          trigger.Disabled,
	}

	crd := testtriggers.MapTestTriggerUpsertRequestToTestTriggerCRD(upsertRequest)
	_, err := c.client.Create(&crd)
	return err
}

func (c *k8sTestTriggerClient) Delete(ctx context.Context, environmentId string, name string, namespace string) error {
	return c.client.Delete(name, namespace)
}

func (c *k8sTestTriggerClient) DeleteAll(ctx context.Context, environmentId string, namespace string) (uint32, error) {
	err := c.client.DeleteAll(namespace)
	if err != nil {
		return 0, err
	}
	// The old client doesn't return count, so we return 0
	// This could be improved by first listing and counting
	return 0, nil
}

func (c *k8sTestTriggerClient) DeleteByLabels(ctx context.Context, environmentId string, selector string, namespace string) (uint32, error) {
	err := c.client.DeleteByLabels(selector, namespace)
	if err != nil {
		return 0, err
	}
	// The old client doesn't return count, so we return 0
	// This could be improved by first listing and counting
	return 0, nil
}
