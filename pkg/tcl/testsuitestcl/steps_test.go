// Copyright 2024 Kubeshop.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/master/licenses/TCL.txt

package testsuitestcl

import (
	"testing"

	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testsuitestclop "github.com/kubeshop/testkube-operator/pkg/tcl/testsuitestcl"
)

func TestHasStepsExecutionRequest(t *testing.T) {
	tests := []struct {
		name      string
		testSuite testsuitesv3.TestSuite
		want      bool
	}{
		{
			name: "TestSuiteSpec with steps execution request in before",
			testSuite: testsuitesv3.TestSuite{
				Spec: testsuitesv3.TestSuiteSpec{
					Before: []testsuitesv3.TestSuiteBatchStep{
						{
							Execute: []testsuitesv3.TestSuiteStepSpec{
								{
									TestSuiteStepExecutionRequest: &testsuitestclop.TestSuiteStepExecutionRequest{
										Name: "execution request",
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "TestSuiteSpec with steps execution request in steps",
			testSuite: testsuitesv3.TestSuite{
				Spec: testsuitesv3.TestSuiteSpec{
					Steps: []testsuitesv3.TestSuiteBatchStep{
						{
							Execute: []testsuitesv3.TestSuiteStepSpec{
								{
									TestSuiteStepExecutionRequest: &testsuitestclop.TestSuiteStepExecutionRequest{
										Name: "execution request",
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "TestSuiteSpec with steps execution request in after",
			testSuite: testsuitesv3.TestSuite{
				Spec: testsuitesv3.TestSuiteSpec{
					After: []testsuitesv3.TestSuiteBatchStep{
						{
							Execute: []testsuitesv3.TestSuiteStepSpec{
								{
									TestSuiteStepExecutionRequest: &testsuitestclop.TestSuiteStepExecutionRequest{
										Name: "execution request",
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "TestSuiteSpec with no steps execution request",
			testSuite: testsuitesv3.TestSuite{
				Spec: testsuitesv3.TestSuiteSpec{
					Before: []testsuitesv3.TestSuiteBatchStep{
						{
							Execute: []testsuitesv3.TestSuiteStepSpec{
								{
									Test: "test",
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasStepsExecutionRequest(tt.testSuite); got != tt.want {
				t.Errorf("HasStepsExecutionRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
