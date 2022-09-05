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
			var err error

			dir := args[0]

			tests := make(map[string]map[string]client.UpsertTestOptions, 0)
			testEnvs := make(map[string]map[string]string, 0)
			testSecretEnvs := make(map[string]map[string]string, 0)

			err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				if filepath.Ext(path) != ".json" {
					return nil
				}

				if envName, testType, ok := d.DetectEnvName(path); ok {
					if _, ok := testEnvs[testType]; !ok {
						testEnvs[testType] = make(map[string]string, 0)
					}

					testEnvs[testType][envName] = path
					return nil
				}

				if secretEnvName, testType, ok := d.DetectSecretEnvName(path); ok {
					if _, ok := testEnvs[testType]; !ok {
						testSecretEnvs[testType] = make(map[string]string, 0)
					}

					testSecretEnvs[testType][secretEnvName] = path
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

			firstEntry := true
			for testType, values := range tests {
				if !firstEntry {
					fmt.Printf("\n---\n")
				} else {
					firstEntry = false
				}

				for testName, test := range values {
					if _, ok := testEnvs[testType]; ok {
						if filename, ok := testEnvs[testType][testName]; ok {
							data, err := os.ReadFile(filename)
							if err != nil {
								ui.UseStderr()
								ui.Warn(fmt.Sprintf("read variables file %s got an error: %v", filename, err))
								continue
							}

							if test.ExecutionRequest == nil {
								test.ExecutionRequest = &testkube.ExecutionRequest{}
							}

							test.ExecutionRequest.VariablesFile = string(data)
						}
					}

					if _, ok := testSecretEnvs[testType]; ok {
						if filename, ok := testSecretEnvs[testType][testName]; ok {
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

								if test.ExecutionRequest == nil {
									test.ExecutionRequest = &testkube.ExecutionRequest{}
								}

								test.ExecutionRequest.Variables = variables
							}
						}
					}

					fmt.Print(crd.ExecuteTemplate(crd.TemplateTest, test))
				}
			}

			ui.ExitOnError("getting directory content", err)
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
	path = strings.TrimSuffix(path, ".postman_collection")

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
