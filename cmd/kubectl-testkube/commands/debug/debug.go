package debug

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/kubeshop/testkube/cmd/tools/commands"
)

type DebugInfo struct {
	ClientVersion     string
	ServerVersion     string
	Commit            string
	BuildBy           string
	BuildDate         string
	ClusterVersion    string
	APILogs           []string
	OperatorLogs      []string
	LastExecutionLogs map[string][]string
}

func GetClientVersion() string {
	return commands.Version
}

func GetClusterVersion() error {
	return nil
}

func GetAPILogs() error {
	return nil
}

func GetOperatorLogs() error {
	return nil
}

func GetLastExecutionLogs() error {
	return nil
}

func BuildInfo(d DebugInfo) (string, error) {
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
|Commit|{{ .Commit }}|
|Build by|{{ .BuildBy }}|
|Build date|{{ .BuildDate }}|
|Cluster version|{{ .ClusterVersion }}|

### API logs
{{ range $log := .APILogs }}
{{ $log }}{{ end }}

### Operator logs
{{ range $log := .OperatorLogs }}
{{ $log }}{{ end }}

### Execution logs
{{ range $id, $executionLogs := .LastExecutionLogs }}
Execution ID: {{ $id }}
{{ range $log := $executionLogs}}
{{ $log }}{{ end }}
{{ end }}
`
}
