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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
)

// TestSourceLister helps list TestSources.
// All objects returned here must be treated as read-only.
type TestSourceLister interface {
	// List lists all TestSources in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsourcev1.TestSource, err error)
	// TestSources returns an object that can list and get TestSources.
	TestSources(namespace string) TestSourceNamespaceLister
	TestSourceListerExpansion
}

// testSourceLister implements the TestSourceLister interface.
type testSourceLister struct {
	indexer cache.Indexer
}

// NewTestSourceLister returns a new TestSourceLister.
func NewTestSourceLister(indexer cache.Indexer) TestSourceLister {
	return &testSourceLister{indexer: indexer}
}

// List lists all TestSources in the indexer.
func (s *testSourceLister) List(selector labels.Selector) (ret []*testsourcev1.TestSource, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*testsourcev1.TestSource))
	})
	return ret, err
}

// TestSources returns an object that can list and get TestSources.
func (s *testSourceLister) TestSources(namespace string) TestSourceNamespaceLister {
	return testSourceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TestSourceNamespaceLister helps list and get TestSources.
// All objects returned here must be treated as read-only.
type TestSourceNamespaceLister interface {
	// List lists all TestSources in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsourcev1.TestSource, err error)
	// Get retrieves the TestSource from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*testsourcev1.TestSource, error)
	TestSourceNamespaceListerExpansion
}

// testSourceNamespaceLister implements the TestSourceNamespaceLister
// interface.
type testSourceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TestSources in the indexer for a given namespace.
func (s testSourceNamespaceLister) List(selector labels.Selector) (ret []*testsourcev1.TestSource, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*testsourcev1.TestSource))
	})
	return ret, err
}

// Get retrieves the TestSource from the indexer for a given namespace and name.
func (s testSourceNamespaceLister) Get(name string) (*testsourcev1.TestSource, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: testsourcev1.GroupVersion.Group, Resource: testsourcev1.Resource},
			name,
		)
	}
	return obj.(*testsourcev1.TestSource), nil
}
