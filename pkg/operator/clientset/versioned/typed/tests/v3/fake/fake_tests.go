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

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

// FakeTests implements TestInterface
type FakeTests struct {
	Fake *FakeTestsV3
	ns   string
}

var testsResource = schema.GroupVersionResource{Group: "tests.testkube.io", Version: "v3", Resource: "Test"}

var testsKind = schema.GroupVersionKind{Group: "tests.testkube.io", Version: "v3", Kind: "Test"}

// List takes label and field selectors, and returns the list of Tests that match those selectors.
func (c *FakeTests) List(ctx context.Context, opts v1.ListOptions) (result *testsv3.TestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(testsResource, testsKind, c.ns, opts), &testsv3.TestList{})

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
	list := &testsv3.TestList{ListMeta: obj.(*testsv3.TestList).ListMeta}
	for _, item := range obj.(*testsv3.TestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tests.
func (c *FakeTests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(testsResource, c.ns, opts))
}
