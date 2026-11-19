// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func TestProcessAgent(t *testing.T) {
	tests := []struct {
		name string
		step testworkflowsv1.Step
		// wantSpec is the step spec that the toolkit must get. Nil means no stage.
		wantSpec *testworkflowsv1.StepAgent
		wantErr  string
	}{
		{
			name: "ignores a step with no agent operation",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{Shell: "true"}},
		},
		{
			name: "fails when the AI Agent has no name",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{
				Agent: &testworkflowsv1.StepAgent{Prompt: "Find the cause."},
			}},
			wantErr: "no AI Agent name",
		},
		{
			name: "gives the toolkit the spec with its expressions not resolved",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{
				Agent: &testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "Read {{ step.tests.outputs.summary }}."},
			}},
			wantSpec: &testworkflowsv1.StepAgent{Name: "failure-triage", Prompt: "Read {{ step.tests.outputs.summary }}."},
		},
		{
			name: "accepts a step with no prompt",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{
				Agent: &testworkflowsv1.StepAgent{Name: "failure-triage"},
			}},
			wantSpec: &testworkflowsv1.StepAgent{Name: "failure-triage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessAgent(nil, testworkflowprocessor.NewIntermediate(""), stage.NewContainer(), tt.step)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantSpec == nil {
				assert.Nil(t, got)
				return
			}

			containerStage, ok := got.(stage.ContainerStage)
			require.True(t, ok, "the stage runs a container")
			container := containerStage.Container()
			assert.Equal(t, []string{constants.DefaultToolkitPath, "agent"}, container.Command())
			require.Len(t, container.Args(), 2)
			assert.Equal(t, "--base64", container.Args()[0])

			var spec testworkflowsv1.StepAgent
			require.NoError(t, expressionstcl.DecodeBase64JSON(container.Args()[1], &spec))
			assert.Equal(t, *tt.wantSpec, spec)
		})
	}
}
