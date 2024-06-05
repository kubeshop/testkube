package tests

import (
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	internalcommon "github.com/kubeshop/testkube/internal/common"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMigrateTestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"tests", "t"},
		Short:   "Migrate all available tests to test workflows",
		Long:    `Migrate all available tests to test workflows from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			var name string
			if len(args) > 0 {
				name = args[0]
				test, err := client.GetTest(name)
				ui.ExitOnError("getting test in namespace "+namespace, err)

				ui.NL()
				ui.Info("Test workflow:")
				testCR := testsmapper.MapTestAPIToCR(test)
				testWorkflow := testworkflowmappers.MapTestKubeToTestWorkflowKube(testCR, "")
				b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflow{testWorkflow}, internalcommon.SerializeOptions{
					OmitCreationTimestamp: true,
					CleanMeta:             true,
					Kind:                  testworkflowsv1.Resource,
					GroupVersion:          &testworkflowsv1.GroupVersion,
				})
				ui.ExitOnError("serializing obj", err)
				ui.Info(string(b))
			} else {
				tests, err := client.ListTests("")
				ui.ExitOnError("getting all tests in namespace "+namespace, err)

				for _, test := range tests {
					testCR := testsmapper.MapTestAPIToCR(test)
					testWorkflow := testworkflowmappers.MapTestKubeToTestWorkflowKube(testCR, "")
					b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflow{testWorkflow}, internalcommon.SerializeOptions{
						OmitCreationTimestamp: true,
						CleanMeta:             true,
						Kind:                  testworkflowsv1.Resource,
						GroupVersion:          &testworkflowsv1.GroupVersion,
					})
					ui.ExitOnError("serializing obj", err)
					ui.Info(string(b))
				}
			}
		},
	}

	return cmd
}
