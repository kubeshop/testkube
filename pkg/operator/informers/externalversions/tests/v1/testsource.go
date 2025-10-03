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

	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/operator/informers/externalversions/internalinterfaces"
	testsourcelisterv1 "github.com/kubeshop/testkube/pkg/operator/listers/tests/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// TestSourceInformer provides access to a shared informer and lister for TestSource.
type TestSourceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() testsourcelisterv1.TestSourceLister
}

type testSourceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewTestSourceInformer constructs a new informer for TestSource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewTestSourceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTestSourceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredTestSourceInformer constructs a new informer for TestSource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewFilteredTestSourceInformer(
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
				return client.TestsV1().TestSource(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TestsV1().TestSource(namespace).Watch(context.TODO(), options)
			},
		},
		&testsourcev1.TestSource{},
		resyncPeriod,
		indexers,
	)
}

func (f *testSourceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTestSourceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *testSourceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&testsourcev1.TestSource{}, f.defaultInformer)
}

func (f *testSourceInformer) Lister() testsourcelisterv1.TestSourceLister {
	return testsourcelisterv1.NewTestSourceLister(f.Informer().GetIndexer())
}
