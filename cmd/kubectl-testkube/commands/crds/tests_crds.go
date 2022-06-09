package crds

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
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
	cmd := &cobra.Command{
		Use:   "tests-crds <manifestDirectory>",
		Short: "Generate tests CRD file based on directory",
		Long:  `Generate tests manifest based on directory (e.g. for ArgoCD sync based on tests files)`,
		Args:  validator.ManifestsDirectory,
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			dir := args[0]
			firstEntry := true

			err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				if !strings.HasSuffix(path, ".json") {
					return nil
				}

				ns, _ := cmd.Flags().GetString("namespace")
				yaml, err := GenerateCRD(ns, path)
				if err != nil {
					if !errors.Is(err, ErrTypeNotDetected) {
						return err
					}

					ui.UseStderr()
					ui.Warn(fmt.Sprintf("for file %s got an error: %v", path, err))
					return nil
				}

				if !firstEntry {
					fmt.Printf("\n---\n")
				} else {
					firstEntry = false
				}

				fmt.Print(yaml)

				return nil
			})

			ui.ExitOnError("getting directory content", err)
		},
	}

	return cmd
}

var ErrTypeNotDetected = fmt.Errorf("type not detected")

// TODO find a way to use internal objects as YAML
// GenerateCRD generates CRDs based on directory of test files
func GenerateCRD(namespace, path string) (string, error) {
	var testType string

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	// try to detect type if none passed
	d := detector.NewDefaultDetector()
	if detectedType, ok := d.Detect(client.UpsertTestOptions{Content: &testkube.TestContent{Data: string(content)}}); ok {
		ui.Debug("Detected test type", detectedType)
		testType = detectedType
	} else {
		return "", ErrTypeNotDetected
	}

	name := filepath.Base(path)
	test := client.UpsertTestOptions{
		Name:      SanitizeName(name),
		Namespace: namespace,
		Content: &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  fmt.Sprintf("%q", strings.TrimSpace(string(content))),
		},
		Type_: testType,
	}

	return crd.ExecuteTemplate(crd.TemplateTest, test)
}

func SanitizeName(path string) string {
	path = strings.TrimSuffix(path, ".json")

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
