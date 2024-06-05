package testsuites

import (
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	internalcommon "github.com/kubeshop/testkube/internal/common"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMigrateTestSuitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "testsuite <testName>",
		Aliases: []string{"testsuites", "ts"},
		Short:   "Migrate all available test suites to test workflows",
		Long:    `Migrate all available test suites to test workflows from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			var name string
			if len(args) > 0 {
				name = args[0]
				testSuite, err := client.GetTestSuite(name)
				ui.ExitOnError("getting test suite in namespace "+namespace, err)

				ui.NL()
				ui.Info("Test workflow:")
				testSuiteCR, err := testsuitesmapper.MapAPIToCR(testSuite)
				ui.ExitOnError("mapping obj", err)

				testWorkflow := testworkflowmappers.MapTestSuiteKubeToTestWorkflowKube(testSuiteCR, "")
				b, err := internalcommon.SerializeCRDs([]testworkflowsv1.TestWorkflow{testWorkflow}, internalcommon.SerializeOptions{
					OmitCreationTimestamp: true,
					CleanMeta:             true,
					Kind:                  testworkflowsv1.Resource,
					GroupVersion:          &testworkflowsv1.GroupVersion,
				})
				ui.ExitOnError("serializing obj", err)
				ui.Info(string(b))
			} else {
				testSuites, err := client.ListTestSuites("")
				ui.ExitOnError("getting all test suites in namespace "+namespace, err)

				for _, testSuite := range testSuites {
					testSuiteCR, err := testsuitesmapper.MapAPIToCR(testSuite)
					ui.ExitOnError("mapping obj", err)

					testWorkflow := testworkflowmappers.MapTestSuiteKubeToTestWorkflowKube(testSuiteCR, "")
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
