package testworkflowtemplateclient

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	labels2 "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

const InlinedGlobalTemplateName = "<inline-global-template>"

var _ TestWorkflowTemplateClient = &k8sTestWorkflowTemplateClient{}

type k8sTestWorkflowTemplateClient struct {
	client                client.Client
	restClient            rest.Interface
	parameterCodec        runtime.ParameterCodec
	namespace             string
	inlinedGlobalTemplate string
}

func NewKubernetesTestWorkflowTemplateClient(client client.Client, restConfig *rest.Config, namespace string,
	disableOfficialTemplates bool, inlinedGlobalTemplate string) (TestWorkflowTemplateClient, error) {
	// Build the scheme
	scheme := runtime.NewScheme()
	if err := metav1.AddMetaToScheme(scheme); err != nil {
		return nil, err
	}
	if err := testworkflowsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	codecs := serializer.NewCodecFactory(scheme)
	parameterCodec := runtime.NewParameterCodec(scheme)

	// Build the REST client
	cfg := *restConfig
	gv := testworkflowsv1.GroupVersion
	cfg.GroupVersion = &gv
	cfg.APIPath = "/apis"
	cfg.NegotiatedSerializer = codecs.WithoutConversion()
	httpClient, err := rest.HTTPClientFor(&cfg)
	if err != nil {
		return nil, err
	}
	restClient, err := rest.RESTClientForConfigAndClient(&cfg, httpClient)
	if err != nil {
		return nil, err
	}

	c := &k8sTestWorkflowTemplateClient{
		client:                client,
		restClient:            restClient,
		parameterCodec:        parameterCodec,
		namespace:             namespace,
		inlinedGlobalTemplate: inlinedGlobalTemplate,
	}

	if disableOfficialTemplates {
		return c, nil
	}
	return NewTestWorkflowTemplateClientWithOfficials(c), nil
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
	if name == InlinedGlobalTemplateName {
		if c.inlinedGlobalTemplate != "" {
			globalTemplateInline := new(testworkflowsv1.TestWorkflowTemplate)
			err := crdcommon.DeserializeCRD(globalTemplateInline, []byte("spec:\n  "+strings.ReplaceAll(c.inlinedGlobalTemplate, "\n", "\n  ")))
			if err != nil {
				return nil, err
			}

			return testworkflows.MapTemplateKubeToAPI(globalTemplateInline), nil
		}

		return nil, errors.New("empty inlline template")
	}

	template, err := c.get(ctx, name)
	if err != nil {
		return nil, err
	}
	return testworkflows.MapTemplateKubeToAPI(template), nil
}

func (c *k8sTestWorkflowTemplateClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	template, err := c.get(ctx, name)
	if err != nil {
		return "", err
	}
	if template != nil {
		return template.UID, nil
	}
	return "", nil
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

func (c *k8sTestWorkflowTemplateClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	// Load initial data
	list := &testworkflowsv1.TestWorkflowTemplateList{}
	if includeInitialData {
		opts := &client.ListOptions{Namespace: c.namespace}
		if err := c.client.List(ctx, list, opts); err != nil {
			return channels.NewError[Update](err)
		}
	}

	// Start watching
	opts := metav1.ListOptions{Watch: true, ResourceVersion: list.ResourceVersion}
	watcher, err := c.restClient.Get().
		Namespace(c.namespace).
		Resource("testworkflowtemplates").
		VersionedParams(&opts, c.parameterCodec).
		Watch(ctx)
	if err != nil {
		return channels.NewError[Update](err)
	}

	result := channels.NewWatcher[Update]()
	go func() {
		// Send initial data
		for _, k8sObject := range list.Items {
			obj := testworkflows.MapTestWorkflowTemplateKubeToAPI(k8sObject)
			updateType := EventTypeCreate
			if !obj.Updated.Equal(obj.Created) {
				updateType = EventTypeUpdate
			}
			result.Send(Update{
				Type:      updateType,
				Timestamp: obj.Updated,
				Resource:  &obj,
			})
		}

		// Watch
		for event := range watcher.ResultChan() {
			// Continue watching if that's just a bookmark
			if event.Type == watch.Bookmark {
				continue
			}

			// Load the current Kubernetes object
			k8sObject, ok := event.Object.(*testworkflowsv1.TestWorkflowTemplate)
			if !ok || k8sObject == nil {
				continue
			}

			// Handle Kubernetes flavours that do not have Deleted event
			if k8sObject.DeletionTimestamp != nil {
				event.Type = watch.Deleted
			}

			obj := testworkflows.MapTestWorkflowTemplateKubeToAPI(*k8sObject)

			switch event.Type {
			case watch.Added:
				result.Send(Update{
					Type:      EventTypeCreate,
					Timestamp: obj.Updated,
					Resource:  &obj,
				})
			case watch.Modified:
				result.Send(Update{
					Type:      EventTypeUpdate,
					Timestamp: obj.Updated,
					Resource:  &obj,
				})
			case watch.Deleted:
				result.Send(Update{
					Type:      EventTypeDelete,
					Timestamp: obj.Updated,
					Resource:  &obj,
				})
			}
		}
		result.Close(context.Cause(ctx))
	}()
	return result
}
