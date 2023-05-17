package commands

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	c "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"

	"github.com/rivo/tview"
)

var (
	executionsTable    *tview.Table
	testsTable         *tview.Table
	executionsView     *tview.TextView
	app                = tview.NewApplication()
	pages              = tview.NewPages()
	header             = newPrimitive("Testkube interactive demo")
	currentTest        string
	currentExecutionId string
)

func NewInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "interactive",
		Aliases:     []string{"i"},
		Short:       "Interactive mode",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)

			executionsView = newPrimitive(ui.LogoString(), tcell.ColorPurple)
			executionsView.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					executionsView.SetText(ui.LogoString()).SetTextColor(tcell.ColorPurple)
					app.SetFocus(executionsTable)
				}
			})

			testsTable = NewTestsTable(client)
			loadDataToTestTable(testsTable, client)

			executionsTable = NewExecutionsTable(client)

			pages.AddPage("tests", testsTable, true, true)
			pages.AddPage("executions", executionsTable, true, true)
			pages.SwitchToPage("tests")

			grid := tview.NewGrid().
				SetRows(1, 0, 1).
				SetColumns(-1, -2).
				SetBorders(true).
				AddItem(header, 0, 0, 1, 2, 0, 0, false).
				AddItem(newPrimitive("Copyright (c) Testkube LLC"), 2, 0, 1, 2, 0, 0, false)

			// Layout for screens narrower than 100 cells (menu and side bar are hidden).
			grid.AddItem(pages, 1, 0, 1, 1, 0, 0, true)
			grid.AddItem(executionsView, 1, 1, 1, 1, 0, 0, true)

			app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				k := event.Key()
				if k == tcell.KeyCtrlR {
					execution, _ := client.ExecuteTest(currentTest, "", c.ExecuteTestOptions{})
					currentExecutionId = execution.Id
					executionsTable.Select(1, 0)
				}
				return event
			})

			go watchExecutions(client)
			go watchExecutionDetails(client)

			if err := app.SetRoot(grid, true).
				SetFocus(testsTable).
				Run(); err != nil {
				panic(err)
			}
		},
	}

	return cmd
}

