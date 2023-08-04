package crds

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/contrib/executor/postman/pkg/postman"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils"
)

var (
	uri   string
	flags tests.CreateCommonFlags
)

// NewCRDTestsCmd is command to generate test CRDs
func NewCRDTestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tests-crds <manifestDirectory>",
		Aliases: []string{"test-crd", "test-crds", "tests-crd"},
		Short:   "Generate tests CRD file based on directory",
		Long:    `Generate tests manifest based on directory (e.g. for ArgoCD sync based on tests files)`,
		Args:    validator.ManifestsDirectory,
		Run: func(cmd *cobra.Command, args []string) {
			var (
				testContentType, file, name string
				crdOnly                     bool
			)
			cmd.Flags().StringVar(&testContentType, "test-content-type", "string", "")
			cmd.Flags().BoolVar(&crdOnly, "crd-only", true, "generate only CRD")
			cmd.Flags().StringVar(&uri, "uri", "", "")
			cmd.Flags().StringVar(&file, "file", "", "")
			cmd.Flags().StringVar(&name, "name", "", "")
			if flags.ExecutorType == postman.PostmanCollectionType {
				processPostmanFiles(cmd, args)
			} else {
				dir := args[0]
				err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
					if err != nil {
						return nil
					}

					if info.IsDir() {
						return nil
					}
					cmd.Flags().Set("file", path)
					cmd.Flags().Set("name", utils.SanitizeName(filepath.Base(path)))
					options, err := tests.NewUpsertTestOptionsFromFlags(cmd)
					if err != nil {
						fmt.Println("# getting test options for file", path, err.Error())
						fmt.Println("---")
						return nil
					}
					(*testkube.TestUpsertRequest)(&options).QuoteTestTextFields()
					data, err := crd.ExecuteTemplate(crd.TemplateTest, options)
					if err != nil {
						fmt.Println("# executing crd template for file", err.Error())
						fmt.Println("---")
						return nil
					}

					fmt.Println(data)
					fmt.Println("---")
					return nil
				})

				ui.ExitOnError("getting directory content", err)
			}
		},
	}

	tests.AddCreateFlags(cmd, &flags)
	return cmd
}

// ErrTypeNotDetected is not detcted test type error
var ErrTypeNotDetected = fmt.Errorf("type not detected")

// processPostmanFiles processes postman files
func processPostmanFiles(cmd *cobra.Command, args []string) error {
	detector := postman.Detector{}
	dir := args[0]

	detectedTests := make(map[string]client.UpsertTestOptions, 0)
	testEnvs := make(map[string]map[string]string, 0)
	testSecretEnvs := make(map[string]map[string]string, 0)

	var preRunScript, postRunScript []byte

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if testName, envName, ok := detector.IsEnvName(path); ok {

			if _, ok := testEnvs[envName]; !ok {
				testEnvs[envName] = make(map[string]string, 0)
			}

			testEnvs[envName][testName] = path
			return nil
		}

		if secretTestName, secretEnvName, ok := detector.IsSecretEnvName(path); ok {

			if _, ok := testSecretEnvs[secretEnvName]; !ok {
				testSecretEnvs[secretEnvName] = make(map[string]string, 0)
			}

			testSecretEnvs[secretEnvName][secretTestName] = path
			return nil
		}

		cmd.Flags().Set("file", path)
		test, err := tests.NewUpsertTestOptionsFromFlags(cmd)
		if err != nil {
			ui.Warn(fmt.Sprintf("generate test for file %s got an error: %v", path, err))
			ui.UseStderr()
			return err
		}

		test.Name = utils.SanitizeName(filepath.Base(path))

		testName, ok := detector.IsTestName(path)
		if !ok {
			testName = test.Name
		}

		preRunScriptBody := string(preRunScript)
		if preRunScriptBody != "" {
			preRunScriptBody = fmt.Sprintf("%q", strings.TrimSpace(preRunScriptBody))
		}

		postRunScriptBody := string(postRunScript)
		if postRunScriptBody != "" {
			postRunScriptBody = fmt.Sprintf("%q", strings.TrimSpace(postRunScriptBody))
		}

		for key, value := range flags.Envs {
			if value != "" {
				flags.Envs[key] = fmt.Sprintf("%q", value)
			}
		}

		vars, err := common.CreateVariables(cmd, true)
		if err != nil {
			return err
		}

		for name, variable := range vars {
			if variable.Value != "" {
				variable.Value = fmt.Sprintf("%q", variable.Value)
				vars[name] = variable
			}
		}

		test.ExecutionRequest = &testkube.ExecutionRequest{
			Command:       flags.Command,
			Args:          flags.ExecutorArgs,
			ArgsMode:      flags.ArgsMode,
			Envs:          flags.Envs,
			Variables:     vars,
			PreRunScript:  preRunScriptBody,
			PostRunScript: postRunScriptBody,
		}
		detectedTests[testName] = test
		return nil
	})

	ui.ExitOnError("getting directory content", err)

	generateCRDs(addEnvToTests(detectedTests, testEnvs, testSecretEnvs))
	return nil

}

