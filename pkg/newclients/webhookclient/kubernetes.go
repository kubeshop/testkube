package webhookclient

import (
	"context"
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

	executorsv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

var _ WebhookClient = &k8sWebhookClient{}

type k8sWebhookClient struct {
	client         client.Client
	restClient     rest.Interface
	parameterCodec runtime.ParameterCodec
	namespace      string
}

func NewKubernetesWebhookClient(client client.Client, restConfig *rest.Config, namespace string) (WebhookClient, error) {
	// Build the scheme
	scheme := runtime.NewScheme()
	if err := metav1.AddMetaToScheme(scheme); err != nil {
		return nil, err
	}
	if err := executorsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	codecs := serializer.NewCodecFactory(scheme)
	parameterCodec := runtime.NewParameterCodec(scheme)

	// Build the REST client
	cfg := *restConfig
	gv := executorsv1.GroupVersion
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

	return &k8sWebhookClient{
		client:         client,
		restClient:     restClient,
		parameterCodec: parameterCodec,
		namespace:      namespace,
	}, nil
}

func (c *k8sWebhookClient) get(ctx context.Context, name string) (*executorsv1.Webhook, error) {
	webhook := executorsv1.Webhook{}
	opts := client.ObjectKey{Namespace: c.namespace, Name: name}
	if err := c.client.Get(ctx, opts, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (c *k8sWebhookClient) Get(ctx context.Context, environmentId string, name string) (*testkube.Webhook, error) {
	webhook, err := c.get(ctx, name)
	if err != nil {
		return nil, err
	}
	apiWebhook := webhooks.MapCRDToAPI(*webhook)
	return &apiWebhook, nil
}

func (c *k8sWebhookClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	webhook, err := c.get(ctx, name)
	if err != nil {
		return "", err
	}
	if webhook != nil {
		return webhook.UID, nil
	}
	return "", nil
}

func (c *k8sWebhookClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.Webhook, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range options.Labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	list := &executorsv1.WebhookList{}
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

	result := make([]testkube.Webhook, 0)
	for i := range list.Items {
		if options.TextSearch != "" && !strings.Contains(strings.ToLower(list.Items[i].Name), options.TextSearch) {
			continue
		}
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, webhooks.MapCRDToAPI(list.Items[i]))
		limit--
		if limit == 0 {
			break
		}
	}
	return result, nil
}

func (c *k8sWebhookClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	labels := map[string][]string{}
	list := &executorsv1.WebhookList{}
	err := c.client.List(ctx, list, &client.ListOptions{Namespace: c.namespace})
	if err != nil {
		return labels, err
	}

	for _, webhook := range list.Items {
		for key, value := range webhook.Labels {
			if !slices.Contains(labels[key], value) {
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

func (c *k8sWebhookClient) Update(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	original, err := c.get(ctx, webhook.Name)
	if err != nil {
		return err
	}
	next := webhooks.MapAPIToCRD(webhook)
	next.Name = original.Name
	next.Namespace = c.namespace
	next.ResourceVersion = original.ResourceVersion
	return c.client.Update(ctx, &next)
}

func (c *k8sWebhookClient) UpdateStatus(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	original, err := c.get(ctx, webhook.Name)
	if err != nil {
		return err
	}

	return c.client.Status().Update(ctx, original)
}

func (c *k8sWebhookClient) Create(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	next := webhooks.MapAPIToCRD(webhook)
	next.Namespace = c.namespace
	return c.client.Create(ctx, &next)
}

func (c *k8sWebhookClient) Delete(ctx context.Context, environmentId string, name string) error {
	original, err := c.get(ctx, name)
	if err != nil {
		return err
	}
	original.Namespace = c.namespace
	return c.client.Delete(ctx, original)
}

func (c *k8sWebhookClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	labelSelector := labels2.NewSelector()
	for k, v := range labels {
		req, _ := labels2.NewRequirement(k, selection.Equals, []string{v})
		labelSelector = labelSelector.Add(*req)
	}

	u := &unstructured.Unstructured{}
	u.SetKind("Webhook")
	u.SetAPIVersion(executorsv1.GroupVersion.String())
	err := c.client.DeleteAllOf(ctx, u,
		client.InNamespace(c.namespace),
		client.MatchingLabelsSelector{Selector: labelSelector})
	return math.MaxInt32, err
}

func (c *k8sWebhookClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	// Load initial data
	list := &executorsv1.WebhookList{}
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
		Resource("webhooks").
		VersionedParams(&opts, c.parameterCodec).
		Watch(ctx)
	if err != nil {
		return channels.NewError[Update](err)
	}
	result := channels.NewWatcher[Update]()
	go func() {
		// Send initial data
		for _, k8sObject := range list.Items {
			obj := webhooks.MapCRDToAPI(k8sObject)
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
			k8SObject, ok := event.Object.(*executorsv1.Webhook)
			if !ok || k8SObject == nil {
				continue
			}

			// Handle Kubernetes flavours that do not have Deleted event
			if k8SObject.DeletionTimestamp != nil {
				event.Type = watch.Deleted
			}

			obj := webhooks.MapCRDToAPI(*k8SObject)

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
