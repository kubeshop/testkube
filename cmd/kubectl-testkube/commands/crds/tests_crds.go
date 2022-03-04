package crds

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
					return err
				}

				if !firstEntry {
					fmt.Printf("\n---\n")
				}
				firstEntry = false

				fmt.Print(yaml)

				return nil
			})

			ui.ExitOnError("getting directory content", err)

		},
	}

	return cmd
}

var ErrTypeNotDetected = fmt.Errorf("type not detected")

type Test struct {
	Name      string
	Namespace string
	Content   string
	Type      string
}

// TODO find a way to use internal objects as YAML
// GenerateCRD generates CRDs based on directory of test files
func GenerateCRD(namespace, path string) (string, error) {
	var testType string

	tpl := `apiVersion: tests.testkube.io/v2
kind: Test
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  content: 
    type: string
    data: {{ .Content }}
  type: {{ .Type }}
`

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

	t := template.Must(template.New("test").Parse(tpl))
	b := bytes.NewBuffer([]byte{})
	err = t.Execute(b, Test{
		Name:      SanitizeName(name),
		Namespace: namespace,
		Content:   fmt.Sprintf("%q", strings.TrimSpace(string(content))),
		Type:      testType,
	})

	return b.String(), err
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
