package templates

import (
	"bytes"
	"embed"
	"io/fs"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

// Templates embeds all our official, build-in templates.
//
//go:embed *
var Templates embed.FS

func ParseAllOfficialTemplates() (officials []testkube.TestWorkflowTemplate, skipped []string, err error) {
	entries, err := Templates.ReadDir(".")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		t, err := parseOfficialTemplate(entry)
		if err != nil {
			skipped = append(skipped, entry.Name())
			continue
		}
		officials = append(officials, *t)
	}

	return
}

func parseOfficialTemplate(e fs.DirEntry) (*testkube.TestWorkflowTemplate, error) {
	file, err := Templates.ReadFile(e.Name())
	if err != nil {
		return nil, err
	}

	var template v1.TestWorkflowTemplate
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(file), len(file))
	if err := decoder.Decode(&template); err != nil {
		return nil, err
	}

	result := testworkflows.MapTemplateKubeToAPI(&template)
	return result, nil
}
