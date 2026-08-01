// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// The spec of an entry stays unresolved until its operation starts, so that its
// configuration can read an execution scheduled before it. This checks that the
// deferred resolution actually reaches the nested config values.
func TestDeferredSpecFinalization(t *testing.T) {
	spec := &testworkflowsv1.StepExecuteWorkflow{
		Name: "consumer",
		Config: map[string]intstr.IntOrString{
			"token": intstr.FromString(`{{ execution("p").outputs.token }}`),
		},
	}

	registry := executiondata.NewRegistry()
	machine := executiondata.NewMachine(executiondata.MachineOptions{Registry: registry})

	t.Run("fails while the referenced execution has not run", func(t *testing.T) {
		workflow := *spec.DeepCopy()
		err := expressions.Finalize(&workflow, machine)
		assert.ErrorContains(t, err, `unknown execution "p"`)
	})

	t.Run("resolves once the execution is registered", func(t *testing.T) {
		registry.Add(executiondata.Execution{
			Id:       "exec-1",
			Workflow: "producer",
			Alias:    "p",
			Outputs:  map[string]string{"token": "abc123"},
		})

		workflow := *spec.DeepCopy()
		require.NoError(t, expressions.Finalize(&workflow, machine))
		assert.Equal(t, "abc123", workflow.Config["token"].StrVal)
	})
}

func TestClaimExecutionRef(t *testing.T) {
	t.Run("an entry is addressed by its workflow name", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRef(claimed, "", "producer"))
		assert.Equal(t, map[string]string{"producer": "producer"}, claimed)
	})

	t.Run("an alias takes precedence over the workflow name", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRef(claimed, "p", "producer"))
		assert.Equal(t, map[string]string{"p": "producer"}, claimed)

		require.NoError(t, claimExecutionRef(claimed, "", "producer"),
			"the unaliased entry still claims the workflow name, so both stay addressable")
	})

	t.Run("rejects the same workflow listed twice without an alias", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRef(claimed, "", "producer"))

		err := claimExecutionRef(claimed, "", "producer")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicated execution reference "producer"`)
		assert.Contains(t, err.Error(), "set a unique 'as'")
	})

	t.Run("rejects two entries sharing an alias", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRef(claimed, "shared", "producer"))

		err := claimExecutionRef(claimed, "shared", "consumer")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `already used by "producer"`)
	})
}
