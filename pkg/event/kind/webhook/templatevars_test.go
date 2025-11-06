package webhook

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/utils"
)

func TestTemplateVars_ExposeRunningContext(t *testing.T) {
	actorType := testkube.USER_TestWorkflowRunningContextActorType

	event := testkube.Event{
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			RunningContext: &testkube.TestWorkflowRunningContext{
				Actor: &testkube.TestWorkflowRunningContextActor{
					Name:  "Jane Doe",
					Type_: &actorType,
				},
			},
		},
	}

	templateBody := "{{ .TestWorkflowExecution.RunningContext.Actor.Name }}|{{ .TestWorkflowExecution.RunningContext.Actor.Type_ }}"
	tmpl, err := utils.NewTemplate("runningContext").Parse(templateBody)
	require.NoError(t, err)

	var buffer bytes.Buffer
	err = tmpl.ExecuteTemplate(&buffer, "runningContext", NewTemplateVars(event, nil, nil))
	require.NoError(t, err)

	require.Equal(t, "Jane Doe|user", buffer.String())
}
