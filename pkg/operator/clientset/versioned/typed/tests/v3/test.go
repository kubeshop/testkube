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

package v3

import (
	"context"
	"time"

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

// TestsGetter has a method to return a TestInterface.
// A group's client should implement this interface.
type TestsGetter interface {
	Tests(namespace string) TestInterface
}

// TestInterface has methods to work with Test resources.
type TestInterface interface {
	List(ctx context.Context, opts v1.ListOptions) (*testsv3.TestList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	TestExpansion
}

// testa implements TestInterface
type tests struct {
	client rest.Interface
	ns     string
}

// newTests returns a Tests
func newTests(c *TestsV3Client, namespace string) *tests {
	return &tests{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// List takes label and field selectors, and returns the list of Tests that match those selectors.
func (c *tests) List(ctx context.Context, opts v1.ListOptions) (result *testsv3.TestList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &testsv3.TestList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tests.
func (c *tests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("tests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
