package tests

import (
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()

	client, err := client.GetClient(client.ClientType(clientType), namespace)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}

func printTestExecutionDetails(execution testkube.TestExecution) {
	ui.Warn("Name:", execution.Name+"\n")
	tab := [][]string{}

	for _, result := range execution.StepResults {
		step := (*result.Step)
		r := []string{step.FullName()}

		switch step.Type() {
		case testkube.EXECUTE_SCRIPT_TestStepType:
			if result.Execution != nil && result.Script != nil {
				status := string(*result.Execution.ExecutionResult.Status)
				switch status {
				case string(testkube.SUCCESS_TestStatus):
					status = ui.Green(status)
				case string(testkube.ERROR__TestStatus):
					status = ui.Red(status)
				}
				r = append(r, status)
			}
		case testkube.DELAY_TestStepType:
			r = append(r, "✓")
		}

		tab = append(tab, r)
	}

	ui.Table(ui.NewArrayTable(tab), os.Stdout)

	ui.NL()
	ui.NL()
}
