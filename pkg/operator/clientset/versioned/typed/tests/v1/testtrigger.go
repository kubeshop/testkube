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
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned/scheme"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

// TestTriggersGetter has a method to return a TestTriggerInterface.
// A group's client should implement this interface.
type TestTriggersGetter interface {
	TestTriggers(namespace string) TestTriggerInterface
}

// TestTriggerInterface has methods to work with TestTrigger resources.
type TestTriggerInterface interface {
	Create(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.CreateOptions) (*testtriggersv1.TestTrigger, error)
	Update(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (*testtriggersv1.TestTrigger, error)
	UpdateStatus(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (*testtriggersv1.TestTrigger, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*testtriggersv1.TestTrigger, error)
	List(ctx context.Context, opts v1.ListOptions) (*testtriggersv1.TestTriggerList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *testtriggersv1.TestTrigger, err error)
	TestTriggerExpansion
}

// testTriggers implements TestTriggerInterface
type testTriggers struct {
	client rest.Interface
	ns     string
}

// newTestTriggers returns a TestTriggers
func newTestTriggers(c *TestsV1Client, namespace string) *testTriggers {
	return &testTriggers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the testTrigger, and returns the corresponding testTrigger object, and an error if there is any.
func (c *testTriggers) Get(ctx context.Context, name string, options v1.GetOptions) (result *testtriggersv1.TestTrigger, err error) {
	result = &testtriggersv1.TestTrigger{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("testtriggers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of TestTriggers that match those selectors.
func (c *testTriggers) List(ctx context.Context, opts v1.ListOptions) (result *testtriggersv1.TestTriggerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &testtriggersv1.TestTriggerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("testtriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested testTriggers.
func (c *testTriggers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("testtriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a testTrigger and creates it.  Returns the server's representation of the testTrigger, and an error, if there is any.
func (c *testTriggers) Create(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.CreateOptions) (result *testtriggersv1.TestTrigger, err error) {
	result = &testtriggersv1.TestTrigger{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("testtriggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(testTrigger).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a testTrigger and updates it. Returns the server's representation of the testTrigger, and an error, if there is any.
func (c *testTriggers) Update(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (result *testtriggersv1.TestTrigger, err error) {
	result = &testtriggersv1.TestTrigger{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("testtriggers").
		Name(testTrigger.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(testTrigger).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *testTriggers) UpdateStatus(ctx context.Context, testTrigger *testtriggersv1.TestTrigger, opts v1.UpdateOptions) (result *testtriggersv1.TestTrigger, err error) {
	result = &testtriggersv1.TestTrigger{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("testtriggers").
		Name(testTrigger.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(testTrigger).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the testTrigger and deletes it. Returns an error if one occurs.
func (c *testTriggers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("testtriggers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *testTriggers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("testtriggers").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched testTrigger.
func (c *testTriggers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *testtriggersv1.TestTrigger, err error) {
	result = &testtriggersv1.TestTrigger{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("testtriggers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
