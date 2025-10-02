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
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// TestTriggerLister helps list TestTriggers.
// All objects returned here must be treated as read-only.
type TestTriggerLister interface {
	// List lists all TestTriggers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testtriggersv1.TestTrigger, err error)
	// TestTriggers returns an object that can list and get TestTriggers.
	TestTriggers(namespace string) TestTriggerNamespaceLister
	TestTriggerListerExpansion
}

// testTriggerLister implements the TestTriggerLister interface.
type testTriggerLister struct {
	indexer cache.Indexer
}

// NewTestTriggerLister returns a new TestTriggerLister.
func NewTestTriggerLister(indexer cache.Indexer) TestTriggerLister {
	return &testTriggerLister{indexer: indexer}
}

// List lists all TestTriggers in the indexer.
func (s *testTriggerLister) List(selector labels.Selector) (ret []*testtriggersv1.TestTrigger, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*testtriggersv1.TestTrigger))
	})
	return ret, err
}

// TestTriggers returns an object that can list and get TestTriggers.
func (s *testTriggerLister) TestTriggers(namespace string) TestTriggerNamespaceLister {
	return testTriggerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TestTriggerNamespaceLister helps list and get TestTriggers.
// All objects returned here must be treated as read-only.
type TestTriggerNamespaceLister interface {
	// List lists all TestTriggers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testtriggersv1.TestTrigger, err error)
	// Get retrieves the TestTrigger from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*testtriggersv1.TestTrigger, error)
	TestTriggerNamespaceListerExpansion
}

// testTriggerNamespaceLister implements the TestTriggerNamespaceLister
// interface.
type testTriggerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TestTriggers in the indexer for a given namespace.
func (s testTriggerNamespaceLister) List(selector labels.Selector) (ret []*testtriggersv1.TestTrigger, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*testtriggersv1.TestTrigger))
	})
	return ret, err
}

// Get retrieves the TestTrigger from the indexer for a given namespace and name.
func (s testTriggerNamespaceLister) Get(name string) (*testtriggersv1.TestTrigger, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: testtriggersv1.GroupVersion.Group, Resource: testtriggersv1.Resource},
			name,
		)
	}
	return obj.(*testtriggersv1.TestTrigger), nil
}
