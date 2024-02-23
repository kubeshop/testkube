package testworkflowtemplates

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	common2 "github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateTestWorkflowTemplateCmd() *cobra.Command {
	var (
		name     string
		filePath string
		update   bool
	)

	cmd := &cobra.Command{
		Use:     "testworkflowtemplate",
		Aliases: []string{"testworkflowtemplates", "twt"},
		Args:    cobra.MaximumNArgs(0),
		Short:   "Create test workflow template",

		Run: func(cmd *cobra.Command, _ []string) {
			namespace := cmd.Flag("namespace").Value.String()

			var input io.Reader
			if filePath == "" {
				fi, err := os.Stdin.Stat()
				ui.ExitOnError("reading stdin", err)
				if fi.Mode()&os.ModeDevice != 0 {
					ui.Failf("you need to pass stdin or --file argument with file path")
				}
				input = cmd.InOrStdin()
			} else {
				file, err := os.Open(filePath)
				ui.ExitOnError("reading "+filePath+" file", err)
				input = file
			}

			bytes, err := io.ReadAll(input)
			ui.ExitOnError("reading input", err)

			obj := new(testworkflowsv1.TestWorkflowTemplate)
			err = common2.DeserializeCRD(obj, bytes)
			ui.ExitOnError("deserializing input", err)
			if obj.Kind != "" && obj.Kind != "TestWorkflowTemplate" {
				ui.Failf("Only TestWorkflowTemplate objects are accepted. Received: %s", obj.Kind)
			}
			common2.AppendTypeMeta("TestWorkflowTemplate", testworkflowsv1.GroupVersion, obj)
			obj.Namespace = namespace
			if name != "" {
				obj.Name = name
			}

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			workflow, _ := client.GetTestWorkflowTemplate(obj.Name)
			if workflow.Name != "" {
				if !update {
					ui.Failf("Test workflow template with name '%s' already exists in namespace %s, use --update flag for upsert", obj.Name, namespace)
				}
				_, err = client.UpdateTestWorkflowTemplate(testworkflows.MapTestWorkflowTemplateKubeToAPI(*obj))
				ui.ExitOnError("updating test workflow template "+obj.Name+" in namespace "+obj.Namespace, err)
				ui.Success("Test workflow template updated", namespace, "/", obj.Name)
			} else {
				_, err = client.CreateTestWorkflowTemplate(testworkflows.MapTestWorkflowTemplateKubeToAPI(*obj))
				ui.ExitOnError("creating test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
				ui.Success("Test workflow template created", namespace, "/", obj.Name)
			}
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "test workflow template name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow template already exists")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "file path to get the test workflow template specification")

	return cmd
}
