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
	"github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	executorv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/executor/v1"
	fakeexecutorv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/executor/v1/fake"
	testsv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v1"
	faketestsv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v1/fake"
	v2 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v2"
	fakev2 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v2/fake"
	v3 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v3"
	fakev3 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/typed/tests/v3/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{tracker: o}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
	tracker   testing.ObjectTracker
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func (c *Clientset) Tracker() testing.ObjectTracker {
	return c.tracker
}

var (
	_ versioned.Interface = &Clientset{}
	_ testing.FakeClient  = &Clientset{}
)

// TestsV1 retrieves the TestsV1Client
func (c *Clientset) TestsV1() testsv1.TestsV1Interface {
	return &faketestsv1.FakeTestsV1{Fake: &c.Fake}
}

// TestsV2 retrieves the TestsV2Client
func (c *Clientset) TestsV2() v2.TestsV2Interface {
	return &fakev2.FakeTestsV2{Fake: &c.Fake}
}

// TestsV3 retrieves the TestsV3Client
func (c *Clientset) TestsV3() v3.TestsV3Interface {
	return &fakev3.FakeTestsV3{Fake: &c.Fake}
}

// ExecutorV1 retrieves the ExecutorV1Client
func (c *Clientset) ExecutorV1() executorv1.ExecutorV1Interface {
	return &fakeexecutorv1.FakeExecutorV1{Fake: &c.Fake}
}
