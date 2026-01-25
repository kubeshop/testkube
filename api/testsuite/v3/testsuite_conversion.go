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
	"time"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	var stepTypes = []struct {
		Source     []TestSuiteBatchStep
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
			for _, value := range stepType.Source[i].Execute {
				step := testkubev2.TestSuiteStepSpec{}

				if value.Delay.Duration != 0 {
					step.Delay = &testkubev2.TestSuiteStepDelay{
						Duration: int32(value.Delay.Duration / time.Millisecond),
					}
				}

				if value.Test != "" {
					step.Execute = &testkubev2.TestSuiteStepExecute{
						Name:          value.Test,
						StopOnFailure: stepType.Source[i].StopOnFailure,
					}
				}

				stepType.Destinaton = append(stepType.Destinaton, step)
			}
		}
	}

	if src.Spec.ExecutionRequest != nil {
		variables := make(map[string]testkubev2.Variable, len(src.Spec.ExecutionRequest.Variables))
		for key, value := range src.Spec.ExecutionRequest.Variables {
			variables[key] = testkubev2.Variable(value)
		}

		dst.Spec.ExecutionRequest = &testkubev2.TestSuiteExecutionRequest{
			Name:            src.Spec.ExecutionRequest.Name,
			Namespace:       src.Spec.ExecutionRequest.Namespace,
			Variables:       variables,
			SecretUUID:      src.Spec.ExecutionRequest.SecretUUID,
			Labels:          src.Spec.ExecutionRequest.Labels,
			ExecutionLabels: src.Spec.ExecutionRequest.ExecutionLabels,
			Sync:            src.Spec.ExecutionRequest.Sync,
			HttpProxy:       src.Spec.ExecutionRequest.HttpProxy,
			HttpsProxy:      src.Spec.ExecutionRequest.HttpsProxy,
			Timeout:         src.Spec.ExecutionRequest.Timeout,
			CronJobTemplate: src.Spec.ExecutionRequest.CronJobTemplate,
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

	dst.Spec.Before = make([]TestSuiteBatchStep, len(src.Spec.Before))
	dst.Spec.Steps = make([]TestSuiteBatchStep, len(src.Spec.Steps))
	dst.Spec.After = make([]TestSuiteBatchStep, len(src.Spec.After))

	var stepTypes = []struct {
		source     []testkubev2.TestSuiteStepSpec
		destinaton []TestSuiteBatchStep
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
			step := TestSuiteStepSpec{}

			if value.Delay != nil {
				step.Delay = metav1.Duration{Duration: time.Duration(value.Delay.Duration) * time.Millisecond}
			}

			var stopOnFailure bool
			if value.Execute != nil {
				step.Test = value.Execute.Name
				stopOnFailure = value.Execute.StopOnFailure
			}

			stepType.destinaton[i] = TestSuiteBatchStep{
				StopOnFailure: stopOnFailure,
				Execute:       []TestSuiteStepSpec{step},
			}
		}
	}

	if src.Spec.ExecutionRequest != nil {
		variables := make(map[string]Variable, len(src.Spec.ExecutionRequest.Variables))
		for key, value := range src.Spec.ExecutionRequest.Variables {
			variables[key] = Variable(value)
		}

		dst.Spec.ExecutionRequest = &TestSuiteExecutionRequest{
			Name:            src.Spec.ExecutionRequest.Name,
			Namespace:       src.Spec.ExecutionRequest.Namespace,
			Variables:       variables,
			SecretUUID:      src.Spec.ExecutionRequest.SecretUUID,
			Labels:          src.Spec.ExecutionRequest.Labels,
			ExecutionLabels: src.Spec.ExecutionRequest.ExecutionLabels,
			Sync:            src.Spec.ExecutionRequest.Sync,
			HttpProxy:       src.Spec.ExecutionRequest.HttpProxy,
			HttpsProxy:      src.Spec.ExecutionRequest.HttpsProxy,
			Timeout:         src.Spec.ExecutionRequest.Timeout,
			CronJobTemplate: src.Spec.ExecutionRequest.CronJobTemplate,
		}
	}

	// Status
	return nil
}
