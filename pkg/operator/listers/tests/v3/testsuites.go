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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"
)

// TestSuiteLister helps list TestSuites.
// All objects returned here must be treated as read-only.
type TestSuiteLister interface {
	// List lists all TestSuites in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsuitev3.TestSuite, err error)
	// TestSuites returns an object that can list and get TestSuites.
	TestSuites(namespace string) TestSuiteNamespaceLister
	TestSuiteListerExpansion
}

// testSuiteLister implements the TestSuiteLister interface.
type testSuiteLister struct {
	indexer cache.Indexer
}

// NewTestSuiteLister returns a new TestSuiteLister.
func NewTestSuiteLister(indexer cache.Indexer) TestSuiteLister {
	return &testSuiteLister{indexer: indexer}
}

// List lists all TestSuites in the indexer.
func (s *testSuiteLister) List(selector labels.Selector) (ret []*testsuitev3.TestSuite, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*testsuitev3.TestSuite))
	})
	return ret, err
}

// TestSuites returns an object that can list and get TestSuites.
func (s *testSuiteLister) TestSuites(namespace string) TestSuiteNamespaceLister {
	return testSuiteNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TestSuiteNamespaceLister helps list and get TestSuites.
// All objects returned here must be treated as read-only.
type TestSuiteNamespaceLister interface {
	// List lists all TestSuites in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsuitev3.TestSuite, err error)
	// Get retrieves the TestSuite from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*testsuitev3.TestSuite, error)
	TestSuiteNamespaceListerExpansion
}

// testSuiteNamespaceLister implements the TestSuiteNamespaceLister
// interface.
type testSuiteNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TestSuites in the indexer for a given namespace.
func (s testSuiteNamespaceLister) List(selector labels.Selector) (ret []*testsuitev3.TestSuite, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*testsuitev3.TestSuite))
	})
	return ret, err
}

// Get retrieves the TestSuite from the indexer for a given namespace and name.
func (s testSuiteNamespaceLister) Get(name string) (*testsuitev3.TestSuite, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: testsuitev3.GroupVersion.Group, Resource: testsuitev3.Resource},
			name,
		)
	}
	return obj.(*testsuitev3.TestSuite), nil
}
