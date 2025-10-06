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

	v2 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v2"
)

type FakeTestsV2 struct {
	*testing.Fake
}

func (c *FakeTestsV2) TestSuites(namespace string) v2.TestSuiteInterface {
	return &FakeTestSuites{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeTestsV2) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
