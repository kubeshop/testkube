package render

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils"
)

type OutputType string

const (
	OutputGoTemplate OutputType = "go"
	OutputJSON       OutputType = "json"
	OutputYAML       OutputType = "yaml"
	OutputPretty     OutputType = "pretty"
)

type CliObjRenderer func(client client.Client, ui *ui.UI, obj interface{}) error

func RenderJSON(obj interface{}, w io.Writer) error {
	return json.NewEncoder(w).Encode(obj)
}

func RenderYaml(obj interface{}, w io.Writer) error {
	return yaml.NewEncoder(w).Encode(obj)
}

func RenderGoTemplate(item interface{}, w io.Writer, tpl string) error {
	tmpl, err := utils.NewTemplate("result").Parse(tpl)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, item)
}

func RenderGoTemplateList(list []interface{}, w io.Writer, tpl string) error {
	tmpl, err := utils.NewTemplate("result").Parse(tpl)
	if err != nil {
		return err
	}

	for _, item := range list {
		err := tmpl.Execute(w, item)
		if err != nil {
			return err
		}
	}

	return nil
}

func RenderPrettyList(obj ui.TableData, w io.Writer) error {
	ui.NL()
	ui.Table(obj, w)
	ui.NL()
	return nil
}

func PrintTestWorkflowExecutionURIs(execution *testkube.TestWorkflowExecution) {
	cfg, err := config.Load()
	ui.ExitOnError("loading config file", err)

	if cfg.ContextType != config.ContextTypeCloud {
		return
	}

	if execution.Result == nil || !execution.Result.IsFinished() {
		return
	}

	ui.NL()
	workflowName := ""
	if execution.Workflow != nil {
		workflowName = execution.Workflow.Name
	}

	ui.ExecutionLink("Test Workflow URI:", fmt.Sprintf("%s/organization/%s/environment/%s/dashboard/test-workflows/%s",
		cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, workflowName))
	ui.ExecutionLink("Test Workflow Execution URI:", fmt.Sprintf("%s/organization/%s/environment/%s/dashboard/test-workflows/%s/execution/%s",
		cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId, workflowName, execution.Id))
	ui.NL()
}
