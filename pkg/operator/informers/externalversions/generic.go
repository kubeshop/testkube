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

package externalversions

import (
	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"

	"github.com/pkg/errors"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=tests.testkube.io, Version=v1
	case testtriggersv1.GroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Tests().V1().TestTriggers().Informer(),
		}, nil
		// Group=tests.testkube.io, Version=v3
	case testsuitev3.GroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Tests().V3().TestSuites().Informer(),
		}, nil
		// Group=tests.testkube.io, Version=v3
	case testsv3.GroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Tests().V3().Tests().Informer(),
		}, nil
		// Group=executor.testkube.io, Version=v1
	case executorv1.ExecutorGroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Executor().V1().Executor().Informer(),
		}, nil
		// Group=executor.testkube.io, Version=v1
	case executorv1.WebhookGroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Executor().V1().Webhook().Informer(),
		}, nil
		// Group=executor.testkube.io, Version=v1
	case executorv1.WebhookTemplateGroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Executor().V1().WebhookTemplate().Informer(),
		}, nil
		// Group=tests.testkube.io, Version=v1
	case testsourcev1.GroupVersionResource:
		return &genericInformer{
			resource: resource.GroupResource(),
			informer: f.Tests().V1().TestSource().Informer(),
		}, nil
	}

	return nil, errors.Errorf("no informer found for %v", resource)
}
