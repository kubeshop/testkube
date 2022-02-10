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
	"github.com/kubeshop/testkube/pkg/test/script/detector"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCRDScriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scripts <manifestDirectory>",
		Short: "Generate scripts CRD file based on directory",
		Long:  `Generate scripts manifest based on directory (e.g. for ArgoCD sync based on scripts files)`,
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

type Script struct {
	Name      string
	Namespace string
	Content   string
	Type      string
}

// TODO find a way to use internal objects as YAML
// GenerateCRD generates CRDs based on directory of test files
func GenerateCRD(namespace, path string) (string, error) {
	var scriptType string

	tpl := `apiVersion: tests.testkube.io/v1
kind: Script
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  content: {{ .Content }}
  type: {{ .Type }}
`

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	// try to detect type if none passed
	d := detector.NewDefaultDetector()
	if detectedType, ok := d.Detect(client.UpsertScriptOptions{Content: &testkube.TestContent{Data: string(content)}}); ok {
		ui.Debug("Detected test script type", detectedType)
		scriptType = detectedType
	} else {
		return "", ErrTypeNotDetected
	}

	name := filepath.Base(path)

	t := template.Must(template.New("script").Parse(tpl))
	b := bytes.NewBuffer([]byte{})
	err = t.Execute(b, Script{
		Name:      SanitizeName(name),
		Namespace: namespace,
		Content:   fmt.Sprintf("%q", strings.TrimSpace(string(content))),
		Type:      scriptType,
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
