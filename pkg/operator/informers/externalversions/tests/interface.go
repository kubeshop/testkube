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

package tests

import (
	"github.com/kubeshop/testkube/pkg/operator/informers/externalversions/internalinterfaces"
	v1 "github.com/kubeshop/testkube/pkg/operator/informers/externalversions/tests/v1"
	v2 "github.com/kubeshop/testkube/pkg/operator/informers/externalversions/tests/v2"
	v3 "github.com/kubeshop/testkube/pkg/operator/informers/externalversions/tests/v3"
)

// Interface provides access to each of this group's versions.
type Interface interface {
	// V1 provides access to shared informers for resources in V1 version.
	V1() v1.Interface
	// V2 provides access to shared informers for resources in V2 version.
	V2() v2.Interface
	// V3 provides access to shared informers for resources in V3 version.
	V3() v3.Interface
}

type group struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(
	f internalinterfaces.SharedInformerFactory,
	namespace string,
	tweakListOptions internalinterfaces.TweakListOptionsFunc,
) Interface {
	return &group{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// V1 returns a new v1.Interface.
func (g *group) V1() v1.Interface {
	return v1.New(g.factory, g.namespace, g.tweakListOptions)
}

// V2 returns a new v2.Interface.
func (g *group) V2() v2.Interface {
	return v2.New(g.factory, g.namespace, g.tweakListOptions)
}

// V3 returns a new v3.Interface.
func (g *group) V3() v3.Interface {
	return v3.New(g.factory, g.namespace, g.tweakListOptions)
}
