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

	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

// FakeTestSource implements TestSourceInterface
type FakeTestSource struct {
	Fake *FakeTestsV1
	ns   string
}

var testSourceResource = schema.GroupVersionResource{Group: "tests.testkube.io", Version: "v1", Resource: "TestSource"}

var testSourceKind = schema.GroupVersionKind{Group: "tests.testkube.io", Version: "v1", Kind: "TestSource"}

// Get takes name of the testSource, and returns the corresponding testSource object, and an error if there is any.
func (c *FakeTestSource) Get(ctx context.Context, name string, options v1.GetOptions) (result *testsourcev1.TestSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(testSourceResource, c.ns, name), &testsourcev1.TestSource{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object %v", name)
	}

	return obj.(*testsourcev1.TestSource), err
}

// List takes label and field selectors, and returns the list of TestSource that match those selectors.
func (c *FakeTestSource) List(ctx context.Context, opts v1.ListOptions) (result *testsourcev1.TestSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(testSourceResource, testSourceKind, c.ns, opts), &testsourcev1.TestSourceList{})

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
	list := &testsourcev1.TestSourceList{ListMeta: obj.(*testsourcev1.TestSourceList).ListMeta}
	for _, item := range obj.(*testsourcev1.TestSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested testSource.
func (c *FakeTestSource) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(testSourceResource, c.ns, opts))

}

// Create takes the representation of a testSource and creates it.  Returns the server's representation of the testSource, and an error, if there is any.
func (c *FakeTestSource) Create(ctx context.Context, testSource *testsourcev1.TestSource, opts v1.CreateOptions) (result *testsourcev1.TestSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(testSourceResource, c.ns, testSource), &testsourcev1.TestSource{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testsourcev1.TestSource), err
}

// Update takes the representation of a testSource and updates it. Returns the server's representation of the testSource, and an error, if there is any.
func (c *FakeTestSource) Update(ctx context.Context, testSource *testsourcev1.TestSource, opts v1.UpdateOptions) (result *testsourcev1.TestSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(testSourceResource, c.ns, testSource), &testsourcev1.TestSource{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testsourcev1.TestSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTestSource) UpdateStatus(ctx context.Context, testSource *testsourcev1.TestSource, opts v1.UpdateOptions) (*testsourcev1.TestSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(testSourceResource, "status", c.ns, testSource), &testsourcev1.TestSource{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testsourcev1.TestSource), err
}

// Delete takes name of the testSource and deletes it. Returns an error if one occurs.
func (c *FakeTestSource) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(testSourceResource, c.ns, name, opts), &testsourcev1.TestSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTestSource) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(testSourceResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &testsourcev1.TestSourceList{})
	return err
}

// Patch applies the patch and returns the patched testSource.
func (c *FakeTestSource) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *testsourcev1.TestSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(testSourceResource, c.ns, name, pt, data, subresources...), &testsourcev1.TestSource{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testsourcev1.TestSource), err
}
