// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	"reflect"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestFlattenSignatures(t *testing.T) {
	tests := []struct {
		name     string
		input    []testkube.TestWorkflowSignature
		expected []testkube.TestWorkflowSignature
	}{
		{
			name:     "empty slice",
			input:    []testkube.TestWorkflowSignature{},
			expected: []testkube.TestWorkflowSignature{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: []testkube.TestWorkflowSignature{},
		},
		{
			name: "single signature without children",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Test Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Test Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
			},
		},
		{
			name: "multiple signatures without children",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Test Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "step2",
					Name:     "Test Step 2",
					Category: "integration",
					Optional: true,
					Negative: false,
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Test Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "step2",
					Name:     "Test Step 2",
					Category: "integration",
					Optional: true,
					Negative: false,
				},
			},
		},
		{
			name: "single parent with single child",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "parent1",
					Name:     "Parent Step",
					Category: "group",
					Children: []testkube.TestWorkflowSignature{
						{
							Ref:      "child1",
							Name:     "Child Step",
							Category: "unit",
							Optional: false,
							Negative: false,
						},
					},
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "child1",
					Name:     "Child Step",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
			},
		},
		{
			name: "single parent with multiple children",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "parent1",
					Name:     "Parent Step",
					Category: "group",
					Children: []testkube.TestWorkflowSignature{
						{
							Ref:      "child1",
							Name:     "Child Step 1",
							Category: "unit",
							Optional: false,
							Negative: false,
						},
						{
							Ref:      "child2",
							Name:     "Child Step 2",
							Category: "integration",
							Optional: true,
							Negative: true,
						},
					},
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "child1",
					Name:     "Child Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "child2",
					Name:     "Child Step 2",
					Category: "integration",
					Optional: true,
					Negative: true,
				},
			},
		},
		{
			name: "nested hierarchy (grandchildren)",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "grandparent",
					Name:     "Grandparent Step",
					Category: "suite",
					Children: []testkube.TestWorkflowSignature{
						{
							Ref:      "parent1",
							Name:     "Parent Step 1",
							Category: "group",
							Children: []testkube.TestWorkflowSignature{
								{
									Ref:      "child1",
									Name:     "Child Step 1",
									Category: "unit",
									Optional: false,
									Negative: false,
								},
								{
									Ref:      "child2",
									Name:     "Child Step 2",
									Category: "unit",
									Optional: true,
									Negative: false,
								},
							},
						},
					},
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "child1",
					Name:     "Child Step 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "child2",
					Name:     "Child Step 2",
					Category: "unit",
					Optional: true,
					Negative: false,
				},
			},
		},
		{
			name: "mixed structure - some with children, some without",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Standalone Step",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "parent1",
					Name:     "Parent Step",
					Category: "group",
					Children: []testkube.TestWorkflowSignature{
						{
							Ref:      "child1",
							Name:     "Child Step 1",
							Category: "integration",
							Optional: true,
							Negative: false,
						},
						{
							Ref:      "child2",
							Name:     "Child Step 2",
							Category: "integration",
							Optional: false,
							Negative: true,
						},
					},
				},
				{
					Ref:      "step2",
					Name:     "Another Standalone Step",
					Category: "e2e",
					Optional: true,
					Negative: false,
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Standalone Step",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "child1",
					Name:     "Child Step 1",
					Category: "integration",
					Optional: true,
					Negative: false,
				},
				{
					Ref:      "child2",
					Name:     "Child Step 2",
					Category: "integration",
					Optional: false,
					Negative: true,
				},
				{
					Ref:      "step2",
					Name:     "Another Standalone Step",
					Category: "e2e",
					Optional: true,
					Negative: false,
				},
			},
		},
		{
			name: "complex nested structure",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "root1",
					Name:     "Root 1",
					Category: "suite",
					Children: []testkube.TestWorkflowSignature{
						{
							Ref:      "branch1",
							Name:     "Branch 1",
							Category: "group",
							Children: []testkube.TestWorkflowSignature{
								{
									Ref:      "leaf1",
									Name:     "Leaf 1",
									Category: "unit",
									Optional: false,
									Negative: false,
								},
							},
						},
						{
							Ref:      "branch2",
							Name:     "Branch 2",
							Category: "group",
							Children: []testkube.TestWorkflowSignature{
								{
									Ref:      "leaf2",
									Name:     "Leaf 2",
									Category: "integration",
									Optional: true,
									Negative: false,
								},
								{
									Ref:      "leaf3",
									Name:     "Leaf 3",
									Category: "integration",
									Optional: false,
									Negative: true,
								},
							},
						},
					},
				},
				{
					Ref:      "root2",
					Name:     "Root 2",
					Category: "unit",
					Optional: true,
					Negative: false,
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "leaf1",
					Name:     "Leaf 1",
					Category: "unit",
					Optional: false,
					Negative: false,
				},
				{
					Ref:      "leaf2",
					Name:     "Leaf 2",
					Category: "integration",
					Optional: true,
					Negative: false,
				},
				{
					Ref:      "leaf3",
					Name:     "Leaf 3",
					Category: "integration",
					Optional: false,
					Negative: true,
				},
				{
					Ref:      "root2",
					Name:     "Root 2",
					Category: "unit",
					Optional: true,
					Negative: false,
				},
			},
		},
		{
			name: "signature with empty children slice",
			input: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Step with empty children",
					Category: "unit",
					Optional: false,
					Negative: false,
					Children: []testkube.TestWorkflowSignature{}, // explicitly empty, not nil
				},
			},
			expected: []testkube.TestWorkflowSignature{
				{
					Ref:      "step1",
					Name:     "Step with empty children",
					Category: "unit",
					Optional: false,
					Negative: false,
					Children: []testkube.TestWorkflowSignature{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenSignatures(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FlattenSignatures() = %+v, want %+v", result, tt.expected)
			}

			// Verify that the result slice is not nil even for empty input
			if result == nil {
				t.Error("FlattenSignatures() returned nil, expected empty slice")
			}
		})
	}
}
