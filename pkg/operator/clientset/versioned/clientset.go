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

package versioned

import (
	"fmt"
	"net/http"

	executorv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/executor/v1"
	v1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v1"
	v2 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v2"
	v3 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v3"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	TestsV1() v1.TestsV1Interface
	TestsV2() v2.TestsV2Interface
	TestsV3() v3.TestsV3Interface
	ExecutorV1() executorv1.ExecutorV1Interface
}

// Clientset contains the clients for groups. Each group has exactly one version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	testsV1    *v1.TestsV1Client
	testsV2    *v2.TestsV2Client
	testsV3    *v3.TestsV3Client
	executorV1 *executorv1.ExecutorV1Client
}

// TestsV1 retrieves the TestsV1Client
func (c *Clientset) TestsV1() v1.TestsV1Interface {
	return c.testsV1
}

// TestsV2 retrieves the TestsV2Client
func (c *Clientset) TestsV2() v2.TestsV2Interface {
	return c.testsV2
}

// TestsV3 retrieves the TestsV3Client
func (c *Clientset) TestsV3() v3.TestsV3Interface {
	return c.testsV3
}

// ExecutorV1 retrieves the ExecutorV1Client
func (c *Clientset) ExecutorV1() executorv1.ExecutorV1Interface {
	return c.executorV1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	// share the transport between all clients
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return NewForConfigAndClient(&configShallowCopy, httpClient)
}

// NewForConfigAndClient creates a new Clientset for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfigAndClient will generate a rate-limiter in configShallowCopy.
func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}

	var cs Clientset
	var err error
	cs.testsV1, err = v1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.testsV2, err = v2.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.testsV3, err = v3.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.executorV1, err = executorv1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.testsV1 = v1.New(c)
	cs.testsV2 = v2.New(c)
	cs.testsV3 = v3.New(c)
	cs.executorV1 = executorv1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
