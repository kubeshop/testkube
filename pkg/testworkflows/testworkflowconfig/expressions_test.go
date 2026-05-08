package testworkflowconfig

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestCreateExecutionMachine_RunningContextVariables(t *testing.T) {
	actorType := testkube.USER_TestWorkflowRunningContextActorType
	interfaceType := testkube.CLI_TestWorkflowRunningContextInterfaceType

	cfg := &ExecutionConfig{
		Id: "test-id",
		RunningContext: &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Name:  "test-user",
				Type_: &actorType,
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: &interfaceType,
			},
		},
	}

	machine := CreateExecutionMachine(cfg)

	actorExpr, err := expressions.Compile("execution.runningContext.actor.name")
	require.NoError(t, err)
	actorResult, err := actorExpr.Resolve(machine)
	require.NoError(t, err)
	actorName, err := actorResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, "test-user", actorName)

	actorTypeExpr, err := expressions.Compile("execution.runningContext.actor.type")
	require.NoError(t, err)
	actorTypeResult, err := actorTypeExpr.Resolve(machine)
	require.NoError(t, err)
	actorTypeValue, err := actorTypeResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, string(actorType), actorTypeValue)

	interfaceExpr, err := expressions.Compile("execution.runningContext.interface.type")
	require.NoError(t, err)
	interfaceResult, err := interfaceExpr.Resolve(machine)
	require.NoError(t, err)
	interfaceTypeValue, err := interfaceResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, string(interfaceType), interfaceTypeValue)
}

func TestCreateExecutionMachine_RunningContextMissing(t *testing.T) {
	cfg := &ExecutionConfig{}

	machine := CreateExecutionMachine(cfg)

	expr, err := expressions.Compile("execution.runningContext.actor.name")
	require.NoError(t, err)

	result, err := expr.Resolve(machine)
	require.NoError(t, err)

	require.Equal(t, "", result.Template())
}
