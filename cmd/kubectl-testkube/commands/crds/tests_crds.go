package crds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/test/detector"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCRDTestsCmd() *cobra.Command {
	// try to detect type if none passed
	d := detector.NewDefaultDetector()

	cmd := &cobra.Command{
		Use:   "tests-crds <manifestDirectory>",
		Short: "Generate tests CRD file based on directory",
		Long:  `Generate tests manifest based on directory (e.g. for ArgoCD sync based on tests files)`,
		Args:  validator.ManifestsDirectory,
		Run: func(cmd *cobra.Command, args []string) {
			dir := args[0]

			tests := make(map[string]map[string]client.UpsertTestOptions, 0)
			testEnvs := make(map[string]map[string]map[string]string, 0)
			testSecretEnvs := make(map[string]map[string]map[string]string, 0)

			err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				if filepath.Ext(path) != ".json" {
					return nil
				}

				if testName, envName, testType, ok := d.DetectEnvName(path); ok {
					if _, ok := testEnvs[testType]; !ok {
						testEnvs[testType] = make(map[string]map[string]string, 0)
					}

					if _, ok := testEnvs[testType][envName]; !ok {
						testEnvs[testType][envName] = make(map[string]string, 0)
					}

					testEnvs[testType][envName][testName] = path
					return nil
				}

				if secretTestName, secretEnvName, testType, ok := d.DetectSecretEnvName(path); ok {
					if _, ok := testSecretEnvs[testType]; !ok {
						testSecretEnvs[testType] = make(map[string]map[string]string, 0)
					}

					if _, ok := testSecretEnvs[testType][secretEnvName]; !ok {
						testSecretEnvs[testType][secretEnvName] = make(map[string]string, 0)
					}

					testSecretEnvs[testType][secretEnvName][secretTestName] = path
					return nil
				}

				ns, _ := cmd.Flags().GetString("namespace")
				test, err := GenerateTest(ns, path)
				if err != nil {
					if !errors.Is(err, ErrTypeNotDetected) {
						return err
					}

					ui.UseStderr()
					ui.Warn(fmt.Sprintf("generate test for file %s got an error: %v", path, err))
					return nil
				}

				testName, testType, ok := d.DetectTestName(path)
				if !ok {
					testName = test.Name
					testType = test.Type_
				}

				if _, ok := tests[testType]; !ok {
					tests[testType] = make(map[string]client.UpsertTestOptions, 0)
				}

				tests[testType][testName] = *test
				return nil
			})

			ui.ExitOnError("getting directory content", err)

			var envTests []client.UpsertTestOptions
			for testType, values := range tests {
				for testName, test := range values {
					testMap := map[string]client.UpsertTestOptions{}
					for envName := range testEnvs[testType] {
						if filename, ok := testEnvs[testType][envName][testName]; ok {
							data, err := os.ReadFile(filename)
							if err != nil {
								ui.UseStderr()
								ui.Warn(fmt.Sprintf("read variables file %s got an error: %v", filename, err))
								continue
							}

							envTest := test
							envTest.Name = SanitizeName(envTest.Name + "-" + envName)
							envTest.ExecutionRequest = &testkube.ExecutionRequest{
								VariablesFile: fmt.Sprintf("%q", strings.TrimSpace(string(data))),
							}

							testMap[envTest.Name] = envTest
						}
					}

					for secretEnvName := range testSecretEnvs[testType] {
						if filename, ok := testSecretEnvs[testType][secretEnvName][testName]; ok {
							data, err := os.ReadFile(filename)
							if err != nil {
								ui.UseStderr()
								ui.Warn(fmt.Sprintf("read secret variables file %s got an error: %v", filename, err))
								continue
							}

							if adapter := d.GetAdapter(testType); adapter != nil {
								variables, err := adapter.GetSecretVariables(string(data))
								if err != nil {
									ui.UseStderr()
									ui.Warn(fmt.Sprintf("parse secret file %s got an error: %v", filename, err))
									continue
								}

								secretEnvTest := test
								secretEnvTest.Name = SanitizeName(secretEnvTest.Name + "-" + secretEnvName)
								if envTest, ok := testMap[secretEnvTest.Name]; ok {
									secretEnvTest = envTest
								}

								if secretEnvTest.ExecutionRequest == nil {
									secretEnvTest.ExecutionRequest = &testkube.ExecutionRequest{}
								}

								secretEnvTest.ExecutionRequest.Variables = variables
								testMap[secretEnvTest.Name] = secretEnvTest
							}
						}
					}

					if len(testMap) == 0 {
						testMap[test.Name] = test
					}

					for _, envTest := range testMap {
						envTests = append(envTests, envTest)
					}
				}
			}

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
		},
	}

	return cmd
}

var ErrTypeNotDetected = fmt.Errorf("type not detected")

// TODO find a way to use internal objects as YAML
// GenerateTest generates Test based on directory of test files
func GenerateTest(namespace, path string) (*client.UpsertTestOptions, error) {
	var testType string

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// try to detect type if none passed
	d := detector.NewDefaultDetector()
	if detectedType, ok := d.Detect(client.UpsertTestOptions{Content: &testkube.TestContent{Data: string(content)}}); ok {
		ui.Debug("Detected test type", detectedType)
		testType = detectedType
	} else {
		return nil, ErrTypeNotDetected
	}

	name := filepath.Base(path)
	test := &client.UpsertTestOptions{
		Name:      SanitizeName(name),
		Namespace: namespace,
		Content: &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  fmt.Sprintf("%q", strings.TrimSpace(string(content))),
		},
		Type_: testType,
	}

	return test, nil
}

func SanitizeName(path string) string {
	path = strings.TrimSuffix(path, filepath.Ext(path))

	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	path = reg.ReplaceAllString(path, "-")
	path = strings.TrimLeft(path, "-")
	path = strings.TrimRight(path, "-")
	path = strings.ToLower(path)

	if len(path) > 63 {
		return path[:63]
	}

	return path
}
