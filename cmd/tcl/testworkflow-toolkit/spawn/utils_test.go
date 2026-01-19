// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestCreateBaseMachineWithoutEnv(t *testing.T) {
	t.Run("env expressions remain unresolved", func(t *testing.T) {
		cfg := &testworkflowconfig.InternalConfig{
			Execution: testworkflowconfig.ExecutionConfig{
				OrganizationSlug: "test-org",
				EnvironmentSlug:  "test-env",
			},
		}

		machine := createBaseMachineWithoutEnv(cfg)

		// Verify env.* expressions are NOT resolved
		expr, err := expressions.Compile("env.TEST_VAR")
		require.NoError(t, err)
		result, err := expr.Resolve(machine)
		require.NoError(t, err)
		assert.Equal(t, "{{env.TEST_VAR}}", result.Template())
	})

	t.Run("uses ID when slug is empty", func(t *testing.T) {
		cfg := &testworkflowconfig.InternalConfig{
			Execution: testworkflowconfig.ExecutionConfig{
				OrganizationId:   "org-123",
				OrganizationSlug: "", // empty
				EnvironmentId:    "env-456",
				EnvironmentSlug:  "", // empty
			},
		}

		machine := createBaseMachineWithoutEnv(cfg)

		// Should still create a valid machine using IDs
		assert.NotNil(t, machine)

		// Verify organization.id is accessible (testing the fallback logic)
		expr, err := expressions.Compile("organization.id")
		require.NoError(t, err)
		result, err := expr.Resolve(machine)
		require.NoError(t, err)
		value, err := result.Static().StringValue()
		require.NoError(t, err)
		assert.Equal(t, "org-123", value)
	})

	t.Run("non-env expressions work normally", func(t *testing.T) {
		cfg := &testworkflowconfig.InternalConfig{
			Execution: testworkflowconfig.ExecutionConfig{
				Id: "exec-123",
			},
			Workflow: testworkflowconfig.WorkflowConfig{
				Name: "test-workflow",
			},
		}

		machine := createBaseMachineWithoutEnv(cfg)

		// execution.id should resolve
		execExpr, err := expressions.Compile("execution.id")
		require.NoError(t, err)
		execResult, err := execExpr.Resolve(machine)
		require.NoError(t, err)
		execValue, err := execResult.Static().StringValue()
		require.NoError(t, err)
		assert.Equal(t, "exec-123", execValue)

		// workflow.name should resolve
		wfExpr, err := expressions.Compile("workflow.name")
		require.NoError(t, err)
		wfResult, err := wfExpr.Resolve(machine)
		require.NoError(t, err)
		wfValue, err := wfResult.Static().StringValue()
		require.NoError(t, err)
		assert.Equal(t, "test-workflow", wfValue)
	})
}
