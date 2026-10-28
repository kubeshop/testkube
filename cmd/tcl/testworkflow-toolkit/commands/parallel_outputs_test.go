// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
// 	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/paralleldata"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func outputsInstruction(values map[string]interface{}) *testkube.TestWorkflowOutput {
	return &testkube.TestWorkflowOutput{Ref: "step", Name: executiondata.OutputsInstructionName, Value: values}
}

func TestWorkerOutputs(t *testing.T) {
	t.Run("a later step wins and a replay changes nothing", func(t *testing.T) {
		outputs := newWorkerOutputs()
		outputs.merge(1, *outputsInstruction(map[string]interface{}{"duration": "1", "name": "a"}))
		outputs.merge(1, *outputsInstruction(map[string]interface{}{"duration": "2"}))
		outputs.merge(1, *outputsInstruction(map[string]interface{}{"duration": "2"}))
		outputs.finish(1, common.Ptr(testkube.FAILED_TestWorkflowStatus))

		assert.Equal(t, paralleldata.Group{
			Count: 3,
			Workers: []paralleldata.Worker{{
				Index:   1,
				Status:  testkube.FAILED_TestWorkflowStatus,
				Outputs: map[string]string{"duration": "2", "name": "a"},
			}},
		}, outputs.group(3))
	})

	t.Run("ignores other instructions", func(t *testing.T) {
		outputs := newWorkerOutputs()
		outputs.merge(0, testkube.TestWorkflowOutput{Name: "parallel", Value: map[string]interface{}{"duration": "1"}})
		assert.Empty(t, outputs.group(1).Workers[0].Outputs)
	})

	t.Run("a nil collector reads nothing", func(t *testing.T) {
		var outputs *workerOutputs
		outputs.merge(0, *outputsInstruction(map[string]interface{}{"duration": "1"}))
		outputs.finish(0, common.Ptr(testkube.PASSED_TestWorkflowStatus))
		assert.Equal(t, paralleldata.Group{Count: 1, Workers: []paralleldata.Worker{}}, outputs.group(1))
	})
}

func TestProcessWorkerNotificationsCollectsOutputs(t *testing.T) {
	cfg := config.NewConfigV2ForTest(testworkflowconfig.InternalConfig{
		Resource: testworkflowconfig.ResourceConfig{Id: "test", RootId: "/tmp", FsPrefix: "test"},
	})
	executor := NewWorkerExecutor(nil, NewMockRegistry(), make(chan Update, 10), nil, cfg)
	executor.outputs = newWorkerOutputs()

	watcher := executionworkertypes.NewStatusNotificationsWatcher()
	go func() {
		watcher.Send(executionworkertypes.StatusNotification{Output: outputsInstruction(map[string]interface{}{"duration": "12"})})
		watcher.Send(executionworkertypes.StatusNotification{Result: &testkube.TestWorkflowResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStatus), FinishedAt: time.Now()}})
		watcher.Close(nil)
	}()

	var lastResult testkube.TestWorkflowResult
	passed, err := executor.processWorkerNotifications(context.Background(), watcher, WorkerSpec{Index: 0}, nil, func(...string) {}, &lastResult)
	require.NoError(t, err)
	assert.True(t, passed)

	group := executor.outputs.group(1)
	require.Len(t, group.Workers, 1)
	assert.Equal(t, map[string]string{"duration": "12"}, group.Workers[0].Outputs)
}

// parentGroups are the parallel steps that finished in the pod of the parallel step.
var parentGroups = map[string]paralleldata.Group{
	"seed":    {Count: 1, Workers: []paralleldata.Worker{{Index: 0, Outputs: map[string]string{"value": "from-parent", "label": "seed"}}}},
	"shard_0": {Count: 1, Workers: []paralleldata.Worker{{Index: 0, Outputs: map[string]string{"value": "from-parent"}}}},
}

func parallelStep(name, id string, steps ...testworkflowsv1.Step) testworkflowsv1.Step {
	return testworkflowsv1.Step{StepMeta: testworkflowsv1.StepMeta{Name: name, Id: id}, Parallel: &testworkflowsv1.StepParallel{Steps: steps}}
}

func shellStep(name, shell string) testworkflowsv1.Step {
	return testworkflowsv1.Step{StepMeta: testworkflowsv1.StepMeta{Name: name}, StepOperations: testworkflowsv1.StepOperations{Shell: shell}}
}

const readSeed = `echo {{ parallel("seed", 0).outputs.value }}`

