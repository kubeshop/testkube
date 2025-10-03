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

package v2

import (
	"context"
	"time"

	testsuitev2 "github.com/kubeshop/testkube/api/testsuite/v2"
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

// TestSuitesGetter has a method to return a TestSuiteInterface.
// A group's client should implement this interface.
type TestSuitesGetter interface {
	TestSuites(namespace string) TestSuiteInterface
}

// TestSuiteInterface has methods to work with TestSuite resources.
type TestSuiteInterface interface {
	List(ctx context.Context, opts v1.ListOptions) (*testsuitev2.TestSuiteList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	TestSuiteExpansion
}

// testSuites implements TestSuiteInterface
type testSuites struct {
	client rest.Interface
	ns     string
}

// newTestSuites returns a TestSuites
func newTestSuites(c *TestsV2Client, namespace string) *testSuites {
	return &testSuites{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// List takes label and field selectors, and returns the list of TestSuites that match those selectors.
func (c *testSuites) List(ctx context.Context, opts v1.ListOptions) (result *testsuitev2.TestSuiteList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &testsuitev2.TestSuiteList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("testsuites").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested testSuites.
func (c *testSuites) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("testsuites").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
