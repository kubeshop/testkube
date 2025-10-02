/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"time"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/operator/informers/externalversions/internalinterfaces"
	executorlisterv1 "github.com/kubeshop/testkube/pkg/operator/listers/executor/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// WebhookInformer provides access to a shared informer and lister for Webhook.
type WebhookInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() executorlisterv1.WebhookLister
}

type webhookInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewWebhookInformer constructs a new informer for Webhook type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewWebhookInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredWebhookInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredWebhookInformer constructs a new informer for Webhook type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewFilteredWebhookInformer(
	client versioned.Interface,
	namespace string,
	resyncPeriod time.Duration,
	indexers cache.Indexers,
	tweakListOptions internalinterfaces.TweakListOptionsFunc,
) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExecutorV1().Webhook(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExecutorV1().Webhook(namespace).Watch(context.TODO(), options)
			},
		},
		&executorv1.Webhook{},
		resyncPeriod,
		indexers,
	)
}

func (f *webhookInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredWebhookInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *webhookInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&executorv1.Webhook{}, f.defaultInformer)
}

func (f *webhookInformer) Lister() executorlisterv1.WebhookLister {
	return executorlisterv1.NewWebhookLister(f.Informer().GetIndexer())
}
