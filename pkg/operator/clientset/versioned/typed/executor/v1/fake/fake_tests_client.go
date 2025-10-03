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
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"

	v1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/executor/v1"
)

type FakeExecutorV1 struct {
	*testing.Fake
}

func (c *FakeExecutorV1) Executor(namespace string) v1.ExecutorInterface {
	return &FakeExecutor{c, namespace}
}

func (c *FakeExecutorV1) Webhook(namespace string) v1.WebhookInterface {
	return &FakeWebhook{c, namespace}
}

func (c *FakeExecutorV1) WebhookTemplate(namespace string) v1.WebhookTemplateInterface {
	return &FakeWebhookTemplate{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExecutorV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
