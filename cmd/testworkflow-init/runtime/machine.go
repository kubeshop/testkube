package runtime

import (
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func GetInternalTestWorkflowMachine() expressions.Machine {
	return expressions.CombinedMachines(data.RefSuccessMachine, data.AliasMachine,
		data.GetBaseTestWorkflowMachine(),
		data.ExecutionMachine(),
		credentials.NewCredentialMachine(data.Credentials(), func(_ string, value string) {
			orchestration.Setup.AddSensitiveWords(value)
			output.Std.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
		}))
}
