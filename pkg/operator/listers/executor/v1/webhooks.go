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

// WebhookLister helps list Webhooks.
// All objects returned here must be treated as read-only.
type WebhookLister interface {
	// List lists all Webhooks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.Webhook, err error)
	// Webhooks returns an object that can list and get Webhooks.
	Webhooks(namespace string) WebhookNamespaceLister
	WebhookListerExpansion
}

// webhookLister implements the WebhookLister interface.
type webhookLister struct {
	indexer cache.Indexer
}

// NewWebhookLister returns a new WebhookLister.
func NewWebhookLister(indexer cache.Indexer) WebhookLister {
	return &webhookLister{indexer: indexer}
}

// List lists all Webhooks in the indexer.
func (s *webhookLister) List(selector labels.Selector) (ret []*executorv1.Webhook, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.Webhook))
	})
	return ret, err
}

// Webhooks returns an object that can list and get Webhooks.
func (s *webhookLister) Webhooks(namespace string) WebhookNamespaceLister {
	return webhookNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// WebhookNamespaceLister helps list and get Webhooks.
// All objects returned here must be treated as read-only.
type WebhookNamespaceLister interface {
	// List lists all Webhooks in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.Webhook, err error)
	// Get retrieves the Webhook from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*executorv1.Webhook, error)
	WebhookNamespaceListerExpansion
}

// webhookNamespaceLister implements the WebhookNamespaceLister
// interface.
type webhookNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Webhooks in the indexer for a given namespace.
func (s webhookNamespaceLister) List(selector labels.Selector) (ret []*executorv1.Webhook, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.Webhook))
	})
	return ret, err
}

// Get retrieves the Webhook from the indexer for a given namespace and name.
func (s webhookNamespaceLister) Get(name string) (*executorv1.Webhook, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: executorv1.GroupVersion.Group, Resource: executorv1.WebhookResource},
			name,
		)
	}
	return obj.(*executorv1.Webhook), nil
}
