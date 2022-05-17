package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestSuiteRenderer(ui *ui.UI, obj interface{}) error {
	ts, ok := obj.(testkube.TestSuite)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestSuite in RenderObj for test suite", obj)
	}

	ui.Warn("Name:     ", ts.Name)
	ui.Warn("Namespace:", ts.Namespace)
	if len(ts.Labels) > 0 {
		ui.NL()
		ui.Warn("Labels:   ", testkube.LabelsToString(ts.Labels))
	}
	if ts.Schedule != "" {
		ui.NL()
		ui.Warn("Schedule: ", ts.Schedule)
	}

	if len(ts.Variables) > 0 {
		ui.NL()
		ui.Warn("Variables: ", fmt.Sprintf("%d", len(ts.Variables)))
		for _, v := range ts.Variables {
			t := ""
			if *v.Type_ == *testkube.VariableTypeSecret {
				t = "🔒"
			}
			ui.Info("-", fmt.Sprintf("%s='%s' %s", v.Name, v.Value, t))
		}

	}

	steps := append(ts.Before, ts.Steps...)
	steps = append(steps, ts.After...)

	ui.NL()
	ui.Warn("Test steps:", fmt.Sprintf("%d", len(steps)))
	d := [][]string{{"Name", "Stop on failure", "Type"}}
	for _, step := range steps {
		d = append(d, []string{
			step.FullName(),
			fmt.Sprintf("%v", step.StopTestOnFailure),
			string(*step.Type()),
		})
	}

	ui.Table(ui.NewArrayTable(d), ui.Writer)
	ui.NL()

	return nil

}
