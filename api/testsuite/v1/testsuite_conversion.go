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

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testkubev2 "github.com/kubeshop/testkube/api/testsuite/v2"
)

// ConvertTo converts this Script to the Hub version (v2).
func (src *TestSuite) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*testkubev2.TestSuite)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.Repeats = src.Spec.Repeats
	dst.Spec.Description = src.Spec.Description
	dst.Spec.Schedule = src.Spec.Schedule

	dst.Spec.Before = make([]testkubev2.TestSuiteStepSpec, len(src.Spec.Before))
	dst.Spec.Steps = make([]testkubev2.TestSuiteStepSpec, len(src.Spec.Steps))
	dst.Spec.After = make([]testkubev2.TestSuiteStepSpec, len(src.Spec.After))

	var stepTypes = []struct {
		Source     []TestSuiteStepSpec
		Destinaton []testkubev2.TestSuiteStepSpec
	}{
		{
			Source:     src.Spec.Before,
			Destinaton: dst.Spec.Before,
		},
		{
			Source:     src.Spec.Steps,
			Destinaton: dst.Spec.Steps,
		},
		{
			Source:     src.Spec.After,
			Destinaton: dst.Spec.After,
		},
	}

	for _, stepType := range stepTypes {
		for i := range stepType.Source {
			value := stepType.Source[i]
			step := testkubev2.TestSuiteStepSpec{
				Type: testkubev2.TestSuiteStepType(value.Type),
			}

			if value.Delay != nil {
				step.Delay = &testkubev2.TestSuiteStepDelay{
					Duration: value.Delay.Duration,
				}
			}

			if value.Execute != nil {
				step.Execute = &testkubev2.TestSuiteStepExecute{
					Namespace:     value.Execute.Namespace,
					Name:          value.Execute.Name,
					StopOnFailure: value.Execute.StopOnFailure,
				}
			}

			stepType.Destinaton[i] = step
		}
	}

	if len(src.Spec.Variables) != 0 || len(src.Spec.Params) != 0 {
		dst.Spec.ExecutionRequest = &testkubev2.TestSuiteExecutionRequest{}
		dst.Spec.ExecutionRequest.Variables = make(map[string]testkubev2.Variable, len(src.Spec.Variables)+len(src.Spec.Params))
		for key, value := range src.Spec.Params {
			dst.Spec.ExecutionRequest.Variables[key] = testkubev2.Variable{
				Type_: commonv1.VariableTypeBasic,
				Name:  key,
				Value: value,
			}
		}

		for key, value := range src.Spec.Variables {
			dst.Spec.ExecutionRequest.Variables[key] = testkubev2.Variable{
				Type_:     value.Type_,
				Name:      value.Name,
				Value:     value.Value,
				ValueFrom: value.ValueFrom,
			}
		}
	}

	return nil
}

// ConvertFrom converts Script from the Hub version (v2) to this version.
func (dst *TestSuite) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*testkubev2.TestSuite)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.Repeats = src.Spec.Repeats
	dst.Spec.Description = src.Spec.Description
	dst.Spec.Schedule = src.Spec.Schedule

	dst.Spec.Before = make([]TestSuiteStepSpec, len(src.Spec.Before))
	dst.Spec.Steps = make([]TestSuiteStepSpec, len(src.Spec.Steps))
	dst.Spec.After = make([]TestSuiteStepSpec, len(src.Spec.After))

	var stepTypes = []struct {
		source     []testkubev2.TestSuiteStepSpec
		destinaton []TestSuiteStepSpec
	}{
		{
			source:     src.Spec.Before,
			destinaton: dst.Spec.Before,
		},
		{
			source:     src.Spec.Steps,
			destinaton: dst.Spec.Steps,
		},
		{
			source:     src.Spec.After,
			destinaton: dst.Spec.After,
		},
	}

	for _, stepType := range stepTypes {
		for i := range stepType.source {
			value := stepType.source[i]
			step := TestSuiteStepSpec{
				Type: string(value.Type),
			}

			if value.Delay != nil {
				step.Delay = &TestSuiteStepDelay{
					Duration: value.Delay.Duration,
				}
			}

			if value.Execute != nil {
				step.Execute = &TestSuiteStepExecute{
					Namespace:     value.Execute.Namespace,
					Name:          value.Execute.Name,
					StopOnFailure: value.Execute.StopOnFailure,
				}
			}

			stepType.destinaton[i] = step
		}
	}

	if src.Spec.ExecutionRequest != nil {
		dst.Spec.Variables = make(map[string]Variable, len(src.Spec.ExecutionRequest.Variables))
		for key, value := range src.Spec.ExecutionRequest.Variables {
			dst.Spec.Variables[key] = Variable{
				Type_:     value.Type_,
				Name:      value.Name,
				Value:     value.Value,
				ValueFrom: value.ValueFrom,
			}
		}
	}

	// Status
	return nil
}
