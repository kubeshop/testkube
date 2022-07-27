package debuginfo

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func BuildInfo(d testkube.DebugInfo) (string, error) {
	if d.ClientVersion == "" || d.ClusterVersion == "" {
		return "", errors.New("client version and cluster version must be populated to create debug message")
	}
	t, err := template.New("debug").Parse(GetTemplate())
	if err != nil {
		return "", fmt.Errorf("cannot create template: %w", err)
	}

	var result bytes.Buffer
	err = t.Execute(&result, d)
	if err != nil {
		return "", fmt.Errorf("cannot parse template: %w", err)
	}

	return result.String(), nil
}

func GetTemplate() string {
	return `
|Property|Value|
|----|----|
|Client version|{{ .ClientVersion }}|
|Server version|{{ .ServerVersion }}|
|Cluster version|{{ .ClusterVersion }}|

### API logs
{{ range $log := .ApiLogs }}
{{ $log }}{{ end }}

### Operator logs
{{ range $log := .OperatorLogs }}
{{ $log }}{{ end }}

### Execution logs
{{ range $id, $executionLogs := .ExecutionLogs }}
Execution ID: {{ $id }}
{{ range $log := $executionLogs}}
{{ $log }}{{ end }}
{{ end }}
`
}
