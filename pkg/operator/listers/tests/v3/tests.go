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

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
)

// TestLister helps list Tests.
// All objects returned here must be treated as read-only.
type TestLister interface {
	// List lists all Tests in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsv3.Test, err error)
	// Tests returns an object that can list and get Tests.
	Tests(namespace string) TestNamespaceLister
	TestListerExpansion
}

// testLister implements the TestLister interface.
type testLister struct {
	indexer cache.Indexer
}

// NewTestLister returns a new TestLister.
func NewTestLister(indexer cache.Indexer) TestLister {
	return &testLister{indexer: indexer}
}

// List lists all Tests in the indexer.
func (s *testLister) List(selector labels.Selector) (ret []*testsv3.Test, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*testsv3.Test))
	})
	return ret, err
}

// Tests returns an object that can list and get Tests.
func (s *testLister) Tests(namespace string) TestNamespaceLister {
	return testNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TestNamespaceLister helps list and get Tests.
// All objects returned here must be treated as read-only.
type TestNamespaceLister interface {
	// List lists all Tests in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*testsv3.Test, err error)
	// Get retrieves the Test from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*testsv3.Test, error)
	TestNamespaceListerExpansion
}

// testNamespaceLister implements the TestNamespaceLister
// interface.
type testNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Tests in the indexer for a given namespace.
func (s testNamespaceLister) List(selector labels.Selector) (ret []*testsv3.Test, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*testsv3.Test))
	})
	return ret, err
}

// Get retrieves the Test from the indexer for a given namespace and name.
func (s testNamespaceLister) Get(name string) (*testsv3.Test, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: testsv3.GroupVersion.Group, Resource: testsv3.Resource},
			name,
		)
	}
	return obj.(*testsv3.Test), nil
}
