package debuginfo

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateTicketCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-ticket",
		Short: "Create bug ticket",
		Long:  "Create an issue of type bug in the Testkube repository",
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)
			debug, err := client.GetDebugInfo()
			ui.ExitOnError("get debug info", err)
			info, err := client.GetServerInfo()
			ui.ExitOnError("get server info", err)
			debug.ClientVersion = common.Version
			debug.ServerVersion = info.Version

			mdInfo, err := BuildInfo(debug)
			ui.ExitOnError("get markdown", err)
			ui.Info(mdInfo)
		},
	}
}

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