func loadDataToTestTable(table *tview.Table, client c.Client) *tview.Table {
	data, err := client.ListTestWithExecutionSummaries("")
	ui.ExitOnError("listing tests", err)

	table.Clear()

	color := tcell.ColorWhite

	table.SetCell(0, 0,
		tview.NewTableCell("NAME                    ").
			SetTextColor(color).
			SetAlign(tview.AlignLeft))

	table.SetCell(0, 1,
		tview.NewTableCell("TYPE    ").
			SetTextColor(color).
			SetAlign(tview.AlignLeft))

	table.SetCell(0, 2,
		tview.NewTableCell("STATUS").
			SetTextColor(color).
			SetBackgroundColor(tcell.ColorDefault).
			SetAlign(tview.AlignLeft))

	for i, e := range data {
		table.SetCell(i+1, 0,
			tview.NewTableCell(e.Test.Name).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		table.SetCell(i+1, 1,
			tview.NewTableCell(e.Test.Type_).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		color := getStatusColor(e.LatestExecution)
		status := "not executed"
		if e.LatestExecution != nil {
			status = string(*e.LatestExecution.Status)
		}

		table.SetCell(i+1, 2,
			tview.NewTableCell(status).
				SetTextColor(color).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

	}

	return table
}

func loadDataToExecutionsTable(table *tview.Table, client c.Client, testName string) {
	data, err := client.ListExecutions(testName, 0, "")
	ui.ExitOnError("listing executions", err)

	table.Clear()

	table.SetCell(0, 0,
		tview.NewTableCell("NAME                    ").
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft))

	table.SetCell(0, 1,
		tview.NewTableCell("SPEED    ").
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft))

	table.SetCell(0, 2,
		tview.NewTableCell("STATUS").
			SetTextColor(tcell.ColorYellow).
			SetBackgroundColor(tcell.ColorDefault).
			SetAlign(tview.AlignLeft))

	for i, e := range data.Results {
		table.SetCell(i+1, 0,
			tview.NewTableCell(e.Name).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		var duration time.Duration
		if e.DurationMs > 0 {
			duration = time.Duration(e.DurationMs) * time.Millisecond
		} else {
			duration = time.Since(e.StartTime)
		}
		duration = duration.Round(time.Millisecond * 100)

		table.SetCell(i+1, 1,
			tview.NewTableCell(duration.String()).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		color := tcell.ColorWhite
		switch string(*e.Status) {
		case "running":
			color = tcell.ColorBlueViolet
		case "passed":
			color = tcell.ColorGreen
		case "failed":
			color = tcell.ColorRed
		default:
			color = tcell.ColorGray
		}

		table.SetCell(i+1, 2,
			tview.NewTableCell(string(*e.Status)).
				SetTextColor(color).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

	}
}

func newPrimitive(text string, col ...tcell.Color) *tview.TextView {
	color := tcell.ColorWhite
	if len(col) > 0 {
		color = col[0]
	}

	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetTextColor(color).
		SetText(text)
}

func loadExecution(client c.Client, executionId string) {

	executionsView.Clear()
	executionsView.SetText("loading execution...")
	executionsView.SetTextColor(tcell.ColorWhite)
	execution, _ := client.GetExecution(executionId)

	executionsView.SetText(
		fmt.Sprintf(`Execution: %s
Status: %s
----------------
Log: 
		`,
			execution.Name,
			*execution.ExecutionResult.Status,
		))
	executionsView.SetTextAlign(tview.AlignLeft)

	if *execution.ExecutionResult.Status == testkube.RUNNING_ExecutionStatus {
		logs, _ := client.Logs(currentExecutionId)
		for l := range logs {
			executionsView.SetText(executionsView.GetText(false) + l.Content)
			executionsView.ScrollToEnd()
		}
		return
	} else {
		executionsView.SetText(executionsView.GetText(false) + execution.ExecutionResult.Output)
		executionsView.ScrollToBeginning()
	}

}

func getStatusColor(result *testkube.ExecutionSummary) tcell.Color {
	if result != nil {
		switch string(*result.Status) {
		case "running":
			return tcell.ColorBlueViolet
		case "passed":
			return tcell.ColorGreen
		case "failed":
			return tcell.ColorRed
		default:
			return tcell.ColorGray
		}
	}
	return tcell.ColorGray
}

func NewTestsTable(client c.Client) *tview.Table {
	testsTable := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		Select(0, 0).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				app.Stop()
			}
			if key == tcell.KeyEnter {

			}
		})

	testsTable.SetSelectedFunc(func(row int, column int) {
		currentTest = testsTable.GetCell(row, 0).Text
		loadDataToExecutionsTable(executionsTable, client, currentTest)
		pages.SwitchToPage("executions")
		executionsTable.ScrollToBeginning()
	})

	return testsTable
}

func NewExecutionsTable(client c.Client) *tview.Table {
	executionsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		Select(0, 0).
		SetDoneFunc(func(key tcell.Key) {
			header.SetText(fmt.Sprintf("%+v", key))
			if key == tcell.KeyEscape {
				pages.SwitchToPage("tests")
			}
			if key == tcell.KeyEnter {
			}
		})
	executionsTable.SetSelectedFunc(func(row int, column int) {
		currentExecutionId = executionsTable.GetCell(row, 0).Text
		loadExecution(client, currentExecutionId)
		app.SetFocus(executionsView)
	})

	return executionsTable
}

func watchExecutions(client c.Client) {
	for {
		if currentTest != "" {
			header.SetText(fmt.Sprintf("Testkube interactive demo - %s", currentTest))
			loadDataToExecutionsTable(executionsTable, client, currentTest)
			app.ForceDraw()
		}
		time.Sleep(time.Second)
	}
}

func watchExecutionDetails(client c.Client) {
	if currentExecutionId != "" {
		loadExecution(client, currentExecutionId)
	}
}
