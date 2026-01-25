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

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

// FakeTestTriggers implements TestTriggerInterface
type FakeTestTriggers struct {
	Fake *FakeTestsV1
	ns   string
}

var testTriggersResource = schema.GroupVersionResource{Group: "tests.testkube.io", Version: "v1", Resource: "TestTrigger"}

var testTriggersKind = schema.GroupVersionKind{Group: "tests.testkube.io", Version: "v1", Kind: "TestTrigger"}

// Get takes name of the testTrigger, and returns the corresponding testTrigger object, and an error if there is any.
func (c *FakeTestTriggers) Get(ctx context.Context, name string, options v1.GetOptions) (result *testtriggersv1.TestTrigger, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(testTriggersResource, c.ns, name), &testtriggersv1.TestTrigger{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object %v", name)
	}

	return obj.(*testtriggersv1.TestTrigger), err
}

// List takes label and field selectors, and returns the list of TestTriggers that match those selectors.
func (c *FakeTestTriggers) List(ctx context.Context, opts v1.ListOptions) (result *testtriggersv1.TestTriggerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(testTriggersResource, testTriggersKind, c.ns, opts), &testtriggersv1.TestTriggerList{})

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
	list := &testtriggersv1.TestTriggerList{ListMeta: obj.(*testtriggersv1.TestTriggerList).ListMeta}
	for _, item := range obj.(*testtriggersv1.TestTriggerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested testTriggers.
func (c *FakeTestTriggers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(testTriggersResource, c.ns, opts))

}

// Create takes the representation of a testTrigger and creates it.  Returns the server's representation of the testTrigger, and an error, if there is any.
func (c *FakeTestTriggers) Create(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.CreateOptions) (result *testtriggersv1.TestTrigger, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(testTriggersResource, c.ns, testTrigger), &testtriggersv1.TestTrigger{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testtriggersv1.TestTrigger), err
}

// Update takes the representation of a testTrigger and updates it. Returns the server's representation of the testTrigger, and an error, if there is any.
func (c *FakeTestTriggers) Update(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (result *testtriggersv1.TestTrigger, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(testTriggersResource, c.ns, testTrigger), &testtriggersv1.TestTrigger{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testtriggersv1.TestTrigger), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTestTriggers) UpdateStatus(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (*testtriggersv1.TestTrigger, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(testTriggersResource, "status", c.ns, testTrigger), &testtriggersv1.TestTrigger{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testtriggersv1.TestTrigger), err
}

// Delete takes name of the testTrigger and deletes it. Returns an error if one occurs.
func (c *FakeTestTriggers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(testTriggersResource, c.ns, name, opts), &testtriggersv1.TestTrigger{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTestTriggers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(testTriggersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &testtriggersv1.TestTriggerList{})
	return err
}

// Patch applies the patch and returns the patched testTrigger.
func (c *FakeTestTriggers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *testtriggersv1.TestTrigger, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(testTriggersResource, c.ns, name, pt, data, subresources...), &testtriggersv1.TestTrigger{})

	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("empty object")
	}

	return obj.(*testtriggersv1.TestTrigger), err
}
