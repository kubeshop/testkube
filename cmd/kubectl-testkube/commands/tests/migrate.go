package tests

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	printedExecutors = make(map[string]struct{})
)

func NewMigrateTestsCmd() *cobra.Command {
	var (
		migrateExecutors bool
	)

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"tests", "t"},
		Short:   "Migrate all available tests to test workflows",
		Long:    `Migrate all available tests to test workflows from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			executors, err := client.ListExecutors("")
			ui.ExitOnError("getting all executors in namespace "+namespace, err)

			executorTypes := make(map[string]testkube.ExecutorDetails)
			for _, executor := range executors {
				for _, executorType := range executor.Executor.Types {
					executorTypes[executorType] = executor
				}
			}

			if len(args) > 0 {
				test, err := client.GetTest(args[0])
				ui.ExitOnError("getting test in namespace "+namespace, err)

				templateName, expandTemplate, confugRun := printExecutor(executorTypes, namespace, test, migrateExecutors)
				common.PrintTestWorkflowCRDForTest(test, expandTemplate, templateName, confugRun)
			} else {
				tests, err := client.ListTests("")
				ui.ExitOnError("getting all tests in namespace "+namespace, err)

				for i, test := range tests {
					templateName, expandTemplate, confugRun := printExecutor(executorTypes, namespace, test, migrateExecutors)
					common.PrintTestWorkflowCRDForTest(test, expandTemplate, templateName, confugRun)
					if i != len(tests)-1 {
						fmt.Printf("\n---\n\n")
					}
				}
			}
		},
	}

	cmd.Flags().BoolVar(&migrateExecutors, "migrate-executors", true, "migrate executors for tests")

	return cmd
}

func printExecutor(executorTypes map[string]testkube.ExecutorDetails, namespace string,
	test testkube.Test, migrateExecutors bool) (string, bool, string) {
	templateName := ""
	expandTemplate := false
	confugRun := ""
	if executor, ok := executorTypes[test.Type_]; ok {
		if official, ok := common.OfficialTestWorkflowTemplates[executor.Name]; !ok {
			templateName = executor.Name
			if executor.Executor != nil {
				expandTemplate = len(executor.Executor.Command) == 0 && len(executor.Executor.Args) == 0
			}

			if _, ok = printedExecutors[templateName]; !ok && migrateExecutors {
				common.PrintTestWorkflowTemplateCRDForExecutor(executor, namespace)
				fmt.Printf("\n---\n\n")
				printedExecutors[templateName] = struct{}{}
			}
		} else {
			templateName = official.Name
			confugRun = official.ConfigRun
		}
	}

	return templateName, expandTemplate, confugRun
}
