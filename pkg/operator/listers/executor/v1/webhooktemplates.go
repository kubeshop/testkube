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

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
)

// WebhookTemplateLister helps list WebhookTemplates.
// All objects returned here must be treated as read-only.
type WebhookTemplateLister interface {
	// List lists all WebhookTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.WebhookTemplate, err error)
	// WebhookTemplates returns an object that can list and get WebhookTemplates.
	WebhookTemplates(namespace string) WebhookTemplateNamespaceLister
	WebhookTemplateListerExpansion
}

// webhookTemplateLister implements the WebhookTemplateLister interface.
type webhookTemplateLister struct {
	indexer cache.Indexer
}

// NewWebhookTemplateLister returns a new WebhookTemplateLister.
func NewWebhookTemplateLister(indexer cache.Indexer) WebhookTemplateLister {
	return &webhookTemplateLister{indexer: indexer}
}

// List lists all WebhookTemplates in the indexer.
func (s *webhookTemplateLister) List(selector labels.Selector) (ret []*executorv1.WebhookTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.WebhookTemplate))
	})
	return ret, err
}

// WebhookTemplates returns an object that can list and get WebhookTemplates.
func (s *webhookTemplateLister) WebhookTemplates(namespace string) WebhookTemplateNamespaceLister {
	return webhookTemplateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// WebhookTemplateNamespaceLister helps list and get WebhookTemplates.
// All objects returned here must be treated as read-only.
type WebhookTemplateNamespaceLister interface {
	// List lists all WebhookTemplates in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*executorv1.WebhookTemplate, err error)
	// Get retrieves the WebhookTemplate from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*executorv1.WebhookTemplate, error)
	WebhookTemplateNamespaceListerExpansion
}

// webhookTemplateNamespaceLister implements the WebhookTemplateNamespaceLister
// interface.
type webhookTemplateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all WebhookTemplates in the indexer for a given namespace.
func (s webhookTemplateNamespaceLister) List(selector labels.Selector) (ret []*executorv1.WebhookTemplate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*executorv1.WebhookTemplate))
	})
	return ret, err
}

// Get retrieves the WebhookTemplate from the indexer for a given namespace and name.
func (s webhookTemplateNamespaceLister) Get(name string) (*executorv1.WebhookTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: executorv1.GroupVersion.Group, Resource: executorv1.WebhookTemplateResource},
			name,
		)
	}
	return obj.(*executorv1.WebhookTemplate), nil
}
