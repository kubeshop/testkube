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

	testsuitev3 "github.com/kubeshop/testkube/api/testsuite/v3"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

// FakeTestSuites implements TestSuiteInterface
type FakeTestSuites struct {
	Fake *FakeTestsV3
	ns   string
}

var testSuitesResource = schema.GroupVersionResource{Group: "tests.testkube.io", Version: "v3", Resource: "TestSuite"}

var testSuitesKind = schema.GroupVersionKind{Group: "tests.testkube.io", Version: "v3", Kind: "TestSuite"}

// List takes label and field selectors, and returns the list of TestSuites that match those selectors.
func (c *FakeTestSuites) List(ctx context.Context, opts v1.ListOptions) (result *testsuitev3.TestSuiteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(testSuitesResource, testSuitesKind, c.ns, opts), &testsuitev3.TestSuiteList{})

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
	list := &testsuitev3.TestSuiteList{ListMeta: obj.(*testsuitev3.TestSuiteList).ListMeta}
	for _, item := range obj.(*testsuitev3.TestSuiteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested testSuites.
func (c *FakeTestSuites) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(testSuitesResource, c.ns, opts))
}
