package scripts

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/test/script/detector"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCRDScriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crd",
		Short: "Generate scripts CRD file based on directory",
		Long:  `Generate scripts manifest based on directory (e.g. for ArgoCD sync based on scripts files)`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			if len(args) == 0 {
				ui.Failf("Please pass directory")
			}

			dir := args[0]
			firstEntry := true

			err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
				if !firstEntry {
					fmt.Printf("\n\n---\n\n")
				}
				firstEntry = false

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
	if detectedType, ok := d.Detect(client.UpsertScriptOptions{Content: string(content)}); ok {
		ui.Debug("Detected test script type", detectedType)
		scriptType = detectedType
	} else {
		return "", ErrTypeNotDetected
	}

	t := template.Must(template.New("script").Parse(tpl))
	b := bytes.NewBuffer([]byte{})
	err = t.Execute(b, Script{
		Name:      SanitizePath(path),
		Namespace: namespace,
		Content:   fmt.Sprintf("%q", string(content)),
		Type:      scriptType,
	})

	return b.String(), err
}

func SanitizePath(path string) string {
	name := strings.Replace(path, "/", "-", -1)
	name = strings.Replace(name, "_", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	name = strings.TrimSuffix(name, ".json")

	return name
}
