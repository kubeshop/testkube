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

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

// WebhookTemplateGetter has a method to return a WebhookTemplateInterface.
// A group's client should implement this interface.
type WebhookTemplateGetter interface {
	WebhookTemplate(namespace string) WebhookTemplateInterface
}

// WebhookTemplateInterface has methods to work with Webhook Template resources.
type WebhookTemplateInterface interface {
	List(ctx context.Context, opts v1.ListOptions) (*executorv1.WebhookTemplateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	WebhookTemplateExpansion
}

// webhookTemplates implements WebhookTemplateInterface
type webhookTemplates struct {
	client rest.Interface
	ns     string
}

// newWebhookTemplate returns a Webhook Template
func newWebhookTemplate(c *ExecutorV1Client, namespace string) *webhookTemplates {
	return &webhookTemplates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// List takes label and field selectors, and returns the list of Webhook Template that match those selectors.
func (c *webhookTemplates) List(ctx context.Context, opts v1.ListOptions) (result *executorv1.WebhookTemplateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &executorv1.WebhookTemplateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("webhookTemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested webhook templates.
func (c *webhookTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("webhookTemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
