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

// WebhookGetter has a method to return a WebhookInterface.
// A group's client should implement this interface.
type WebhookGetter interface {
	Webhook(namespace string) WebhookInterface
}

// WebhookInterface has methods to work with Webhook resources.
type WebhookInterface interface {
	List(ctx context.Context, opts v1.ListOptions) (*executorv1.WebhookList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	WebhookExpansion
}

// webhooks implements WebhookInterface
type webhooks struct {
	client rest.Interface
	ns     string
}

// newWebhook returns a Webhook
func newWebhook(c *ExecutorV1Client, namespace string) *webhooks {
	return &webhooks{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// List takes label and field selectors, and returns the list of Webhook that match those selectors.
func (c *webhooks) List(ctx context.Context, opts v1.ListOptions) (result *executorv1.WebhookList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &executorv1.WebhookList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("webhooks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested webhooks.
func (c *webhooks) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("webhooks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
