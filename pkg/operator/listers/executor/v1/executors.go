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
	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// ExecutorLister helps list Executors.
// All objects returned here must be treated as read-only.
type ExecutorLister interface {
	// List lists all Executors in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.Executor, err error)
	// Executors returns an object that can list and get Executors.
	Executors(namespace string) ExecutorNamespaceLister
	ExecutorListerExpansion
}

// executorLister implements the ExecutorLister interface.
type executorLister struct {
	indexer cache.Indexer
}

// NewExecutorLister returns a new ExecutorLister.
func NewExecutorLister(indexer cache.Indexer) ExecutorLister {
	return &executorLister{indexer: indexer}
}

// List lists all Executors in the indexer.
func (s *executorLister) List(selector labels.Selector) (ret []*executorv1.Executor, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.Executor))
	})
	return ret, err
}

// Executors returns an object that can list and get Executors.
func (s *executorLister) Executors(namespace string) ExecutorNamespaceLister {
	return executorNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ExecutorNamespaceLister helps list and get Executors.
// All objects returned here must be treated as read-only.
type ExecutorNamespaceLister interface {
	// List lists all Executors in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.Executor, err error)
	// Get retrieves the Executor from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*executorv1.Executor, error)
	ExecutorNamespaceListerExpansion
}

// executorNamespaceLister implements the ExecutorNamespaceLister
// interface.
type executorNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Executors in the indexer for a given namespace.
func (s executorNamespaceLister) List(selector labels.Selector) (ret []*executorv1.Executor, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.Executor))
	})
	return ret, err
}

// Get retrieves the Executor from the indexer for a given namespace and name.
func (s executorNamespaceLister) Get(name string) (*executorv1.Executor, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: executorv1.GroupVersion.Group, Resource: executorv1.ExecutorResource},
			name,
		)
	}
	return obj.(*executorv1.Executor), nil
}
