package testsuites

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	printedExecutors = make(map[string]struct{})
	printedTests     = make(map[string]struct{})
)

func NewMigrateTestSuitesCmd() *cobra.Command {
	var (
		migrateExecutors bool
		migrateTests     bool
	)

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"testsuites", "ts"},
		Short:   "Migrate all available test suites to test workflows",
		Long:    `Migrate all available test suites to test workflows from given namespace - if no namespace given "testkube" namespace is used`,
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
				testSuite, err := client.GetTestSuite(args[0])
				ui.ExitOnError("getting test suite in namespace "+namespace, err)

				if migrateTests {
					printTestSuiteTests(client, namespace, executorTypes, testSuite, migrateExecutors)
				}

				common.PrintTestWorkflowCRDForTestSuite(testSuite)
			} else {
				testSuites, err := client.ListTestSuites("")
				ui.ExitOnError("getting all test suites in namespace "+namespace, err)

				for i, testSuite := range testSuites {
					if migrateTests {
						printTestSuiteTests(client, namespace, executorTypes, testSuite, migrateExecutors)
					}

					common.PrintTestWorkflowCRDForTestSuite(testSuite)
					if i != len(testSuites)-1 {
						fmt.Printf("\n---\n\n")
					}
				}
			}
		},
	}

	cmd.Flags().BoolVar(&migrateTests, "migrate-tests", false, "migrate tests for test suites")
	cmd.Flags().BoolVar(&migrateExecutors, "migrate-executors", true, "migrate executors for tests")

	return cmd
}

func printTestSuiteTests(client client.Client, namespace string, executorTypes map[string]testkube.ExecutorDetails,
	testSuite testkube.TestSuite, migrateExecutors bool) {
	testNames := testSuite.GetTestNames()
	for _, testName := range testNames {
		test, err := client.GetTest(testName)
		ui.ExitOnError("getting test in namespace "+namespace, err)

		templateName := ""
		expandTemplate := false
		configRun := ""
		if executor, ok := executorTypes[test.Type_]; ok {
			templateName = executor.Name
			if official, ok := common.OfficialTestWorkflowTemplates[templateName]; !ok {
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
				configRun = official.ConfigRun
			}
		}

		if _, ok := printedTests[testName]; !ok {
			common.PrintTestWorkflowCRDForTest(test, expandTemplate, templateName, configRun)
			fmt.Printf("\n---\n\n")
			printedTests[testName] = struct{}{}
		}
	}
}
