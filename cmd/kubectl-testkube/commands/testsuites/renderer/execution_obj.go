package renderer

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestSuiteExecutionRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.TestSuiteExecution)
	if !ok {
		return fmt.Errorf("can't render execution, expecrted obj to be testkube.Execution but got '%T'", obj)
	}

	ui.Warn("Id:      ", execution.Id)
	ui.Warn("Name:    ", execution.Name)
	if execution.Status != nil {
		ui.Warn("Status:  ", string(*execution.Status))
	}
	ui.Warn("Duration:", execution.CalculateDuration().String()+"\n")
	ui.Warn("Labels:  ", testkube.MapToString(execution.Labels))
	if execution.RunningContext != nil {
		ui.Warn("Running context:")
		ui.Warn("Type:   ", execution.RunningContext.Type_)
		ui.Warn("Context:", execution.RunningContext.Context)
	}

	info, err := client.GetServerInfo()
	ui.ExitOnError("getting server info", err)

	render.PrintTestSuiteExecutionURIs(&execution, info.DashboardUri)

	ui.Table(execution, os.Stdout)

	ui.NL()

	return nil
}
