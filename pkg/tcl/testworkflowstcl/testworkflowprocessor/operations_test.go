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
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func parallelArgs(t *testing.T, step testworkflowsv1.Step) []string {
	t.Helper()
	result, err := ProcessParallel(nil, testworkflowprocessor.NewIntermediate(""), stage.NewContainer(), step)
	require.NoError(t, err)
	containerStage, ok := result.(stage.ContainerStage)
	require.True(t, ok)
	return containerStage.Container().Args()
}

func TestProcessParallelPassesStepId(t *testing.T) {
	args := parallelArgs(t, testworkflowsv1.Step{
		StepMeta: testworkflowsv1.StepMeta{Id: "shard"},
		Parallel: &testworkflowsv1.StepParallel{},
	})
	require.Len(t, args, 4)
	assert.Equal(t, "--base64", args[0])
	assert.Equal(t, []string{"--id", "shard"}, args[2:])
}

func TestProcessParallelWithoutStepId(t *testing.T) {
	args := parallelArgs(t, testworkflowsv1.Step{Parallel: &testworkflowsv1.StepParallel{}})
	assert.Len(t, args, 2)
	assert.NotContains(t, args, "--id")
}