func TestResolveWorkerSpec(t *testing.T) {
	worker0 := expressions.NewMachine().Register("index", 0)

	t.Run("reads a parent group the worker does not declare", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{shellStep("Read", readSeed)}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Equal(t, "echo from-parent", spec.Steps[0].Shell)
	})

	t.Run("the worker's own step shadows the parent group", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("", "seed"),
			shellStep("Read", readSeed),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Contains(t, spec.Steps[1].Shell, `parallel("seed",0)`)
	})

	t.Run("the id is derived from the name as the worker resolves it", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("Shard {{ index }}", ""),
			shellStep("Read", `echo {{ parallel("shard_0", 0).outputs.value }}`),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Equal(t, "Shard 0", spec.Steps[0].Name)
		assert.Contains(t, spec.Steps[1].Shell, `parallel("shard_0",0)`)
	})

	t.Run("a name that reads a parent group is saved before the ids are read", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep(`{{ parallel("seed", 0).outputs.label }}`, ""),
			shellStep("Read", readSeed),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Equal(t, "seed", spec.Steps[0].Name)
		assert.Contains(t, spec.Steps[1].Shell, `parallel("seed",0)`)
	})

	t.Run("a step of a nested worker does not shadow the parent group around it", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("", "inner",
				parallelStep("", "seed"),
				shellStep("Read in the nested worker", readSeed),
			),
			shellStep("Read", readSeed),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Equal(t, "echo from-parent", spec.Steps[1].Shell)
		assert.Contains(t, spec.Steps[0].Parallel.Steps[1].Shell, `parallel("seed",0)`)
	})

	t.Run("a step of a grandchild worker does not shadow the parent group around it", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("", "inner",
				parallelStep("", "deeper", parallelStep("", "seed")),
				shellStep("Read in the nested worker", readSeed),
			),
			shellStep("Read", readSeed),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Equal(t, "echo from-parent", spec.Steps[1].Shell)
		assert.Equal(t, "echo from-parent", spec.Steps[0].Parallel.Steps[1].Shell)
	})

	t.Run("a step of the worker shadows the parent group in its nested workers", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("", "seed"),
			parallelStep("", "inner", shellStep("Read in the nested worker", readSeed)),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Contains(t, spec.Steps[1].Parallel.Steps[0].Shell, `parallel("seed",0)`)
	})

	t.Run("an unresolved name defers parallel() to the final resolution", func(t *testing.T) {
		spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
			parallelStep("Shard {{ later }}", ""),
			shellStep("Read", `echo {{ parallel("shard_0", 0).outputs.value }} {{ parallel("seed", 0).outputs.value }}`),
		}}
		require.NoError(t, resolveWorkerSpec(spec, parentGroups, false, worker0))
		assert.Contains(t, spec.Steps[1].Shell, `parallel("seed",0)`)

		final := expressions.NewMachine().Register("later", 0)
		workflow := spec.NewTestWorkflowSpec()
		require.NoError(t, resolveWorkerSpec(workflow, parentGroups, true, final))
		assert.Equal(t, "Shard 0", workflow.Steps[0].Name)
		assert.Contains(t, workflow.Steps[1].Shell, `parallel("shard_0",0)`)
		assert.Contains(t, workflow.Steps[1].Shell, "from-parent")
	})
}

func TestFinishedStatus(t *testing.T) {
	assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *finishedStatus(testkube.TestWorkflowResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStatus)}))
	assert.Equal(t, testkube.FAILED_TestWorkflowStatus, *finishedStatus(testkube.TestWorkflowResult{Status: common.Ptr(testkube.FAILED_TestWorkflowStatus)}))
	assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *finishedStatus(testkube.TestWorkflowResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStatus)}))
	assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *finishedStatus(testkube.TestWorkflowResult{}))
}

func TestInterruptedWatchLeavesANonterminalStatus(t *testing.T) {
	cfg := config.NewConfigV2ForTest(testworkflowconfig.InternalConfig{
		Resource: testworkflowconfig.ResourceConfig{Id: "test", RootId: "/tmp", FsPrefix: "test"},
	})
	executor := NewWorkerExecutor(nil, NewMockRegistry(), make(chan Update, 10), nil, cfg)

	watcher := executionworkertypes.NewStatusNotificationsWatcher()
	go func() {
		watcher.Send(executionworkertypes.StatusNotification{Result: &testkube.TestWorkflowResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStatus), Initialization: &testkube.TestWorkflowStepResult{}}})
		watcher.Close(errors.New("watch interrupted"))
	}()

	var lastResult testkube.TestWorkflowResult
	passed, err := executor.processWorkerNotifications(context.Background(), watcher, WorkerSpec{Index: 0}, nil, func(...string) {}, &lastResult)
	require.Error(t, err)
	assert.False(t, passed)
	// The watch error returns before the fallback, which is why the status is normalized before it is recorded
	assert.Equal(t, testkube.RUNNING_TestWorkflowStatus, *lastResult.Status)
	assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *finishedStatus(lastResult))
}

func TestResolveWorkerSpecSharesTheNamespaceOfANestedWorker(t *testing.T) {
	// The setup and the steps of the nested worker share one namespace, so the second "Shards" is shards_1
	spec := &testworkflowsv1.StepParallel{Steps: []testworkflowsv1.Step{
		{StepMeta: testworkflowsv1.StepMeta{Id: "inner"}, Parallel: &testworkflowsv1.StepParallel{
			Setup: []testworkflowsv1.Step{parallelStep("Shards", "")},
			Steps: []testworkflowsv1.Step{
				parallelStep("Shards", ""),
				shellStep("Read", `echo {{ parallel("shards_1", 0).outputs.value }}`),
			},
		}},
	}}
	groups := map[string]paralleldata.Group{"shards_1": {Count: 1, Workers: []paralleldata.Worker{{Index: 0, Outputs: map[string]string{"value": "from-parent"}}}}}
	require.NoError(t, resolveWorkerSpec(spec, groups, false))
	assert.Contains(t, spec.Steps[0].Parallel.Steps[1].Shell, `parallel("shards_1",0)`)
}
