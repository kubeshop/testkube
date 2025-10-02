package tests

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/operator/client/common"
)

type Update struct {
	Type      common.EventType
	Timestamp time.Time
	Resource  *testsv3.Test
}

type WatcherUpdate common.Watcher[Update]

//go:generate mockgen -source=./rest.go -destination=./mock_rest.go -package=tests "github.com/kubeshop/testkube/pkg/operator/client/tests/v3" RESTInterface
type RESTInterface interface {
	WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) WatcherUpdate
}

// NewRESTClient creates new Test client
func NewRESTClient(client client.Client, restConfig *rest.Config, namespace string) (*TestsRESTClient, error) {
	// Build the scheme
	scheme := runtime.NewScheme()
	if err := metav1.AddMetaToScheme(scheme); err != nil {
		return nil, err
	}

	if err := testsv3.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}

	codecs := serializer.NewCodecFactory(scheme)
	parameterCodec := runtime.NewParameterCodec(scheme)

	// Build the REST client
	cfg := *restConfig
	gv := testsv3.GroupVersion
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

	return &TestsRESTClient{
		k8sClient:      client,
		restClient:     restClient,
		parameterCodec: parameterCodec,
		namespace:      namespace}, nil
}

// TestsRESTClient implements REST methods to work with Test
type TestsRESTClient struct {
	k8sClient      client.Client
	restClient     *rest.RESTClient
	parameterCodec runtime.ParameterCodec
	namespace      string
}

func (s TestsRESTClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) WatcherUpdate {
	// Load initial data
	list := &testsv3.TestList{}
	if includeInitialData {
		opts := &client.ListOptions{Namespace: s.namespace}
		if err := s.k8sClient.List(ctx, list, opts); err != nil {
			return common.NewError[Update](err)
		}
	}

	// Start watching
	opts := metav1.ListOptions{Watch: true, ResourceVersion: list.ResourceVersion}
	watcher, err := s.restClient.Get().
		Namespace(s.namespace).
		Resource("tests").
		VersionedParams(&opts, s.parameterCodec).
		Watch(ctx)
	if err != nil {
		return common.NewError[Update](err)
	}

	result := common.NewWatcher[Update]()
	go func() {
		// Send initial data
		for _, k8sObject := range list.Items {
			updateType := common.EventTypeCreate
			updateTime := common.GetUpdateTime(k8sObject.ObjectMeta)
			if !updateTime.Equal(k8sObject.CreationTimestamp.Time) {
				updateType = common.EventTypeUpdate
			}

			result.Send(Update{
				Type:      updateType,
				Timestamp: updateTime,
				Resource:  &k8sObject,
			})
		}

		// Watch
		for event := range watcher.ResultChan() {
			// Continue watching if that's just a bookmark
			if event.Type == watch.Bookmark {
				continue
			}

			// Load the current Kubernetes object
			k8SObject, ok := event.Object.(*testsv3.Test)
			if !ok || k8SObject == nil {
				continue
			}

			// Handle Kubernetes flavours that do not have Deleted event
			if k8SObject.DeletionTimestamp != nil {
				event.Type = watch.Deleted
			}

			updateTime := common.GetUpdateTime(k8SObject.ObjectMeta)
			switch event.Type {
			case watch.Added:
				result.Send(Update{
					Type:      common.EventTypeCreate,
					Timestamp: updateTime,
					Resource:  k8SObject,
				})
			case watch.Modified:
				result.Send(Update{
					Type:      common.EventTypeUpdate,
					Timestamp: updateTime,
					Resource:  k8SObject,
				})
			case watch.Deleted:
				result.Send(Update{
					Type:      common.EventTypeDelete,
					Timestamp: updateTime,
					Resource:  k8SObject,
				})
			}
		}

		result.Close(context.Cause(ctx))
	}()

	return result
}
