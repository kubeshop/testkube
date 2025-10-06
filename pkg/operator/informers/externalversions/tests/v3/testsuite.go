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

package v3

import (
	"context"
	"time"

	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/operator/informers/externalversions/internalinterfaces"
	testsuiteslisterv3 "github.com/kubeshop/testkube/pkg/operator/listers/tests/v3"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// TestSuiteInformer provides access to a shared informer and lister for TestSuite.
type TestSuiteInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() testsuiteslisterv3.TestSuiteLister
}

type testSuiteInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewTestSuiteInformer constructs a new informer for TestSuite type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewTestSuiteInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTestSuiteInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredTestSuiteInformer constructs a new informer for TestSuite type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory print and number of connections to the server.
func NewFilteredTestSuiteInformer(
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
				return client.TestsV3().TestSuites(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TestsV3().TestSuites(namespace).Watch(context.TODO(), options)
			},
		},
		&testsuitev3.TestSuite{},
		resyncPeriod,
		indexers,
	)
}

func (f *testSuiteInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTestSuiteInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *testSuiteInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&testsuitev3.TestSuite{}, f.defaultInformer)
}

func (f *testSuiteInformer) Lister() testsuiteslisterv3.TestSuiteLister {
	return testsuiteslisterv3.NewTestSuiteLister(f.Informer().GetIndexer())
}
