package debug

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
)

type DebugInfo struct {
	ClientVersion     string
	ClusterVersion    string
	APILogs           []string
	OperatorLogs      []string
	LastExecutionLogs map[string][]string
}

func GetClientVersion() error {
	return nil
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
|Property|Version|
|----|----|
|Client|{{ .ClientVersion }}|
|Kubernetes cluster|{{ .ClusterVersion }}|

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
