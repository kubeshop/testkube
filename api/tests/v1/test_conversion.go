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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	testkubev2 "github.com/kubeshop/testkube/api/tests/v2"
)

// ConvertTo converts this Script to the Hub version (v1).
func (src *Test) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*testkubev2.Test)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	return nil
}

// ConvertFrom converts Script from the Hub version (v1) to this version.
func (dst *Test) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*testkubev2.Test)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	// Status
	return nil
}
