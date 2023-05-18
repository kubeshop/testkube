package commands

import (
	"fmt"
	"strings"
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
	executionsTable        *tview.Table
	testsTable             *tview.Table
	executionsView         *tview.TextView
	app                    = tview.NewApplication()
	pages                  = tview.NewPages()
	fpages                 = tview.NewPages()
	header                 = newPrimitive("Testkube interactive demo")
	executionsFooter       = newPrimitive("Executions: [f] -> failed [a] -> all ")
	testsFooter            = newPrimitive("Tests: [f] -> failed [a] -> all ")
	currentTest            string
	currentExecutionId     string
	currentExecutionStatus string
	currentTestStatus      string
)

func NewInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "interactive",
		Aliases:     []string{"i"},
		Short:       "Interactive mode",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)

			executionsView = NewExecutionsView()

			testsTable = NewTestsTable(client)
			loadDataToTestTable(testsTable, client)

			executionsTable = NewExecutionsTable(client)

			pages.AddPage("tests", testsTable, true, true)
			pages.AddPage("executions", executionsTable, true, true)
			pages.SwitchToPage("tests")

			fpages.AddPage("tests", testsFooter, true, true)
			fpages.AddPage("executions", executionsFooter, true, true)
			fpages.SwitchToPage("tests")

			grid := tview.NewGrid().
				SetRows(1, 0, 1).
				SetColumns(-3, -4).
				SetBorders(true).
				AddItem(header, 0, 0, 1, 2, 0, 0, false).
				AddItem(fpages, 2, 0, 1, 2, 0, 0, false)

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
				if k == tcell.KeyCtrlL {
					if name, _ := pages.GetFrontPage(); name == "tests" {
						loadDataToTestTable(testsTable, client)
					} else {
						loadDataToExecutionsTable(executionsTable, client, currentTest)
					}

					if currentExecutionId != "" {
						loadExecution(client, currentExecutionId)
					}

				}
				return event
			})

			// go watchTests(client)
			go watchExecutions(client)
			// go watchExecutionDetails(client)

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

	i := 1
	for _, e := range data {
		if currentTestStatus != "" && *e.LatestExecution.Status != testkube.ExecutionStatus(currentTestStatus) {
			continue
		}
		table.SetCell(i, 0,
			tview.NewTableCell(e.Test.Name).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		tt := strings.Split(e.Test.Type_, "/")[0]

		table.SetCell(i, 1,
			tview.NewTableCell(tt).
				SetTextColor(tcell.ColorGray).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		color := getStatusColor(e.LatestExecution)
		status := "not executed"
		if e.LatestExecution != nil {
			status = string(*e.LatestExecution.Status)
		}

		table.SetCell(i, 2,
			tview.NewTableCell(status).
				SetTextColor(color).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		i++

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

	i := 1
	for _, e := range data.Results {
		// filter out status here as there is no option in api client
		if currentExecutionStatus != "" && *e.Status != testkube.ExecutionStatus(currentExecutionStatus) {
			continue
		}

		table.SetCell(i, 0,
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

		table.SetCell(i, 1,
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

		table.SetCell(i, 2,
			tview.NewTableCell(string(*e.Status)).
				SetTextColor(color).
				SetBackgroundColor(tcell.ColorDefault).
				SetAlign(tview.AlignLeft))

		i++
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
	execution, err := client.GetExecution(executionId)

	if err != nil {
		executionsView.SetText(fmt.Sprintf("error: [red]%s[white]", err.Error()))
		return
	}

	if execution.ExecutionResult == nil {
		return
	}

	status := string(*execution.ExecutionResult.Status)

	executionsView.SetText(
		fmt.Sprintf(`execution: %s
status: %s
----------------
log: 
		`,
			execution.Name,
			status,
		))

	executionsView.SetTextAlign(tview.AlignLeft)

	// TODO: some issues with getting logs
	// if status == string(testkube.RUNNING_ExecutionStatus) {
	// 	logs, _ := client.Logs(currentExecutionId)
	// 	for l := range logs {
	// 		executionsView.SetText(executionsView.GetText(false) + l.Content)
	// 		executionsView.ScrollToEnd()
	// 	}
	// 	return
	// } else {
	logs := "getting logs ..."
	if execution.ExecutionResult.Output != "" {
		logs = execution.ExecutionResult.Output
	}

	logs = strings.Replace(logs, "\n\n\n\n", "", -1)
	executionsView.SetText(executionsView.GetText(false) + logs)
	executionsView.ScrollToBeginning()
	// }

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

func NewExecutionsView() *tview.TextView {
	executionsView = newPrimitive(ui.LogoString(), tcell.ColorPurple)
	executionsView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			executionsView.SetText(ui.LogoString()).SetTextColor(tcell.ColorPurple)
			app.SetFocus(executionsTable)
		}
	})
	return executionsView
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
		})

	testsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlL {
			loadDataToTestTable(testsTable, client)
			// loadDataToExecutionsTable(executionsTable, client, currentTest)
		}
		if event.Rune() == 'f' {
			currentTestStatus = "failed"
			loadDataToTestTable(testsTable, client)
		}
		if event.Rune() == 'a' {
			currentTestStatus = ""
			loadDataToTestTable(testsTable, client)
		}

		return event
	})

	testsTable.SetSelectedFunc(func(row int, column int) {
		currentTest = testsTable.GetCell(row, 0).Text
		loadDataToExecutionsTable(executionsTable, client, currentTest)
		pages.SwitchToPage("executions")
		fpages.SwitchToPage("executions")
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
			if key == tcell.KeyEscape {
				executionsView.SetText(ui.LogoString()).SetTextColor(tcell.ColorPurple)
				pages.SwitchToPage("tests")
				fpages.SwitchToPage("tests")
				loadDataToTestTable(testsTable, client)
			}
		})
	executionsTable.SetSelectedFunc(func(row int, column int) {
		currentExecutionId = executionsTable.GetCell(row, 0).Text
		loadExecution(client, currentExecutionId)
		app.SetFocus(executionsView)
	})
	executionsTable.SetSelectionChangedFunc(func(row int, column int) {
	})
	executionsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlL {
			loadDataToExecutionsTable(executionsTable, client, currentTest)
		}

		if event.Rune() == 'f' {
			currentExecutionStatus = "failed"
			loadDataToExecutionsTable(executionsTable, client, currentTest)
		}
		if event.Rune() == 'a' {
			currentExecutionStatus = ""
			loadDataToExecutionsTable(executionsTable, client, currentTest)
		}

		return event
	})

	return executionsTable
}

func watchExecutions(client c.Client) {
	for {
		if currentTest != "" {
			header.SetText(fmt.Sprintf("Testkube interactive demo - %s", currentTest))
			loadDataToExecutionsTable(executionsTable, client, currentTest)
		}
		time.Sleep(time.Second)
	}
}

func watchTests(client c.Client) {
	for {
		loadDataToTestTable(testsTable, client)
		time.Sleep(time.Second)
	}
}

func watchExecutionDetails(client c.Client) {
	for {
		if currentExecutionId != "" {
			loadExecution(client, currentExecutionId)
		}
		time.Sleep(time.Second)
	}
}
