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

package fake

import (
	"context"
	"fmt"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

// FakeWebhook implements WebhookInterface
type FakeWebhook struct {
	Fake *FakeExecutorV1
	ns   string
}

var webhookResource = schema.GroupVersionResource{Group: "executor.testkube.io", Version: "v1", Resource: "Webhook"}

var webhookKind = schema.GroupVersionKind{Group: "executor.testkube.io", Version: "v1", Kind: "Webhook"}

// List takes label and field selectors, and returns the list of Webhook that match those selectors.
func (c *FakeWebhook) List(ctx context.Context, opts v1.ListOptions) (result *executorv1.WebhookList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(webhookResource, webhookKind, c.ns, opts), &executorv1.WebhookList{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &executorv1.WebhookList{ListMeta: obj.(*executorv1.WebhookList).ListMeta}
	for _, item := range obj.(*executorv1.WebhookList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested Webhooks.
func (c *FakeWebhook) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(webhookResource, c.ns, opts))
}