// addEnvToTest adds env files to tests
func addEnvToTests(tests map[string]client.UpsertTestOptions,
	testEnvs, testSecretEnvs map[string]map[string]string) (envTests []client.UpsertTestOptions) {
	detector := postman.Detector{}
	for testName, test := range tests {
		testMap := map[string]client.UpsertTestOptions{}
		for envName := range testEnvs {
			if filename, ok := testEnvs[envName][testName]; ok {
				data, err := os.ReadFile(filename)
				if err != nil {
					ui.UseStderr()
					ui.Warn(fmt.Sprintf("read variables file %s got an error: %v", filename, err))
					continue
				}

				envTest := test
				envTest.Name = utils.SanitizeName(envTest.Name + "-" + envName)
				if test.ExecutionRequest != nil {
					envTest.ExecutionRequest = &testkube.ExecutionRequest{}
					*envTest.ExecutionRequest = *test.ExecutionRequest
				}

				if envTest.ExecutionRequest == nil {
					envTest.ExecutionRequest = &testkube.ExecutionRequest{}
				}

				envTest.ExecutionRequest.VariablesFile = fmt.Sprintf("%q", strings.TrimSpace(string(data)))
				testMap[envTest.Name] = envTest
			}
		}

		for secretEnvName := range testSecretEnvs {
			if filename, ok := testSecretEnvs[secretEnvName][testName]; ok {
				data, err := os.ReadFile(filename)
				if err != nil {
					ui.UseStderr()
					ui.Warn(fmt.Sprintf("read secret variables file %s got an error: %v", filename, err))
					continue
				}

				variables, err := detector.GetSecretVariables(string(data))
				if err != nil {
					ui.UseStderr()
					ui.Warn(fmt.Sprintf("parse secret file %s got an error: %v", filename, err))
					continue
				}

				secretEnvTest := test
				secretEnvTest.Name = utils.SanitizeName(secretEnvTest.Name + "-" + secretEnvName)
				if test.ExecutionRequest != nil {
					secretEnvTest.ExecutionRequest = &testkube.ExecutionRequest{}
					*secretEnvTest.ExecutionRequest = *test.ExecutionRequest
				}

				if envTest, ok := testMap[secretEnvTest.Name]; ok {
					secretEnvTest = envTest
				}

				if secretEnvTest.ExecutionRequest == nil {
					secretEnvTest.ExecutionRequest = &testkube.ExecutionRequest{}
				}

				for key, value := range variables {
					if value.Value != "" {
						value.Value = fmt.Sprintf("%q", value.Value)
						variables[key] = value
					}
				}

				secretEnvTest.ExecutionRequest.Variables = variables
				testMap[secretEnvTest.Name] = secretEnvTest
			}
		}

		if len(testMap) == 0 {
			testMap[test.Name] = test
		}

		for _, envTest := range testMap {
			envTests = append(envTests, envTest)
		}
	}

	return envTests
}

// generateCRDs generates CRDs for tests
func generateCRDs(envTests []client.UpsertTestOptions) {
	firstEntry := true
	for _, test := range envTests {
		if !firstEntry {
			fmt.Printf("\n---\n")
		} else {
			firstEntry = false
		}

		yaml, err := crd.ExecuteTemplate(crd.TemplateTest, test)
		ui.ExitOnError("executing test template", err)

		fmt.Print(yaml)
	}
}
