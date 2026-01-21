package checker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
)

type CheckResult struct {
	Description string `json:"description"`
	Result      string `json:"result"`
	Error       string `json:"error,omitempty"`
}

type CheckSuiteOutput struct {
	CheckSuiteName CheckSuiteName `json:"checksuitename"`
	CheckResults   []CheckResult  `json:"checkresults"`
}

func PrintTableResults(checkSuiteOutputs []CheckSuiteOutput) {
	for _, suite := range checkSuiteOutputs {
		ui.Info(string(suite.CheckSuiteName))
		underline := ""
		for i := 0; i < len(suite.CheckSuiteName); i++ {
			underline += "-"
		}
		ui.Info(underline)
		for _, check := range suite.CheckResults {
			if len(check.Error) != 0 {
				fmt.Printf("%s %s - %s %s\n", ui.IconCross, check.Description, ui.Red("Check Failed:"), check.Error)
			} else {
				fmt.Printf("%s %s - %s\n", ui.IconCheckMark, check.Description, ui.Green("Check Passed"))
			}
		}
		ui.NL()
	}
}

func PrintJSONResults(success bool, checkSuiteOutputs []CheckSuiteOutput) {
	finalJSONOutput := map[string]interface{}{
		"success":           success,
		"checkSuiteResults": checkSuiteOutputs,
	}

	resultJSON, err := json.MarshalIndent(finalJSONOutput, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serialize JSON output for check results: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "%s\n", resultJSON)
}

func PrintFinalResults(success bool, suiteResults []CheckSuiteOutput, outputFormat string) {
	if outputFormat == "json" {
		PrintJSONResults(success, suiteResults)
		return
	}
	PrintTableResults(suiteResults)
	if success {
		ui.Success("All checks passed successfully!")
	} else {
		ui.Alert(ui.LightRed("Checks completed with errors. Please review the output above.") + ui.IconError)
	}
}
