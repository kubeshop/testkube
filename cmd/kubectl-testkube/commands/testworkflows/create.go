package testworkflows

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

func NewCreateTestWorkflowCmd() *cobra.Command {
	var (
		name     string
		filePath string
		update   bool
	)

	cmd := &cobra.Command{
		Use:     "testworkflow",
		Aliases: []string{"testworkflows", "tw"},
		Args:    cobra.MaximumNArgs(0),
		Short:   "Create test workflow",

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

			obj := new(testworkflowsv1.TestWorkflow)
			err = common2.DeserializeCRD(obj, bytes)
			ui.ExitOnError("deserializing input", err)
			if obj.Kind != "" && obj.Kind != "TestWorkflow" {
				ui.Failf("Only TestWorkflow objects are accepted. Received: %s", obj.Kind)
			}
			common2.AppendTypeMeta("TestWorkflow", testworkflowsv1.GroupVersion, obj)
			obj.Namespace = namespace
			if name != "" {
				obj.Name = name
			}

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			workflow, _ := client.GetTestWorkflow(obj.Name)
			if workflow.Name != "" {
				if !update {
					ui.Failf("Test workflow with name '%s' already exists in namespace %s, use --update flag for upsert", obj.Name, namespace)
				}
				_, err = client.UpdateTestWorkflow(testworkflows.MapTestWorkflowKubeToAPI(*obj))
				ui.ExitOnError("updating test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
				ui.Success("Test workflow updated", namespace, "/", obj.Name)
			} else {
				_, err = client.CreateTestWorkflow(testworkflows.MapTestWorkflowKubeToAPI(*obj))
				ui.ExitOnError("creating test workflow "+obj.Name+" in namespace "+obj.Namespace, err)
				ui.Success("Test workflow created", namespace, "/", obj.Name)
			}
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "test workflow name")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test workflow already exists")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "file path to get the test workflow specification")

	return cmd
}
