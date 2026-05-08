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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testkubev2 "github.com/kubeshop/testkube/api/tests/v2"
)

// ConvertTo converts this Script to the Hub version (v1).
func (src *Test) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*testkubev2.Test)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.Type_ = src.Spec.Type_
	dst.Spec.Name = src.Spec.Name
	dst.Spec.Schedule = src.Spec.Schedule

	if src.Spec.ExecutionRequest != nil {
		dst.Spec.Variables = make(map[string]testkubev2.Variable, len(src.Spec.ExecutionRequest.Variables))
		for key, value := range src.Spec.ExecutionRequest.Variables {
			dst.Spec.Variables[key] = testkubev2.Variable{
				Type_:     value.Type_,
				Name:      value.Name,
				Value:     value.Value,
				ValueFrom: value.ValueFrom,
			}
		}

		dst.Spec.ExecutorArgs = make([]string, len(src.Spec.ExecutionRequest.Args))
		copy(dst.Spec.ExecutorArgs, src.Spec.ExecutionRequest.Args)
	}

	if src.Spec.Content != nil {
		dst.Spec.Content = &testkubev2.TestContent{
			Data:  src.Spec.Content.Data,
			Type_: string(src.Spec.Content.Type_),
			Uri:   src.Spec.Content.Uri,
		}
	}

	if src.Spec.Content != nil && src.Spec.Content.Repository != nil {
		dst.Spec.Content.Repository = &testkubev2.Repository{
			Type_:  src.Spec.Content.Repository.Type_,
			Uri:    src.Spec.Content.Repository.Uri,
			Branch: src.Spec.Content.Repository.Branch,
			Commit: src.Spec.Content.Repository.Commit,
			Path:   src.Spec.Content.Repository.Path,
		}
	}

	return nil
}

// ConvertFrom converts Script from the Hub version (v1) to this version.
func (dst *Test) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*testkubev2.Test)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.Type_ = src.Spec.Type_
	dst.Spec.Name = src.Spec.Name
	dst.Spec.Schedule = src.Spec.Schedule

	if len(src.Spec.Variables) != 0 || len(src.Spec.ExecutorArgs) != 0 || len(src.Spec.Params) != 0 {
		dst.Spec.ExecutionRequest = &ExecutionRequest{}
		dst.Spec.ExecutionRequest.Variables = make(map[string]Variable, len(src.Spec.Variables)+len(src.Spec.Params))
		for key, value := range src.Spec.Params {
			dst.Spec.ExecutionRequest.Variables[key] = Variable{
				Type_: commonv1.VariableTypeBasic,
				Name:  key,
				Value: value,
			}
		}

		for key, value := range src.Spec.Variables {
			dst.Spec.ExecutionRequest.Variables[key] = Variable{
				Type_:     value.Type_,
				Name:      value.Name,
				Value:     value.Value,
				ValueFrom: value.ValueFrom,
			}
		}

		dst.Spec.ExecutionRequest.Args = make([]string, len(src.Spec.ExecutorArgs))
		copy(dst.Spec.ExecutionRequest.Args, src.Spec.ExecutorArgs)
	}

	if src.Spec.Content != nil {
		if src.Spec.Content != nil {
			dst.Spec.Content = &TestContent{
				Data:  src.Spec.Content.Data,
				Type_: TestContentType(src.Spec.Content.Type_),
				Uri:   src.Spec.Content.Uri,
			}
		}
	}

	if src.Spec.Content != nil && src.Spec.Content.Repository != nil {
		dst.Spec.Content.Repository = &Repository{
			Type_:  src.Spec.Content.Repository.Type_,
			Uri:    src.Spec.Content.Repository.Uri,
			Branch: src.Spec.Content.Repository.Branch,
			Commit: src.Spec.Content.Repository.Commit,
			Path:   src.Spec.Content.Repository.Path,
		}
	}

	// Status
	return nil
}
